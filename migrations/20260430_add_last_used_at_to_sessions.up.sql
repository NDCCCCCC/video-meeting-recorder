-- 添加 last_used_at 字段到 sessions 表
-- 用于支持 token 刷新的宽限期机制

ALTER TABLE sessions ADD COLUMN last_used_at DATETIME NULL;

-- 添加索引以支持宽限期查询
CREATE INDEX idx_sessions_user_id_created_at ON sessions(user_id, created_at);

-- 添加注释
COMMENT ON COLUMN sessions.last_used_at IS 'Token 最后使用时间，用于宽限期机制判断';
