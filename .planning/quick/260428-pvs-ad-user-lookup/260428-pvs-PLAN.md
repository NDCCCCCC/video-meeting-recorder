---
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/auth/ad_auth.go
  - internal/auth/ad_config.go
  - internal/handlers/admin_handler.go
  - cmd/server/app.go
  - frontend/src/api/auth.ts
  - frontend/src/types/auth.ts
  - frontend/src/pages/system/users/index.tsx
autonomous: true
requirements:
  - ad-user-lookup
---

<objective>
Add an "AD Lookup" button to the new/edit user modal that queries Active Directory by username and auto-fills full_name and email fields from the returned AD user info.

Purpose: When creating or editing users, admins can look up real user information from the domain controller instead of manually typing names and emails, reducing errors and ensuring consistency with AD data.

Output: Backend endpoint + frontend UI integration for AD user lookup with auto-fill.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md
</context>

<interfaces>
<!-- Existing contracts the executor needs -->

From internal/auth/ad_auth.go - ADAuthenticator.connectAD():
```go
func (a *ADAuthenticator) connectAD() (*ldap.Conn, error)
```
Connects to AD using LDAP or LDAPS based on config. Returns a connected (but unbound) ldap.Conn.

From internal/auth/ad_auth.go - ADAuthenticator.parseLDAPEntry():
```go
func (a *ADAuthenticator) parseLDAPEntry(entry *ldap.Entry) *ADUser
```
Parses LDAP entry fields: sAMAccountName, dn, objectGUID, mail, displayName, department, userPrincipalName, userAccountControl.

From internal/auth/ad_config.go - ADUser struct:
```go
type ADUser struct {
    Username           string
    DN                 string
    ObjectGUID         string
    Email              string
    DisplayName        string
    Department         string
    UserPrincipalName  string
    UserAccountControl uint32
}
```

From internal/auth/service.go - Service struct has:
```go
adAuth Authenticator  // This is *ADAuthenticator
```
Access via auth.Service for getting the AD config.

From internal/handlers/admin_handler.go - AdminHandler struct:
```go
type AdminHandler struct {
    cfg           *config.Config
    logger        *zap.Logger
    configService *services.ConfigService
}
```
Has access to cfg.Auth.AD for AD connection config.

From cmd/server/app.go - registerRoutes():
Admin routes use middleware.SM4Auth + middleware.RequireRole("admin"):
```go
adminGroup := a.router.Group("/api/v1/admin/auth")
adminGroup.Use(middleware.SM4Auth(a.tokenService), middleware.RequireRole(a.db, "admin"))
```

From frontend/src/api/apiClient.ts:
```typescript
export async function apiRequest<T>(url: string, options?: RequestInit): Promise<ApiResponse<T>>
```

From frontend/src/types/auth.ts - existing types for AD config.

From frontend/src/pages/system/users/index.tsx - User modal form fields:
- username (Form.Item name="username")
- email (Form.Item name="email")
- full_name (Form.Item name="full_name")
- Uses `form` from `Form.useForm()`
</interfaces>

<tasks>

<task type="auto">
  <name>Task 1: Backend - Add AD user lookup endpoint</name>
  <files>internal/auth/ad_auth.go, internal/auth/ad_config.go, internal/handlers/admin_handler.go, cmd/server/app.go</files>
  <action>
1. In `internal/auth/ad_config.go`, add a new response struct for the lookup:
```go
// ADUserLookupResult holds the result of an AD user lookup
type ADUserLookupResult struct {
    Found      bool   `json:"found"`
    Username   string `json:"username"`
    Email      string `json:"email,omitempty"`
    FullName   string `json:"full_name,omitempty"`
    Department string `json:"department,omitempty"`
    UPN        string `json:"upn,omitempty"`
    DN         string `json:"dn,omitempty"`
    Disabled   bool   `json:"disabled,omitempty"`
    Message    string `json:"message,omitempty"`
}
```

