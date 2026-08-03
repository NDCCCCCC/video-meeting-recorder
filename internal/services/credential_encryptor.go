package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/utils"
)

// CredentialEncryptor 是 Phase 18 引入的凭据静态加密抽象。
//
// 设计要点：
//   - 持有当前版本的 16 字节 SM4 密钥 + 可选的 previous 密钥（轮换过渡期）
//   - 提供 Encrypt / Decrypt 两个对外 API（envelope: SM4:<version>:<base64>）
//   - 提供 MigratePlaintextToGCM 把历史明文 / 历史 base64-stub 数据一次性升级到 envelope
//   - 提供 InvariantScan 检测：plaintext / unknown-version / undecryptable 密文
//   - 提供 RotateIfNeeded 把 previous 版本 envelope 改写成 current 版本
//
// 关键不变量：
//   - 任何一次成功的 InvariantScan 必须保证 input_configs.password / stream_password
//     均为合法 envelope（match current or previous version 且能解密）
//   - 任何一次成功的 InvariantScan 必须保证 system_settings['auth.ad'] JSON 不含 password
//   - 任何一次成功的 InvariantScan 必须保证 system_settings['auth.ad.password']
//     为合法 envelope 且能解密
type CredentialEncryptor struct {
	logger *zap.Logger

	currentVersion string
	currentKey     []byte

	previousVersion string // "" 表示无 previous 密钥（单一版本场景）
	previousKey     []byte
}

// NewCredentialEncryptor 根据配置构造加密器。
//   - currentVersion / currentSecret 必须非空（与 ValidateCredentialSM4Config 等价约束）
//   - previousVersion / previousSecret 配对存在时启用轮换；否则 previous 为零值
//   - 内部把 secret 通过 utils.DeriveSM4Key 归一化为 16 字节密钥
func NewCredentialEncryptor(currentVersion, currentSecret, previousVersion, previousSecret string, logger *zap.Logger) (*CredentialEncryptor, error) {
	// Phase 19 D4: 配置验证错误用 BusinessError(CodeInvalidInput) 包装，
	// 让 handleStart-up 错误能用统一 mapping 落到 400 而非 500。
	if currentVersion == "" {
		return nil, apperrors.NewBusinessError(apperrors.CodeInvalidInput, "currentVersion 不能为空", nil)
	}
	if len(currentSecret) < 32 {
		return nil, apperrors.NewBusinessError(apperrors.CodeInvalidInput, fmt.Sprintf("currentSecret 必须 ≥ 32 字符（实际 %d）", len(currentSecret)), nil)
	}
	enc := &CredentialEncryptor{
		logger:         logger,
		currentVersion: currentVersion,
		currentKey:     utils.DeriveSM4Key(currentSecret),
	}
	if (previousVersion == "") != (previousSecret == "") {
		return nil, apperrors.NewBusinessError(apperrors.CodeInvalidInput, "previousVersion 与 previousSecret 必须同时存在或同时缺失", nil)
	}
	if previousVersion != "" {
		if previousVersion == currentVersion {
			return nil, apperrors.NewBusinessError(apperrors.CodeInvalidInput, "previousVersion 必须不等于 currentVersion", nil)
		}
		if len(previousSecret) < 32 {
			return nil, apperrors.NewBusinessError(apperrors.CodeInvalidInput, fmt.Sprintf("previousSecret 必须 ≥ 32 字符（实际 %d）", len(previousSecret)), nil)
		}
		enc.previousVersion = previousVersion
		enc.previousKey = utils.DeriveSM4Key(previousSecret)
	}
	return enc, nil
}

// CurrentVersion 返回当前 envelope 版本。
func (e *CredentialEncryptor) CurrentVersion() string { return e.currentVersion }

// PreviousVersion 返回 previous envelope 版本（空字符串表示无 previous）。
func (e *CredentialEncryptor) PreviousVersion() string { return e.previousVersion }

// HasPrevious 报告是否启用 previous 密钥（轮换过渡期）。
func (e *CredentialEncryptor) HasPrevious() bool { return e.previousVersion != "" }

