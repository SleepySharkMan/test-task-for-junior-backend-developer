ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS last_scheduled_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_tasks_last_scheduled_at ON tasks (last_scheduled_at);
