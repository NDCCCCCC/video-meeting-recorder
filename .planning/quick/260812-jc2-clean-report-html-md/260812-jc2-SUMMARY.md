---
phase: quick-260812-jc2
plan: 01
subsystem: clean/report
tags: [html, report, clean-tool, file-uri, security-escaping]
requires:
  - clean/internal/browser.Summarize
  - clean/internal/im.Summarize
  - clean/internal/sensitive.SensitiveFile
  - clean/internal/scanner.FileHit
provides:
  - "clean/internal/report/html.go: writeHTML(io.Writer, *FullReport) error + fileDirLink + htmlStyle"
  - "clean/internal/report/report.go: Write 返回 (jsonPath, mdPath, htmlPath, err)"
affects:
  - "clean/cmd/clean/main.go: runAll / runScan / runClean / runReport 4 处调用点适配 3 返回值"
tech-stack:
  added: []
  patterns:
    - "html.EscapeString 全量转义动态内容（防注入）"
    - "file:/// + url.PathEscape 按段编码 + 末尾斜杠 → file 目录超链接"
    - "自包含 HTML（内联 <style>，零外部依赖）"
key-files:
  created:
    - clean/internal/report/html.go
    - clean/internal/report/html_test.go
  modified:
    - clean/internal/report/report.go
    - clean/cmd/clean/main.go
decisions:
  - "writeHTML 接受 io.Writer 而非 *os.File，便于测试用 *strings.Builder 注入"
  - "url.PathEscape 之后追加 strings.ReplaceAll(\":\", \"%3A\")，因为 RFC 3986 视冒号为合法 path 字符不编码——但 plan done-criteria 显式要求 %3A（Windows 盘符冒号在 file:/// URL 中应被编码）"
  - "敏感文件表排序用 SliceStable + 副本拷贝，避免修改 rep.SensitiveFiles（writeMarkdown 之后调用是幂等的，但拷贝更安全）"
metrics:
  duration: 18min
  completed: 2026-08-12
  tasks: 2
  files: 4
  loc_added: 1342
  loc_modified: 0
---

# Phase quick-260812-jc2 Plan 01: Clean Tool HTML Report Summary

为 clean 工具的 report 包增加自包含 HTML 报告输出，五节结构完全对齐同名 .md，并把所有文件路径渲染成 `file:///` 目录超链接——本地双击 .html 后点击路径即可在资源管理器中打开所在目录，审计流程从"复制路径 → win+R → 粘贴"变成"点一下"。

## What Was Built

### Task 1 — `clean/internal/report/html.go` + 单元测试

- **writeHTML(w io.Writer, rep *FullReport) error**：把 FullReport 渲染为自包含 HTML。
  - 顶层结构：`<!DOCTYPE html><html lang="zh-CN">...<head><meta charset="utf-8">...<style>...</style></head><body>...</body></html>`
  - 五节标题文字与同名 MD 完全一致（含「（已移到回收站）」「（<strong>仅报告，不删除</strong>）」）
  - 一/二/三节为 `<table>` 表格 + 末尾总计行；二节 IM 行使用 `s.App`（与 MD 保持差异）
  - 四节按 Risk rank (high=0, medium=1, low=2) 排序，使用副本 `SliceStable` 避免副作用
  - 五节按 `credential → pii` 顺序分组，每组 `<h3>` 标题 + `<p>` 说明，每个文件 `<h4>` 路径超链接 + `<pre><code>` 代码块
  - 空 PasswordHits 输出 `<p><em>未发现敏感数据</em></p>`
- **fileDirLink(path string) string**：渲染 `<a href="file:///...">path</a>` 超链接。
  - 三斜杠开头 + 段级 `url.PathEscape` + 手动 `:` → `%3A` 后处理 + 末尾斜杠
  - 显示文本 = `html.EscapeString(path)`（人读原样），title = "点击在资源管理器中打开所在目录"
  - 空路径返回空串（defensive）
- **htmlStyle 常量**：内联 `<style>`，仅出现一次（`<head>` 内）。
  - 字体栈：Win `Microsoft YaHei` / mac `PingFang SC` / Linux `WenQuanYi Micro Hei` + sans-serif fallback
  - 等宽字体：`Cascadia Code / Consolas / Courier New / monospace`
  - body 最大宽度 1100px 居中；表格 border-collapse + nth-child(even) 浅灰底；pre 横向滚动；a 颜色 #1565c0
- **测试 `html_test.go`**：3 个测试覆盖结构 / 转义 / 链接编码 / 空命中分支。
  - `TestWriteHTML_StructureAndEscape`：18 个 `strings.Contains` 断言（DOCTYPE / title / 五节 h2 / 转义主机名 / file 链接 href / pre+code / L1 标记 / `&lt;y&gt;` / `&#34;x&lt;y&gt;z&#34;` 等）
  - `TestWriteHTML_EmptyHits`：空 PasswordHits 输出 `<em>未发现敏感数据</em>`
  - `TestFileDirLink_URLCoding`：`%3A` / `%20` / 三斜杠 / 末尾斜杠 / 显示文本原样 / 空路径 defensive

