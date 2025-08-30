-- migrations/000001_create_initial_tables.down.sql

-- Drop triggers first
DROP TRIGGER IF EXISTS update_task_comments_updated_at ON task_comments;
DROP TRIGGER IF EXISTS update_time_entries_updated_at ON time_entries;
DROP TRIGGER IF EXISTS update_tasks_updated_at ON tasks;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_task_comments_created_at;
DROP INDEX IF EXISTS idx_task_comments_user_id;
DROP INDEX IF EXISTS idx_task_comments_task_id;

DROP INDEX IF EXISTS idx_project_members_project_id;
DROP INDEX IF EXISTS idx_project_members_user_id;

DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_token;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;

DROP INDEX IF EXISTS idx_time_entries_end_time;
DROP INDEX IF EXISTS idx_time_entries_start_time;
DROP INDEX IF EXISTS idx_time_entries_user_id;
DROP INDEX IF EXISTS idx_time_entries_task_id;

DROP INDEX IF EXISTS idx_tasks_created_at;
DROP INDEX IF EXISTS idx_tasks_due_date;
DROP INDEX IF EXISTS idx_tasks_priority;
DROP INDEX IF EXISTS idx_tasks_status;
DROP INDEX IF EXISTS idx_tasks_created_by;
DROP INDEX IF EXISTS idx_tasks_assignee_id;
DROP INDEX IF EXISTS idx_tasks_project_id;

DROP INDEX IF EXISTS idx_projects_created_at;
DROP INDEX IF EXISTS idx_projects_end_date;
DROP INDEX IF EXISTS idx_projects_start_date;
DROP INDEX IF EXISTS idx_projects_status;
DROP INDEX IF EXISTS idx_projects_created_by;
DROP INDEX IF EXISTS idx_projects_manager_id;

DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS task_comments CASCADE;
DROP TABLE IF EXISTS project_members CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS time_entries CASCADE;
DROP TABLE IF EXISTS tasks CASCADE;
DROP TABLE IF EXISTS projects CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop UUID extension if no other tables are using it
-- DROP EXTENSION IF EXISTS "uuid-ossp";