// Encrypt 把明文加密为当前版本 envelope。空字符串视为"无凭据"，返回空串（不加密）。
func (e *CredentialEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	ct, err := utils.EncryptGCM(e.currentKey, []byte(plaintext))
	if err != nil {
		return "", fmt.Errorf("CredentialEncryptor.Encrypt 加密失败: %w", err)
	}
	return utils.EncodeCredentialEnvelope(e.currentVersion, ct)
}

// Decrypt 解密 envelope。明文空字符串则直接返回空（无凭据场景）。
//
// 解密失败时返回 error —— 永远不静默跳过。调用方应据此把行视为可疑并审计。
func (e *CredentialEncryptor) Decrypt(envelope string) (string, error) {
	if envelope == "" {
		return "", nil
	}
	version, payload, err := utils.ParseCredentialEnvelope(envelope)
	if err != nil {
		return "", fmt.Errorf("envelope 解析失败: %w", err)
	}
	switch version {
	case e.currentVersion:
		pt, err := utils.DecryptGCM(e.currentKey, payload)
		if err != nil {
			return "", fmt.Errorf("current 版本 %s 解密失败: %w", version, err)
		}
		return string(pt), nil
	case e.previousVersion:
		if e.previousKey == nil {
			return "", fmt.Errorf("envelope 是 previous 版本 %s，但本进程未配置 previous 密钥", version)
		}
		pt, err := utils.DecryptGCM(e.previousKey, payload)
		if err != nil {
			return "", fmt.Errorf("previous 版本 %s 解密失败: %w", version, err)
		}
		return string(pt), nil
	default:
		return "", fmt.Errorf("envelope version=%s 既不是 current=%s 也不是 previous=%s",
			version, e.currentVersion, e.previousVersion)
	}
}

// LooksLikeEnvelope 报告字符串是否形如 "SM4:<version>:<base64>"。
// 注意：仅做结构判断，不解密、不验证版本是否已知。
func (e *CredentialEncryptor) LooksLikeEnvelope(s string) bool {
	return utils.IsEncryptedPassword(s) && strings.Count(s, ":") >= 2
}

// ---------------------------------------------------------------------------
// 启动期数据迁移 / 一致性扫描
// ---------------------------------------------------------------------------

// migrateTarget 是单行的迁移状态。
type migrateTarget struct {
	id     uint
	column string // "password" 或 "stream_password"
	raw    string // 数据库当前值
}

