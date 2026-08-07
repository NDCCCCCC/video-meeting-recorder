package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// skipIfMissing 安全跳过：当目标 YAML 不存在时 t.Skip（本测试是部署期模板
// drift gate，不是文件存在性 gate）。
//   - 本地开发：config.yaml + bin/config.yaml 由运维手工准备，测试正常执行。
//   - 干净 CI checkout：文件被 .gitignore 排除，bin/ 由 cross-build job 创建，
//     root 可能在 createDefaultConfigFile() 触发后存在、也可能尚未触发。
//     Skip 比 FAIL 更准确反映"无模板可校验"的现实。
func skipIfMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("yaml drift gate skipped: %s not present (likely gitignored / fresh checkout)", path)
		}
		t.Fatalf("stat %s: %v", path, err)
	}
}

// expectedSmartEndKeys 是 REQUIREMENTS.md:58 与 23-RESEARCH.md CFG-02 节明确锁定的
// 14 项 smart_end 配置键集合。任意漂移(多一个/少一个/重命名)都应被本测试捕获。
//
// 顺序与 SmartEndConfig struct 字段顺序保持一致以便 diff 友好。
var expectedSmartEndKeys = []string{
	"enabled",
	"silence_db",
	"silence_duration_s",
	"file_stall_s",
	"file_min_growth_bps",
	"huawei_enabled",
	"huawei_poll_interval_s",
	"huawei_persist_s",
	"huawei_failure_threshold",
	"check_interval_s",
	"extend_step_min",
	"max_extend_count",
	"stat_failure_threshold",
	"degrade_on_silence_loss",
}

// expectedSmartEndDefaults 与 REQUIREMENTS.md:58 及 applySmartEndDefaults + Viper
// SetDefault 三处保持一致的默认值。TestSmartEndYAML_ExpectedDefaults 校验根
// config.yaml 中的字面值与此映射逐项相等。
//
// 期望与 SmartEndConfig 实际加载值在 Viper Unmarshal + setDefaults 后等价 —
// 即 cfg.SmartEnd.* 的运行期值等于此 map(前提是 YAML 字面值就是这些默认)。
var expectedSmartEndDefaults = map[string]interface{}{
	"enabled":                  true,
	"silence_db":               -30,
	"silence_duration_s":       30,
	"file_stall_s":             120,
	"file_min_growth_bps":      int64(1024),
	"huawei_enabled":           true,
	"huawei_poll_interval_s":   30,
	"huawei_persist_s":         30,
	"huawei_failure_threshold": 3,
	"check_interval_s":         5,
	"extend_step_min":          30,
	"max_extend_count":         4,
	"stat_failure_threshold":   3,
	"degrade_on_silence_loss":  true,
}

// projectRoot 返回本测试源文件相对仓库根的解析路径。内部测试运行的工作目录
// 会随 `go test` / IDE runner 改变,使用 runtime.Caller(0) 直接定位本文件再向上
// 走两级(internal/config -> internal -> repo root)以保证从仓库任意位置
// `go test ./internal/config` 都能正确读到 config.yaml + bin/config.yaml。
func projectRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller(0) failed")
	}
	// thisFile = .../internal/config/smart_end_yaml_test.go
	dir := filepath.Dir(thisFile)
	root := filepath.Join(dir, "..", "..") // internal/config -> ../../  = repo root
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) failed: %v", root, err)
	}
	return abs
}

// loadSmartEndSection 读取 config.yaml 的 smart_end: 段并返回 *yaml.Node 树状结构。
// 使用 yaml.v3 而不是 map[string]interface{}:Node 保留顺序(OrderedMap)便于
// RootBinSync 测试逐项比对 key/value 而不依赖 map iteration randomness。
func loadSmartEndSection(t *testing.T, absPath string) *yaml.Node {
	t.Helper()
	raw, err := os.ReadFile(absPath)
	require.NoError(t, err, "read %s", absPath)

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(raw, &root),
		"yaml.Unmarshal %s (smart_end section must parse cleanly)", absPath)

	// root 是 DocumentNode;其第一个 Content 是顶层 MappingNode。
	require.Equal(t, yaml.DocumentNode, root.Kind, "%s root must be DocumentNode", absPath)
	require.NotEmpty(t, root.Content, "%s document content empty", absPath)
	topMapping := root.Content[0]
	require.Equal(t, yaml.MappingNode, topMapping.Kind, "%s top level must be MappingNode", absPath)

	for i := 0; i < len(topMapping.Content); i += 2 {
		keyNode := topMapping.Content[i]
		valNode := topMapping.Content[i+1]
		if keyNode.Value == "smart_end" {
			require.Equal(t, yaml.MappingNode, valNode.Kind,
				"smart_end section in %s must be a MappingNode, got Kind=%d", absPath, valNode.Kind)
			return valNode
		}
	}
	t.Fatalf("smart_end: top-level section not found in %s", absPath)
	return nil
}