### Task 2 — `report.Write` 签名扩展 + 4 调用点适配

- **`report.Write(rep, outDir) (jsonPath, mdPath, htmlPath string, err error)`**：
  - 在 writeMarkdown 调用块后新增 `os.Create(htmlPath)` + `defer Close()` + `writeHTML(hf, rep)`
  - 所有早退 return（MkdirAll / JSON Create / JSON Encode / MD Create / writeMarkdown / HTML Create / writeHTML）统一返回 3 个空串 + err
  - 文件顶部 doc comment 更新为「.json / .md / .html」三种输出
- **`cmd/clean/main.go` 4 处调用点**：
  - `runAll`（~L177）：解构 3 返回值 + 追加一行 `fmt.Printf("    %s\n", htmlPath)`
  - `runScan`（~L391）：解构 3 返回值 + format 改为 `\n  %s\n  %s\n  %s\n`
  - `runClean`（~L527）：同 runScan
  - `runReport`（~L811）：解构 `_, md, htmlPath, err` + 追加一行 `fmt.Printf("已重新生成: %s\n", htmlPath)`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] url.PathEscape 不编码冒号 → 手动追加了 `:` → `%3A` 后处理**
- **Found during:** Task 1 GREEN 阶段，测试 `sensitive file dir link href` 失败
- **Issue:** plan action 写「每段用 url.PathEscape 编码」，但 RFC 3986 视 `:` 为合法 path 字符（sub-delims），url.PathEscape 不编码 → 实际 href 是 `file:///C:/Users/A%20B/.ssh/`（冒号未编码），与 plan done-criteria「file:// 链接含 %3A」冲突
- **Fix:** 在 PathEscape 之后追加 `strings.ReplaceAll(segs[i], ":", "%3A")` —— 两步互不冲突，最终 href 变成 `file:///C%3A/Users/A%20B/.ssh/`，满足 plan 要求
- **Files modified:** clean/internal/report/html.go
- **Commit:** 86c3733

**2. [Rule 1 - Bug] 测试断言 `&quot;` 与 Go html.EscapeString 实际输出 `&#34;` 不符**
- **Found during:** Task 1 GREEN 阶段，测试 `evidence double quote escaped` 失败
- **Issue:** plan 行为规范引用「Go 的 html.EscapeString」，但断言期望 `&quot;x&lt;y&gt;z&quot;`（HTML4 实体），而 Go 标准库实际输出 `&#34;x&lt;y&gt;z&#34;`（数字实体）
- **Fix:** 测试断言改为 `&#34;x&lt;y&gt;z&#34;`（对齐 Go html.EscapeString 实际行为，而非 HTML4 实体名）；comment 解释差异来源
- **Files modified:** clean/internal/report/html_test.go
- **Commit:** 86c3733

## Verification

所有 `<verification>` 与 `<verify>` / `<done>` gate 全部通过：

| Gate | 命令 | 结果 |
|---|---|---|
| 模块编译 | `go build ./...` | exit 0 |
| 静态检查 | `go vet ./...` | exit 0 |
| 单元测试 | `go test ./internal/report/... -v` | 3/3 通过（StructureAndEscape / EmptyHits / FileDirLink_URLCoding） |
| 全量测试 | `go test ./...` | report + shred 全绿；其他包 no test files |
| E2E 生成 | `./clean.exe report reports/20260812-001021-report.json` | 重新生成 .md + .html，HTML = 85,896 字节（≈ 84KB，远超 5KB 阈值） |
| HTML 结构 | grep 抽查 | DOCTYPE + `<title>清理报告  20260812-001021</title>` + 五节 `<h2>` 齐全（一/二/三/四/五） |
| file:// 链接 | grep `file:///` | 至少一处含 `file:///C%3A/Users/CPIC/.ssh/`（冒号 %3A + 空格 %20 + 三斜杠 + 末尾斜杠） |
| 命中代码块 | grep `<pre><code>` | 145 处（与源 JSON 145 个命中文件一致） |
| 防注入 | grep `<script` | 0 处未转义 `<script` |

## Known Stubs

None — 全部数据流通（rep.SensitiveFiles / rep.PasswordHits / rep.Hostname / rep.UserHome / rep.Timestamp）均接入实际渲染管线，无 placeholder / TODO / FIXME。

## Threat Flags

None — 未引入新的网络端点 / 鉴权路径 / 文件系统访问模式 / 信任边界 schema 变更。所有动态内容（路径、命中行文本、Hostname 等）经 `html.EscapeString` 转义，HTML 自包含无外部资源加载。`file:///` 链接是浏览器本地协议，仅在用户主动点击时调起 explorer，不构成执行面。

## Self-Check

文件存在性 + commit 存在性已验证：

- FOUND: clean/internal/report/html.go
- FOUND: clean/internal/report/html_test.go
- FOUND: clean/internal/report/report.go（已含 htmlPath）
- FOUND: clean/cmd/clean/main.go（4 处调用点均已适配）
- FOUND commit: 86c3733（Task 1）
- FOUND commit: f87e9ac（Task 2）

## Self-Check: PASSED