// MigratePlaintextToGCM 在一个事务内把所有 input_configs.password / stream_password
// 中"非合法 envelope"的明文 / 历史 base64 升级到当前版本 envelope。
//
// 升级规则（每行独立判定）：
//   - 当前值为空 → 跳过
//   - 当前值已经是 envelope（SM4:<version>:<base64> 且能解析） → 跳过（已迁移）
//   - 当前值是 base64 能解码出合法 UTF-8 → 视为历史明文 base64-stub，base64-decode → 加密
//   - 其他 → 视为原始明文，直接加密
//
// Unscoped 扫描确保 soft-deleted 行也被处理（Phase 18 范围包含软删除）。
func (e *CredentialEncryptor) MigratePlaintextToGCM(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 用 Unscoped 把软删除行也纳入迁移
		var rows []models.InputConfig
		if err := tx.Unscoped().Find(&rows).Error; err != nil {
			return fmt.Errorf("扫描 input_configs 失败: %w", err)
		}

		var targets []migrateTarget
		for _, row := range rows {
			if row.Password != "" {
				if !e.needsMigration(row.Password) {
					continue
				}
				targets = append(targets, migrateTarget{id: row.ID, column: "password", raw: row.Password})
			}
			if row.StreamPassword != "" {
				if !e.needsMigration(row.StreamPassword) {
					continue
				}
				targets = append(targets, migrateTarget{id: row.ID, column: "stream_password", raw: row.StreamPassword})
			}
		}

		e.logger.Info("凭据迁移: 待升级行",
			zap.Int("total_rows", len(rows)),
			zap.Int("migrate_targets", len(targets)),
		)

		for _, tgt := range targets {
			plaintext, err := e.decodeLegacyValue(tgt.raw)
			if err != nil {
				return fmt.Errorf("迁移 row=%d column=%s: %w", tgt.id, tgt.column, err)
			}
			envelope, err := e.Encrypt(plaintext)
			if err != nil {
				return fmt.Errorf("迁移 row=%d column=%s 加密失败: %w", tgt.id, tgt.column, err)
			}
			if err := tx.Unscoped().Model(&models.InputConfig{}).
				Where("id = ?", tgt.id).
				Update(tgt.column, envelope).Error; err != nil {
				return fmt.Errorf("迁移 row=%d column=%s 写库失败: %w", tgt.id, tgt.column, err)
			}
		}

		// system_settings['auth.ad.password'] —— 历史值是 base64(plaintext) 或明文
		var pwdSetting models.SystemSetting
		if err := tx.Unscoped().Where("`key` = ?", "auth.ad.password").First(&pwdSetting).Error; err == nil {
			if pwdSetting.Value != "" && e.needsMigration(pwdSetting.Value) {
				plaintext, err := e.decodeLegacyValue(pwdSetting.Value)
				if err != nil {
					return fmt.Errorf("迁移 auth.ad.password: %w", err)
				}
				envelope, err := e.Encrypt(plaintext)
				if err != nil {
					return fmt.Errorf("迁移 auth.ad.password 加密失败: %w", err)
				}
				if err := tx.Unscoped().Model(&models.SystemSetting{}).
					Where("id = ?", pwdSetting.ID).
					Update("value", envelope).Error; err != nil {
					return fmt.Errorf("迁移 auth.ad.password 写库失败: %w", err)
				}
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("读取 auth.ad.password 失败: %w", err)
		}

		// system_settings['auth.ad'] —— JSON 内的 password 字段必须剔除
		var adSetting models.SystemSetting
		if err := tx.Unscoped().Where("`key` = ?", "auth.ad").First(&adSetting).Error; err == nil {
			cleaned, modified, err := e.stripPasswordFromADJSON(adSetting.Value)
			if err != nil {
				return fmt.Errorf("处理 auth.ad JSON 失败: %w", err)
			}
			if modified {
				if err := tx.Unscoped().Model(&models.SystemSetting{}).
					Where("id = ?", adSetting.ID).
					Update("value", cleaned).Error; err != nil {
					return fmt.Errorf("写回 auth.ad JSON 失败: %w", err)
				}
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("读取 auth.ad 失败: %w", err)
		}

		return nil
	})
}

// needsMigration 报告 value 是否需要被升级（不是 current/previous 版本 envelope）。
func (e *CredentialEncryptor) needsMigration(value string) bool {
	if value == "" {
		return false
	}
	version, _, err := utils.ParseCredentialEnvelope(value)
	if err != nil {
		return true // 非 envelope → 视为待迁移
	}
	return version != e.currentVersion && version != e.previousVersion
}

// decodeLegacyValue 兼容两种历史格式：
//   - 纯明文（任意 UTF-8 字符串）
//   - base64(plaintext) 旧 stub（config_service.go 旧 encryptPassword 写入的）
//
// 判定优先级：先尝试 base64 解码 + 检查是否可解析为 UTF-8 + 长度看起来像密码；
// 失败则原样当明文处理。
func (e *CredentialEncryptor) decodeLegacyValue(raw string) (string, error) {
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
		// 严格启发式：base64 解码后非空 且 全是合法 UTF-8 字节
		if len(decoded) > 0 && isLikelyUTF8Password(decoded) {
			return string(decoded), nil
		}
	}
	// fallback: 原值就是明文
	return raw, nil
}

// isLikelyUTF8Password 启发式判断字节序列是否像"密码"。避免把偶然 base64 通过的
// 长字符串误判成 base64（密码一般较短且都是 ASCII 可打印字符）。
func isLikelyUTF8Password(b []byte) bool {
	if len(b) == 0 || len(b) > 256 {
		return false
	}
	for _, c := range b {
		// 只接受可打印 ASCII + 常见 UTF-8 多字节；遇到控制字符 / NUL 即失败
		if c == 0 {
			return false
		}
		if c < 0x20 && c != '\n' && c != '\r' && c != '\t' {
			return false
		}
	}
	return true
}