2. In `internal/auth/ad_auth.go`, add a public method `LookupUser(username string) (*ADUserLookupResult, error)` on `ADAuthenticator`. Reuse the existing `connectAD()` and `parseLDAPEntry()` methods. The flow:
   - Call `connectAD()` to get an LDAP connection
   - Bind with admin credentials (`a.adConfig.BindDN`, `a.adConfig.Password`)
   - Search using the same filter pattern as in Login: `(&(objectClass=user)(sAMAccountName={escaped_username}))`
   - Request attributes: `mail`, `displayName`, `department`, `userPrincipalName`, `userAccountControl`
   - If no entries found, return `Found: false` with a friendly message
   - If found, parse via `parseLDAPEntry()` and populate `ADUserLookupResult`
   - Close the connection via defer
   - Use `ldap.EscapeFilter(username)` to prevent LDAP injection (same pattern as Login)

3. In `internal/handlers/admin_handler.go`, add a new handler method `LookupADUser` on `AdminHandler`:
   - Accept POST with JSON body `{ "username": "string" }`
   - Validate username is non-empty (binding:"required,min=1,max=100")
   - Read current auth mode from `h.cfg.Auth.Mode`. If mode is not "ad", return error "当前认证模式不是AD域控模式"
   - Create an `ADAuthenticator` or better: the AdminHandler needs access to the auth service or AD authenticator. The cleanest approach: add a field `authService *auth.Service` to `AdminHandler`, and add a getter method on `auth.Service` like `GetADAuthenticator() *ADAuthenticator` that returns the underlying adAuth (nil if not available). Then:
     - Get the AD authenticator via the service
     - Call `adAuth.LookupUser(username)`
     - Return the result as JSON via `response.GinSuccess(c, result)`
   - Update `NewAdminHandler` to accept and store the auth service. Update the call site in `cmd/server/app.go` in `initHandlers()`.

4. In `cmd/server/app.go`, in `registerRoutes()`, add the new route inside the existing `adminGroup` block:
```go
adminGroup.POST("/lookup-ad-user", a.handlers.Admin.LookupADUser)
```

5. Also update the `NewAdminHandler` call in `initHandlers()` to pass the `authService`:
```go
Admin: handlers.NewAdminHandler(a.config, a.logger, configService, authService),
```
Where `authService` is the local variable created earlier: `authService := auth.NewService(a.config, a.db, a.logger)`. Note this variable is already created on line 555.
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/record_V2 && go build ./cmd/server/ && echo "Build successful"</automated>
  </verify>
  <done>POST /api/v1/admin/auth/lookup-ad-user endpoint returns AD user info (email, full_name, department) when given a valid username, returns found:false when user not found, returns error when not in AD mode. Build compiles without errors.</done>
</task>

<task type="auto">
  <name>Task 2: Frontend - Add lookup button and auto-fill in user modal</name>
  <files>frontend/src/api/auth.ts, frontend/src/types/auth.ts, frontend/src/pages/system/users/index.tsx</files>
  <action>
1. In `frontend/src/types/auth.ts`, add the AD user lookup types:
```typescript
// AD User Lookup
export interface ADUserLookupRequest {
  username: string
}

export interface ADUserLookupResult {
  found: boolean
  username: string
  email?: string
  full_name?: string
  department?: string
  upn?: string
  dn?: string
  disabled?: boolean
  message?: string
}
```

2. In `frontend/src/api/auth.ts`, add the API function at the end of the file (in the AD section):
```typescript
/**
 * Lookup AD user by username
 */
export async function lookupADUser(
  username: string
): Promise<ApiResponse<ADUserLookupResult>> {
  return apiRequest<ADUserLookupResult>('/api/v1/admin/auth/lookup-ad-user', {
    method: 'POST',
    body: JSON.stringify({ username }),
  })
}
```
Import `ADUserLookupResult` from types/auth in the import block at the top.

