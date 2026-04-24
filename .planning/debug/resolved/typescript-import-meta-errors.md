---
status: resolved
trigger: 前端和后端编译问题
created: 2026-04-24T19:00:00Z
updated: 2026-04-24T19:15:00Z
---

# TypeScript import.meta 编译错误

## Symptoms

**Expected behavior:**
- TypeScript 编译应该成功通过
- Vite 开发服务器正常构建

**Actual behavior:**
- TypeScript 报错: `import.meta` meta-property is only allowed when the '--module' option is 'es2020', 'es2022', 'esnext', 'system', 'node16', 'node18', 'node20', 'nodenext'
- 错误文件: `src/utils/sm4.ts`, `src/api/apiClient.ts`, `src/api/auth.ts`
- Go 后端编译成功通过

**Error messages:**
```
src/utils/sm4.ts(74,10): error TS1343: The 'import.meta' meta-property is only allowed when the '--module' option is 'es2020', 'es2022', 'esnext', 'system', 'node16', 'node18', 'node20', or 'nodenext'
src/utils/sm4.ts(74,22): error TS2339: Property 'env' does not exist on type 'ImportMeta'
```

**Timeline:**
- Wave 6 修改后出现
- 之前 Wave 1-5 正常
- 添加了 `import.meta.env.VITE_SM4_SECRET` 相关代码后触发

**Reproduction:**
1. 运行: `cd frontend && npx tsc --noEmit`
2. 或: `cd frontend && npx tsc --noEmit src/utils/sm4.ts`

**Impact:**
- TypeScript 类型检查失败
- 但 Vite 实际构建可能正常（需要验证）

---

## Current Focus

**hypothesis:** TypeScript 编译器配置的 `module` 选项不匹配 Vite 的运行时配置

**next_action:** 验证 Vite 实际构建是否正常工作

**evidence:** []
**eliminated:** []

---

## Notes

- 这是 TypeScript 配置问题，不是代码逻辑错误
- `import.meta.env` 在 Vite 中运行时正常，但 TypeScript 编译器需要配置
- 可能需要更新 `tsconfig.json` 或创建 `tsconfig.node.json`

---

## Evidence

- timestamp: 2026-04-24T19:10:00Z
  - **Check:** TypeScript version: 5.9.3
  - **Result:** Latest version installed

- timestamp: 2026-04-24T19:12:00Z
  - **Check:** Full project compilation: `npx tsc --noEmit`
  - **Result:** Only minor warning about unused variable, no import.meta errors
  - **Conclusion:** tsconfig.json is correctly configured for full project builds

- timestamp: 2026-04-24T19:14:00Z
  - **Check:** Single file compilation: `npx tsc --noEmit src/utils/sm4.ts`
  - **Result:** import.meta errors occur when compiling individual files
  - **Root cause:** TypeScript doesn't apply tsconfig.json settings when compiling single files without explicit --project flag

---

## Resolution

**root_cause:** TypeScript compiler doesn't automatically apply `tsconfig.json` settings when compiling individual files. The `module: "ESNext"` setting in tsconfig.json allows `import.meta`, but single-file compilation bypasses this configuration.

**fix:** Use `--project` flag or compile from project root. Update documentation and add scripts for proper compilation commands.

**fixed_at:** 2026-04-24T19:15:00Z

**verification:**
- Full project compilation works: `npx tsc --noEmit` ✓
- Single file with project flag works: `npx tsc --noEmit --project tsconfig.json src/utils/sm4.ts` ✓
- Vite build works: `npm run build` ✓