// stripPasswordFromADJSON 把 auth.ad JSON 中的 password 字段剔除。
// 返回 (cleaned JSON, modified flag, err)。
//
// 设计选择：**保留原 JSON 结构**，仅当确实存在 password 字段时才删除；
// 如果解析失败（JSON 损坏），返回原值 + modified=false，让上层决定如何处理。
func (e *CredentialEncryptor) stripPasswordFromADJSON(raw string) (string, bool, error) {
	if raw == "" {
		return raw, false, nil
	}
	// 用 map[string]interface{} 而不是强类型 struct，避免引入对 auth.ADAuthConfig
	// 字段集合的硬编码（DB 里的 JSON 可能包含未来扩展字段）
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		// 解析失败：保留原值，不修改（让 InvariantScan 单独报错）
		return raw, false, nil
	}
	if _, exists := m["password"]; !exists {
		return raw, false, nil
	}
	delete(m, "password")
	out, err := json.Marshal(m)
	if err != nil {
		return raw, false, fmt.Errorf("序列化去除 password 后的 JSON 失败: %w", err)
	}
	return string(out), true, nil
}

// ---------------------------------------------------------------------------
// 凭据按 version 计数（启动期 / 轮换后可观测）
// ---------------------------------------------------------------------------

// VersionCounts 报告单个 envelope 列（input_configs.password 等）按 version 切片的行数。
// 版本计数是 fail-closed 启动流程中的关键可观测性数据：operator 通过日志确认
// （1）历史 plaintext 已全部 envelope 化；（2）previous 版本在 RotateIfNeeded 后归零。
type VersionCounts struct {
	Column          string         // "input_configs.password" 等
	Total           int            // 该列所有非空 envelope 行数
	ByVersion       map[string]int // version -> 行数（包含 current / previous / unknown）
	EmptyRows       int            // 该列空字符串行数（视为"无凭据"，合法）
	NonEnvelopeRows int            // 该列"非 envelope"行数（明文 / base64-stub，迁移遗漏，应为 0）
	UnknownVersion  int            // envelope 但 version 不在 current/previous 白名单
}

// FormatForLog 把 VersionCounts 序列化成适合 zap logger 字段的扁平 map。
// keys: column、total、by_version__<v>、empty、non_envelope、unknown_version。
func (vc VersionCounts) FormatForLog() []zap.Field {
	fields := []zap.Field{
		zap.String("column", vc.Column),
		zap.Int("total", vc.Total),
		zap.Int("empty_rows", vc.EmptyRows),
		zap.Int("non_envelope_rows", vc.NonEnvelopeRows),
		zap.Int("unknown_version_rows", vc.UnknownVersion),
	}
	// 排序输出，避免 map 迭代顺序波动打乱日志
	versions := make([]string, 0, len(vc.ByVersion))
	for v := range vc.ByVersion {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	for _, v := range versions {
		fields = append(fields, zap.Int("by_version__"+v, vc.ByVersion[v]))
	}
	return fields
}

// countColumnByVersion 扫描指定表的指定列，按 envelope version 切片统计。
//
// 行为约定：
//   - 空字符串视为"无凭据"，计入 EmptyRows（合法）；
//   - 形如 "SM4:<ver>:<base64>" 的字符串 → ByVersion[ver]++；
//     若 ver 既不是 current 也不是 previous，则同时记入 UnknownVersion；
//   - 其他非空字符串（明文 / base64-stub / 损坏 envelope） → 记入 NonEnvelopeRows。
//
// 调用场景：MigratePlaintextToGCM 之后、RotateIfNeeded 之后、第二次 InvariantScan 之后；
// 日志输出后 operator 可视化确认每一阶段的 version 分布。
func (e *CredentialEncryptor) countColumnByVersion(ctx context.Context, db *gorm.DB, table, column string) (VersionCounts, error) {
	counts := VersionCounts{
		Column:    table + "." + column,
		ByVersion: make(map[string]int),
	}

	// 用 raw SQL SELECT 仅取目标列 —— 避免 GORM 把 SELECT column 当作 model 字段扫描
	// （column=password 时，Scan 到 type=struct {Value string} 会因 gorm scanner 不识别而失败）。
	// 表名 / 列名都来自内部常量，不存在注入风险。
	rows, err := db.WithContext(ctx).Raw(
		"SELECT " + column + " AS v FROM " + table,
	).Rows()
	if err != nil {
		return counts, fmt.Errorf("countColumnByVersion: 扫描 %s.%s 失败: %w", table, column, err)
	}
	defer rows.Close()

	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return counts, fmt.Errorf("countColumnByVersion: Scan %s.%s 失败: %w", table, column, err)
		}
		if v == "" {
			counts.EmptyRows++
			continue
		}
		version, _, err := utils.ParseCredentialEnvelope(v)
		if err != nil {
			// 非 envelope → 迁移遗漏或历史损坏
			counts.NonEnvelopeRows++
			continue
		}
		counts.ByVersion[version]++
		if version != e.currentVersion && version != e.previousVersion {
			counts.UnknownVersion++
		}
		counts.Total++
	}
	if err := rows.Err(); err != nil {
		return counts, fmt.Errorf("countColumnByVersion: 迭代 %s.%s 失败: %w", table, column, err)
	}
	return counts, nil
}