3. In `frontend/src/pages/system/users/index.tsx`:

   a. Add imports:
   - Add `SearchOutlined` is already imported. Add `UserOutlined` from `@ant-design/icons` (for the lookup button icon).
   - Add `lookupADUser` to the import from `'../../../api/auth'`.

   b. Add state for lookup loading:
   ```typescript
   const [adLookupLoading, setAdLookupLoading] = useState(false)
   ```

   c. Add the lookup handler function (after `closeModal`):
   ```typescript
   // AD用户查找
   const handleADLookup = async () => {
     const username = form.getFieldValue('username')
     if (!username || username.trim() === '') {
       message.warning('请先输入用户名')
       return
     }
     setAdLookupLoading(true)
     try {
       const response = await lookupADUser(username.trim())
       if (response.data?.found) {
         const adUser = response.data
         // Auto-fill fields only if they are currently empty
         if (!form.getFieldValue('full_name') && adUser.full_name) {
           form.setFieldsValue({ full_name: adUser.full_name })
         }
         if (!form.getFieldValue('email') && adUser.email) {
           form.setFieldsValue({ email: adUser.email })
         }
         message.success(`已找到AD用户: ${adUser.full_name || adUser.username}${adUser.department ? ' (' + adUser.department + ')' : ''}`)
       } else {
         message.info(response.data?.message || '未在域控中找到该用户')
       }
     } catch (error) {
       message.error(error instanceof Error ? error.message : '域控查询失败')
     } finally {
       setAdLookupLoading(false)
     }
   }
   ```

   d. Modify the username Form.Item in the modal. Add a lookup button next to the username input. Change the username Form.Item to use `addonAfter` or better: use an `Input.Group` / `Space` approach. The cleanest approach with Ant Design is to use `Input` with an `addonAfter` button:
   ```tsx
   <Form.Item
     name="username"
     label="用户名"
     rules={[
       { required: true, message: '请输入用户名' },
       { min: 3, max: 50, message: '用户名长度为3-50个字符' },
     ]}
   >
     <Input
       placeholder="请输入用户名"
       disabled={!!editingUser}
       addonAfter={
         <Button
           type="link"
           size="small"
           icon={<UserOutlined />}
           loading={adLookupLoading}
           onClick={handleADLookup}
           style={{ padding: 0, height: 'auto', border: 'none' }}
           title="从域控查找用户信息"
         >
           查找
         </Button>
       }
     />
   </Form.Item>
   ```
   Note: For the `disabled={!!editingUser}` case, the lookup button should still work (since we want to look up AD info even when editing). The username field itself is disabled during edit but the addonAfter button should remain clickable. However, when the Input is disabled, addonAfter buttons are also visually dimmed. So use a different approach: wrap the Input in a Space or use `suffix` instead. Actually the simplest approach that works well: use `addonAfter` but conditionally -- when editing, the button can be placed differently. A cleaner approach is to add the button as a separate element below the input or next to the label. Let me use a simpler approach:

   Replace the username Form.Item with:
   ```tsx
   <Form.Item
     name="username"
     label="用户名"
     rules={[
       { required: true, message: '请输入用户名' },
       { min: 3, max: 50, message: '用户名长度为3-50个字符' },
     ]}
   >
     <Space.Compact style={{ width: '100%' }}>
       <Input
         placeholder="请输入用户名"
         disabled={!!editingUser}
         style={{ width: '100%' }}
       />
       <Button
         icon={<UserOutlined />}
         loading={adLookupLoading}
         onClick={handleADLookup}
         title="从域控查找用户信息"
       >
         AD查找
       </Button>
     </Space.Compact>
   </Form.Item>
   ```
   Note: `Space.Compact` makes the Input and Button visually connected. Import `Space` is already in the imports.

   The button is always clickable (not disabled by editingUser) because even when editing an existing user, you might want to refresh their AD info.
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/record_V2/frontend && npx tsc --noEmit --pretty 2>&1 | head -30 && echo "TypeScript check complete"</automated>
  </verify>
  <done>User modal has an "AD查找" button next to the username field. Clicking it with a valid username queries the backend AD lookup endpoint and auto-fills full_name and email fields if found. Shows success message with user info or info message when not found. Shows error on failure. Button shows loading state during request.</done>
</task>

</tasks>

<verification>
1. Backend compiles: `go build ./cmd/server/`
2. Frontend compiles: `cd frontend && npx tsc --noEmit`
3. Manual: Open user management, click "新建用户", type an AD username, click "AD查找", verify name/email auto-fill
4. Manual: Edit existing user, click "AD查找", verify info refreshes
</verification>

<success_criteria>
- POST /api/v1/admin/auth/lookup-ad-user returns AD user info for valid usernames
- Endpoint returns found:false for non-existent users
- Endpoint returns error when auth mode is not "ad"
- User modal has "AD查找" button next to username field
- Clicking lookup auto-fills full_name and email from AD data
- Lookup works in both create and edit user modes
- Backend and frontend compile without errors
</success_criteria>

<output>
After completion, create `.planning/quick/260428-pvs-ad-user-lookup/260428-pvs-SUMMARY.md`
</output>
