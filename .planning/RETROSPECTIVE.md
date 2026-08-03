# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.1 — 文件管理与编辑增强 / 后端安全加固

**Shipped:** 2026-08-03
**Phases:** 6 (17-22) | **Plans:** 25 | **Timeline:** 2026-07-30 → 2026-08-03 (5 days)
**Git:** 247 commits, 267 files, +35261/-5976 LOC

### What Was Built
- 后端代码审查 56 项发现全量修复（13 HIGH + 18 MEDIUM + 25 LOW），45 原子 commit 零回归
- 凭据静态加密 SM4-GCM envelope + 自动密钥轮换（SEC-003b），fail-closed 启动不变量
- ctx 全量级联（403 GORM + ~190 service 方法）+ jti replay 防御（TTL sweeper）+ error-mapping 三组件
- HandleError 统一收敛 + sentinel 体系（SentinelField 4-state）+ 自动文档生成 + CI sync-check 门禁
- v1.1 过程闭环：retro-verify phase 17/18/19 + REQUIREMENTS.md（~80 REQ-ID，0 orphan）+ auth:57 规范化
- 审计 tech debt 收尾：errors.md regenerate + 4 VALIDATION.md 回填 + phase 20 Nyquist 签核翻转

### What Worked
- **P0→P1a→P1b→P2 分级修复**：先堵高危漏洞（SEC/BUG）再清理低危（STYLE），每级独立 wave + atomic commit，可审计性极强（phase 17 的 45 commit 可逐条追溯）
- **atomic commit 纪律**：每个审查发现独立 commit，回归失败时二分定位精准
- **wave-based 并行执行**：无文件重叠的 plan 并行（如 20-02/20-03、20-04/20-05），缩短 wall-clock
- **retro-verify 范式**：代码已落地但过程产物缺失时，从 SUMMARY + git commits + live code 三源交叉重建 VERIFICATION.md（phase 21 成功补救 phase 18/19 目录丢失）
- **诚实归档纪律**：重审 audit 明确区分 gaps（全空）vs tech_debt（文档化残留），不掩盖

### What Was Inefficient
- **.planning 目录早期清理丢失 phase 18/19**：导致 phase 21 不得不 retro-verify 重建目录 + VERIFICATION，额外开销（本可随 phase 执行同步产出）
- **过程产物事后补**：VERIFICATION/VALIDATION/REQUIREMENTS 未在 phase 执行时同步写，phase 21+22 闭环占 v1.1 约 40% 工作量
- **Wave 3 执行器上游 API 错误崩溃**：phase 17 P1b 的 SUMMARY 叙述中断（代码已落地，由 orchestrator 文件系统核验恢复）
- **两轮 audit**：原始 gaps_found → 重审 tech_debt，中间需 phase 21+22 闭环；若首次执行即同步过程产物可省一轮
- **quick task 与 phase 边界模糊**：7 个审计 quick task（260729-*/260730-*）在 phase 17 之前发生，归属 v1.1 还是前置不清

### Patterns Established
- **HandleError canonical pattern**（`response.HandleError(c, err); return`）—— CR-01 双写防御，9 handler families 全量收敛
- **SentinelField 4-state**（sentinel / BusinessError / ad-hoc）—— 错误分类标准化，zap logger 统一输出
- **error-doc-gen + CI sync-check** —— 文档漂移有 CI 门禁（`.github/workflows/ci.yml:44-51`），防止 docs/errors.md 与代码脱节
- **SM4-GCM envelope + 密钥族隔离** —— 凭据加密范式（`CREDENTIAL_SM4_*` vs `SM4_SECRET` 物理隔离）
- **retro-VERIFICATION 三源交叉** —— 过程产物缺失的补救范式（SUMMARY + commits + live code）

### Key Lessons
1. **过程产物必须随 phase 执行同步产出**——VERIFICATION/VALIDATION/REQUIREMENTS 事后补的代价远高于同步写（phase 21+22 占 40%）
2. **56 项审查全量修复时，分级（P0→P2）+ atomic commit 是可审计性的关键**——每发现一 commit，回归二分定位精准
3. **架构决策（如 jti replay 加 DB 表 vs TTL sweeper）应早做**——避免规划与执行间返工
4. **.planning 目录清理要谨慎**——phase 18/19 目录丢失导致 retro-verify；归档而非删除
5. **诚实归档 > 完美归档**——区分 gaps（阻断）vs tech_debt（文档化残留），重审明确标注 provenance

### Cost Observations
- Model mix: sonnet 为主（执行 + 规划）+ opus 用于难判断（audit 重审、auth:57 控制流等价论据）；haiku 极少
- Sessions: 多个（phase 17/18/19/20/21/22 各自 + audit 两轮 + 7 quick task）
- Notable: 过程闭环（phase 21+22）占 v1.1 约 40% 工作量，提示过程纪律前置可大幅降本；retro-verify 三源交叉虽有效但本可避免

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Timeline | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.0 | ~3 周 (04-17→05-06) | 14 | 产品功能从 0 到 1，FFmpeg/通义听悟/OSS/AD 集成 |
| v1.1 | 5 天 (07-30→08-08-03) | 6 | 后端安全加固 + 质量提升，无新业务功能；引入 atomic commit + retro-verify 纪律 |

### Cumulative Quality

| Milestone | Tests Added | -race Green | Audit Status |
|-----------|-------------|-------------|--------------|
| v1.0 | ~基准 | (未系统化) | (无 formal audit) |
| v1.1 | 12 包 phase-17 + 多 wave contract tests | ✅ 全绿 | tech_debt（gaps 全空，5/5 VERIFICATION passed） |

### Top Lessons (Verified Across Milestones)

1. **atomic commit + 分级修复**可审计性高（v1.1 phase 17 验证）——延续到下个里程碑
2. **过程产物同步产出** > 事后补（v1.1 phase 21+22 反面教训）——下个里程碑强制 VERIFICATION 随 phase 执行
3. **外部服务依赖放最后**（v1.0 phase 4 经验）+ **安全加固独立里程碑**（v1.1 经验）——复杂度隔离