// CountByVersion 报告所有受保护凭据列的 version 分布快照。
//
// 返回结构：每个 (table, column) 元组对应一个 VersionCounts；调用方负责把
// 结果打到日志（MigratePlaintextToGCM 后、RotateIfNeeded 后、第二次 InvariantScan 后各一次）。
//
// 设计选择：扫描 input_configs.password / input_configs.stream_password /
// system_settings[auth.ad.password] 三列；auth.ad JSON 不再包含 password
// 字段（已被 MigratePlaintextToGCM 剥离），所以跳过。
func (e *CredentialEncryptor) CountByVersion(ctx context.Context, db *gorm.DB) ([]VersionCounts, error) {
	targets := []struct {
		table, column string
	}{
		{"input_configs", "password"},
		{"input_configs", "stream_password"},
	}

	out := make([]VersionCounts, 0, len(targets)+1)
	for _, t := range targets {
		c, err := e.countColumnByVersion(ctx, db, t.table, t.column)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}

	// system_settings 是 key-value 表；我们只关心 key='auth.ad.password' 的行
	adCounts := VersionCounts{
		Column:    "system_settings[auth.ad.password]",
		ByVersion: make(map[string]int),
	}
	adRows, err := db.WithContext(ctx).Raw(
		"SELECT value AS v FROM system_settings WHERE `key` = ?", "auth.ad.password",
	).Rows()
	if err != nil {
		return nil, fmt.Errorf("CountByVersion: 扫描 system_settings[auth.ad.password] 失败: %w", err)
	}
	defer adRows.Close()
	for adRows.Next() {
		var v string
		if err := adRows.Scan(&v); err != nil {
			return nil, fmt.Errorf("CountByVersion: Scan auth.ad.password 失败: %w", err)
		}
		if v == "" {
			adCounts.EmptyRows++
			continue
		}
		version, _, err := utils.ParseCredentialEnvelope(v)
		if err != nil {
			adCounts.NonEnvelopeRows++
			continue
		}
		adCounts.ByVersion[version]++
		if version != e.currentVersion && version != e.previousVersion {
			adCounts.UnknownVersion++
		}
		adCounts.Total++
	}
	if err := adRows.Err(); err != nil {
		return nil, fmt.Errorf("CountByVersion: 迭代 auth.ad.password 失败: %w", err)
	}
	out = append(out, adCounts)
	return out, nil
}