// smartEndMap 把 yaml.Node Mapping 转为 map[string]yaml.Node 便于断言。
// 同时返回有序 key 切片(按 yaml 中出现的顺序而非字母序)以便诊断 diff。
func smartEndMap(t *testing.T, n *yaml.Node) (map[string]yaml.Node, []string) {
	t.Helper()
	require.Equal(t, yaml.MappingNode, n.Kind, "expected MappingNode")
	out := make(map[string]yaml.Node, len(n.Content)/2)
	ordered := make([]string, 0, len(n.Content)/2)
	for i := 0; i < len(n.Content); i += 2 {
		k := n.Content[i].Value
		out[k] = *n.Content[i+1]
		ordered = append(ordered, k)
	}
	return out, ordered
}

// nodeComparableValue 把 yaml 标量节点规整为可与 expectedSmartEndDefaults 中
// 字面值比较的 Go 值。yaml.v3 对整数解码为 int、bool 解码为 bool、浮点为 float64;
// 我们把 int / int64 / float64 全部归一化以便一处断言覆盖三种字面写法。
func nodeComparableValue(t *testing.T, n yaml.Node) interface{} {
	t.Helper()
	if n.Kind != yaml.ScalarNode {
		t.Fatalf("nodeComparableValue: expected ScalarNode, got Kind=%d", n.Kind)
	}
	switch n.Tag {
	case "!!bool":
		return n.Value == "true"
	case "!!int":
		if strings.Contains(n.Value, ".") {
			f, err := strconv.ParseFloat(n.Value, 64)
			require.NoError(t, err, "parse float %q", n.Value)
			return int64(f)
		}
		i, err := strconv.ParseInt(n.Value, 10, 64)
		require.NoError(t, err, "parse int %q", n.Value)
		return i
	case "!!float":
		f, err := strconv.ParseFloat(n.Value, 64)
		require.NoError(t, err, "parse float %q", n.Value)
		return f
	case "!!str", "!!string":
		return n.Value
	default:
		// 未声明 tag 的裸数字/bool 兜底(YAML 1.2 resolver 行为差异):
		return n.Value
	}
}

// TestSmartEndYAML_Exactly14Keys 校验 config.yaml 与 bin/config.yaml 两份部署
// 模板的 smart_end: 段都恰好包含 14 个键且键集合与 REQUIREMENTS.md 锁定列表相等。
//
// 这是 PLAN §"Files to Modify" 表明确要求的"exact-key-set 防漂移门禁"。
func TestSmartEndYAML_Exactly14Keys(t *testing.T) {
	root := projectRoot(t)
	cases := []struct {
		name string
		path string
	}{
		{"root config.yaml", filepath.Join(root, "config.yaml")},
		{"bin/config.yaml", filepath.Join(root, "bin", "config.yaml")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			skipIfMissing(t, tc.path)
			n := loadSmartEndSection(t, tc.path)
			m, _ := smartEndMap(t, n)
			assert.Equal(t, 14, len(m),
				"%s smart_end must have exactly 14 keys, got %d (%v)",
				tc.path, len(m), sortedKeys(m))
			// 取出 YAML 实际键集合,与期望做集合相等比对(顺序不敏感)
			actual := sortedKeys(m)
			expectedSorted := append([]string(nil), expectedSmartEndKeys...)
			sort.Strings(expectedSorted)
			assert.Equal(t, expectedSorted, actual,
				"%s smart_end key set drift:\n  got:      %v\n  expected: %v",
				tc.path, actual, expectedSorted)
		})
	}
}

// TestSmartEndYAML_RootBinSync 是 RESEARCH.md "Common Pitfall" 节
// "Updating only root config.yaml: packaged bin/config.yaml would drift." 的回归门禁。
//
// 强校验:两份模板的 smart_end: 段必须:
//
//  1. 键集合完全相同 (集合相等)
//  2. 每个键对应值的字面值完全相同 (标量解析后逐项相等)
//
// 任何未来只改其中一个文件的提交都会触发本测试失败。
func TestSmartEndYAML_RootBinSync(t *testing.T) {
	root := projectRoot(t)
	rootPath := filepath.Join(root, "config.yaml")
	binPath := filepath.Join(root, "bin", "config.yaml")
	skipIfMissing(t, rootPath)
	skipIfMissing(t, binPath)

	rootNode := loadSmartEndSection(t, rootPath)
	binNode := loadSmartEndSection(t, binPath)

	rootMap, _ := smartEndMap(t, rootNode)
	binMap, _ := smartEndMap(t, binNode)

	// (1) 键集合相等
	rootKeys := sortedKeys(rootMap)
	binKeys := sortedKeys(binMap)
	assert.Equal(t, rootKeys, binKeys,
		"root vs bin key set drift:\n  root: %v\n  bin:  %v", rootKeys, binKeys)

	// (2) 每个键对应值字面值相等
	for _, k := range rootKeys {
		rv := rootMap[k]
		bv, ok := binMap[k]
		require.True(t, ok, "key %q present in root but missing in bin", k)
		rVal := nodeComparableValue(t, rv)
		bVal := nodeComparableValue(t, bv)
		assert.True(t, reflect.DeepEqual(normalizeNumericValue(rVal), normalizeNumericValue(bVal)),
			"value drift on key %q:\n  root: %#v (%s)\n  bin:  %#v (%s)",
			k, rVal, rv.Tag, bVal, bv.Tag)
	}
}

