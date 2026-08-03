package models

import "time"

// HLSJtiRecord HLS Token 一次性标识符持久化记录（Phase 19 D3）
//
// 历史背景：Phase 17 引入 jti 防重放；Phase 19 Wave 1 SEC-004 重构为内存 map
// `usedJTIs map[string]int64`，定位为"TTL 索引追踪"，限定为单实例 +
// tokenDuration TTL 窗口。Phase 19 D3 将其从内存升级为 DB 表，跨实例/重启
// 仍能保持重放窗口与未来吊销能力。
//
// 字段设计：
//   - Jti（PK）：32 字符 hex，crypto/rand 生成
//   - ExpiresAt：Unix 时间戳（与 HLSTokenClaims.ExpiresAt 对齐）；用作驱逐条件
//   - CreatedAt：审计字段——用于"表过大时按插入时间驱逐最老"硬上限检查
//
// 注意：jti 本身无业务敏感性（纯随机），存明文即可。表体量受 tokenDuration + QPS
// 控制，长 tokenDuration（如 24h）+ 高 QPS 时 1 天累计 ≤ tokenDuration * qps；常态
// 远低于 100k 上限。
type HLSJtiRecord struct {
	Jti       string    `gorm:"primaryKey;size:64;column:jti" json:"jti"`
	ExpiresAt int64     `gorm:"index;not null;column:expires_at" json:"expires_at"` // Unix 秒
	CreatedAt time.Time `gorm:"index;not null;column:created_at" json:"created_at"`
}

// TableName 指定 GORM 表名（Phase 19 命名约定，与模型文件名对齐）。
func (HLSJtiRecord) TableName() string {
	return "hls_jti_records"
}