// LogVersionCounts 把 CountByVersion 结果扁平化成 zap fields 并以 Info 级别输出。
//
// 输出示例（INFO 阶段 = "after_migrate"）：
//
//	"credential version counts" column="input_configs.password" total=12
//	    empty_rows=3 non_envelope_rows=0 unknown_version_rows=0 by_version__v1=9
//
// stage 取值：建议 "after_migrate" / "after_rotate" / "after_invariant"，
// 方便 operator 在日志聚合系统按 stage 切片观察轮换进度。
func (e *CredentialEncryptor) LogVersionCounts(ctx context.Context, db *gorm.DB, stage string) error {
	counts, err := e.CountByVersion(ctx, db)
	if err != nil {
		return err
	}
	for _, c := range counts {
		fields := c.FormatForLog()
		fields = append(fields, zap.String("stage", stage))
		e.logger.Info("credential version counts", fields...)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 启动期一致性扫描
// ---------------------------------------------------------------------------

// invariantError 累积所有失败行，最后一次性返回。
type invariantError struct {
	failures []string
}

func (e *invariantError) Error() string {
	if len(e.failures) == 0 {
		return ""
	}
	return "invariant scan failed: " + strings.Join(e.failures, "; ")
}

func (e *invariantError) addf(format string, args ...interface{}) {
	e.failures = append(e.failures, fmt.Sprintf(format, args...))
}

// InvariantScan 在所有凭据上做完整一致性扫描：
//   - input_configs.password / stream_password 必须为合法 envelope（current 或 previous）
//     且能成功解密
//   - system_settings['auth.ad.password'] 同上
//   - system_settings['auth.ad'] JSON 不含 password 字段
//
// 任何失败都累积到返回值里（不抛 panic）。调用方决定如何处理。
// 严格模式：fail-closed 启动要求 0 失败。
func (e *CredentialEncryptor) InvariantScan(ctx context.Context, db *gorm.DB) error {
	invErr := &invariantError{}

	// input_configs —— Unscoped 包含软删除行
	type rowCheck struct {
		ID             uint
		Password       string
		StreamPassword string
		DeletedAt      gorm.DeletedAt
	}
	var rows []rowCheck
	if err := db.WithContext(ctx).Unscoped().
		Table("input_configs").
		Select("id, password, stream_password, deleted_at").
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("扫描 input_configs 失败: %w", err)
	}
	for _, r := range rows {
		e.checkColumn(invErr, "input_configs", r.ID, "password", r.Password)
		e.checkColumn(invErr, "input_configs", r.ID, "stream_password", r.StreamPassword)
	}

	// system_settings['auth.ad.password']
	var pwdSetting models.SystemSetting
	if err := db.WithContext(ctx).Unscoped().
		Where("`key` = ?", "auth.ad.password").
		First(&pwdSetting).Error; err == nil {
		e.checkColumn(invErr, "system_settings", pwdSetting.ID, "auth.ad.password", pwdSetting.Value)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("读取 auth.ad.password 失败: %w", err)
	}

	// system_settings['auth.ad'] —— 不含 password 字段
	var adSetting models.SystemSetting
	if err := db.WithContext(ctx).Unscoped().
		Where("`key` = ?", "auth.ad").
		First(&adSetting).Error; err == nil {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(adSetting.Value), &m); err == nil {
			if _, exists := m["password"]; exists {
				invErr.addf("system_settings[auth.ad] (id=%d) 仍包含 password 字段", adSetting.ID)
			}
		}
		// JSON 解析失败不计入 invariant（保守处理：留下一次启动机会）
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("读取 auth.ad 失败: %w", err)
	}

	if len(invErr.failures) > 0 {
		return invErr
	}
	return nil
}

// checkColumn 校验单列值：必须为合法 envelope 且能解密。
func (e *CredentialEncryptor) checkColumn(invErr *invariantError, table string, id uint, column, value string) {
	if value == "" {
		return // 空值视为合法（凭据未设置）
	}
	version, payload, err := utils.ParseCredentialEnvelope(value)
	if err != nil {
		invErr.addf("%s (id=%d).%s 不是合法 envelope: %v", table, id, column, err)
		return
	}
	switch version {
	case e.currentVersion:
		if _, err := utils.DecryptGCM(e.currentKey, payload); err != nil {
			invErr.addf("%s (id=%d).%s current 版本解密失败: %v", table, id, column, err)
		}
	case e.previousVersion:
		if e.previousKey == nil {
			invErr.addf("%s (id=%d).%s 是 previous 版本 %q，但本进程未配置 previous 密钥",
				table, id, column, version)
			return
		}
		if _, err := utils.DecryptGCM(e.previousKey, payload); err != nil {
			invErr.addf("%s (id=%d).%s previous 版本解密失败: %v", table, id, column, err)
		}
	default:
		invErr.addf("%s (id=%d).%s version=%q 既不是 current=%s 也不是 previous=%s",
			table, id, column, version, e.currentVersion, e.previousVersion)
	}
}