// TestSmartEndYAML_ExpectedDefaults 校验根 config.yaml 中 smart_end: 段的字面
// 值与 REQUIREMENTS.md:58 + applySmartEndDefaults 一致;避免"键在但值错"导致的
// 静默配置漂移。
//
// 不校验 bin/config.yaml:RootBinSync 已经把两份文件绑死,任一边漂移即失败。
func TestSmartEndYAML_ExpectedDefaults(t *testing.T) {
	root := projectRoot(t)
	rootPath := filepath.Join(root, "config.yaml")
	skipIfMissing(t, rootPath)

	node := loadSmartEndSection(t, rootPath)
	m, _ := smartEndMap(t, node)

	// 把 YAML 实际值规整后与期望 map 对比:数量、键集合、值三重一致。
	require.Equal(t, len(expectedSmartEndDefaults), len(m),
		"root smart_end key count must equal expected (%d), got %d",
		len(expectedSmartEndDefaults), len(m))

	for k, want := range expectedSmartEndDefaults {
		gotNode, ok := m[k]
		require.True(t, ok, "missing key %q in root config.yaml smart_end", k)
		got := nodeComparableValue(t, gotNode)
		// 期望 map 中数字按 int/int64 标注;YAML 解码后归一为 int64;用
		// normalizeNumericValue 统一 int / int64 后 reflect.DeepEqual 比较。
		assert.True(t, reflect.DeepEqual(normalizeNumericValue(want), normalizeNumericValue(got)),
			"default drift on key %q:\n  got:  %#v\n  want: %#v", k, got, want)
	}
}

// TestSmartEndYAML_ViperLoadsCleanly 端到端:从根 config.yaml 整文件内容,通过 Viper
// SetDefaults (3 bool) + Unmarshal 装入 Config;校验 cfg.SmartEnd 字段值与 YAML
// 字面值一致,证明 mapstructure triple tag 在嵌套段上工作。
//
// 与 smart_end_test.go TestSmartEndConfig_Defaults 的区别:后者用 inline YAML,
// 这里用真实的 config.yaml 源文件,确保 Viper 真的能读项目实际部署配置。
func TestSmartEndYAML_ViperLoadsCleanly(t *testing.T) {
	root := projectRoot(t)
	rootPath := filepath.Join(root, "config.yaml")
	skipIfMissing(t, rootPath)

	// 读整份 config.yaml:Viper 需要 Document 形式,只 encode 子节点会丢失
	// Document wrapper,导致 unmarshal 解析不出数字字面值。
	raw, err := os.ReadFile(rootPath)
	require.NoError(t, err, "read %s", rootPath)

	v := viper.New()
	// 与 Load() 行为对齐:3 个 true-valued bool 在 Unmarshal 前 SetDefault
	v.SetDefault("smart_end.enabled", true)
	v.SetDefault("smart_end.huawei_enabled", true)
	v.SetDefault("smart_end.degrade_on_silence_loss", true)
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(bytes.NewReader(raw)))

	var cfg Config
	require.NoError(t, v.Unmarshal(&cfg))

	// spot-check 关键字段,证明 mapstructure 解析无误
	assert.True(t, cfg.SmartEnd.Enabled, "Enabled must be true from YAML default")
	assert.Equal(t, -30, cfg.SmartEnd.SilenceDB)
	assert.Equal(t, 30, cfg.SmartEnd.SilenceDurationS)
	assert.Equal(t, 120, cfg.SmartEnd.FileStallS)
	assert.Equal(t, int64(1024), cfg.SmartEnd.FileMinGrowthBPS)
	assert.True(t, cfg.SmartEnd.HuaweiEnabled)
	assert.Equal(t, 30, cfg.SmartEnd.HuaweiPollIntervalS)
	assert.Equal(t, 30, cfg.SmartEnd.HuaweiPersistS)
	assert.Equal(t, 3, cfg.SmartEnd.HuaweiFailureThreshold)
	assert.Equal(t, 5, cfg.SmartEnd.CheckIntervalS)
	assert.Equal(t, 30, cfg.SmartEnd.ExtendStepMin)
	assert.Equal(t, 4, cfg.SmartEnd.MaxExtendCount)
	assert.Equal(t, 3, cfg.SmartEnd.StatFailureThreshold)
	assert.True(t, cfg.SmartEnd.DegradeOnSilenceLoss)
}

// sortedKeys 返回 map 键的有序切片(便于 set comparison + 错误信息可读)。
func sortedKeys(m map[string]yaml.Node) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// normalizeNumericValue 把 int / int64 / float64 统一规整为 int64,使 expectedSmartEndDefaults
// 中以 int 字面量(-30 / 30 / 120 等)声明的值与 yaml.v3 解码得到的 int64 能
// reflect.DeepEqual 比较。bool / string 透传。
func normalizeNumericValue(v interface{}) interface{} {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case uint:
		return int64(x)
	case uint32:
		return int64(x)
	case uint64:
		return int64(x)
	case float64:
		return int64(x)
	default:
		return v
	}
}