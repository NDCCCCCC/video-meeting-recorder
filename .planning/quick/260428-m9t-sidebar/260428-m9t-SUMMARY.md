# Quick Task 260428-m9t: Sidebar Menu Updates - Summary

**Status:** Complete
**Date:** 2026-04-28

## Objective

Clean up header dropdown and add auth-config sidebar menu entry.

## Changes Made

### 1. frontend/src/layouts/BasicLayout.tsx

- Added `SafetyCertificateOutlined` icon import
- Removed "个人信息" (profile) menu item from user dropdown - now only shows "系统设置" and "退出登录"
- Changed "系统设置" dropdown key from 'settings' to '/system/settings' for proper routing
- Added `handleUserMenuClick` callback to handle dropdown navigation
- Updated Dropdown component to include `onClick` handler
- Added "认证管理" menu item to system management submenu (uses SafetyCertificateOutlined icon)
- Updated `hasSystemAccess` check to include `/system/auth-config` path
- Added `defaultOpenKeys` computed property to auto-expand system menu when on system pages

### 2. frontend/src/utils/permissions.ts

- Added `AUTH_CONFIG_VIEW` and `AUTH_CONFIG_EDIT` permission constants
- Added `/system/auth-config` to `MENU_PERMISSIONS` mapping
- Added `/system/auth-config` to `SUBMENU_PERMISSIONS` mapping

## Verification

- [x] TypeScript compilation passes for modified files
- [x] Header dropdown now shows only "系统设置" and "退出登录"
- [x] Clicking "系统设置" navigates to /system/settings
- [x] Sidebar "系统管理" submenu shows "认证管理" entry
- [x] System menu auto-expands when visiting /system/* pages

## Files Modified

- `frontend/src/layouts/BasicLayout.tsx`
- `frontend/src/utils/permissions.ts`

## Commit

Single atomic commit covering all changes.