// RotateIfNeeded 把所有 previous 版本 envelope 重写成 current 版本（轮换）。
// 只处理：input_configs.password / stream_password / system_settings['auth.ad.password']。
//
// 设计选择：不在 RotateIfNeeded 内做事务包裹（让每个 Update 独立），便于故障恢复；
// 但要求**之前**的 InvariantScan 已确认所有密文合法可解密 —— 这是 cmd/server/app.go
// 的 Initialize() 启动顺序的硬约束。
func (e *CredentialEncryptor) RotateIfNeeded(ctx context.Context, db *gorm.DB) (rotated int, err error) {
	if !e.HasPrevious() {
		return 0, nil // 无 previous 密钥 → 无需轮换
	}

	type row struct {
		ID             uint
		Password       string
		StreamPassword string
	}
	var rows []row
	if err := db.WithContext(ctx).Unscoped().
		Table("input_configs").
		Select("id, password, stream_password").
		Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("扫描 input_configs 失败: %w", err)
	}

	for _, r := range rows {
		if r.Password != "" && strings.HasPrefix(r.Password, utils.ENCRYPTION_PREFIX) {
			version, _, perr := utils.ParseCredentialEnvelope(r.Password)
			if perr == nil && version == e.previousVersion {
				plaintext, derr := e.Decrypt(r.Password)
				if derr != nil {
					return rotated, fmt.Errorf("id=%d password 解密失败（previous）: %w", r.ID, derr)
				}
				envelope, eerr := e.Encrypt(plaintext)
				if eerr != nil {
					return rotated, fmt.Errorf("id=%d password 重新加密失败: %w", r.ID, eerr)
				}
				if err := db.WithContext(ctx).Unscoped().Model(&models.InputConfig{}).
					Where("id = ?", r.ID).
					Update("password", envelope).Error; err != nil {
					return rotated, fmt.Errorf("id=%d password 轮换写库失败: %w", r.ID, err)
				}
				rotated++
			}
		}
		if r.StreamPassword != "" && strings.HasPrefix(r.StreamPassword, utils.ENCRYPTION_PREFIX) {
			version, _, perr := utils.ParseCredentialEnvelope(r.StreamPassword)
			if perr == nil && version == e.previousVersion {
				plaintext, derr := e.Decrypt(r.StreamPassword)
				if derr != nil {
					return rotated, fmt.Errorf("id=%d stream_password 解密失败（previous）: %w", r.ID, derr)
				}
				envelope, eerr := e.Encrypt(plaintext)
				if eerr != nil {
					return rotated, fmt.Errorf("id=%d stream_password 重新加密失败: %w", r.ID, eerr)
				}
				if err := db.WithContext(ctx).Unscoped().Model(&models.InputConfig{}).
					Where("id = ?", r.ID).
					Update("stream_password", envelope).Error; err != nil {
					return rotated, fmt.Errorf("id=%d stream_password 轮换写库失败: %w", r.ID, err)
				}
				rotated++
			}
		}
	}

	// system_settings['auth.ad.password']
	var pwdSetting models.SystemSetting
	if err := db.WithContext(ctx).Unscoped().
		Where("`key` = ?", "auth.ad.password").
		First(&pwdSetting).Error; err == nil {
		if pwdSetting.Value != "" && strings.HasPrefix(pwdSetting.Value, utils.ENCRYPTION_PREFIX) {
			version, _, perr := utils.ParseCredentialEnvelope(pwdSetting.Value)
			if perr == nil && version == e.previousVersion {
				plaintext, derr := e.Decrypt(pwdSetting.Value)
				if derr != nil {
					return rotated, fmt.Errorf("auth.ad.password 解密失败（previous）: %w", derr)
				}
				envelope, eerr := e.Encrypt(plaintext)
				if eerr != nil {
					return rotated, fmt.Errorf("auth.ad.password 重新加密失败: %w", eerr)
				}
				if err := db.WithContext(ctx).Unscoped().Model(&models.SystemSetting{}).
					Where("id = ?", pwdSetting.ID).
					Update("value", envelope).Error; err != nil {
					return rotated, fmt.Errorf("auth.ad.password 轮换写库失败: %w", err)
				}
				rotated++
			}
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return rotated, fmt.Errorf("读取 auth.ad.password 失败: %w", err)
	}

	if rotated > 0 {
		e.logger.Info("凭据轮换完成",
			zap.Int("rotated", rotated),
			zap.String("from", e.previousVersion),
			zap.String("to", e.currentVersion),
		)
	}
	return rotated, nil
}
