CREATE TABLE IF NOT EXISTS tasks (
	id BIGSERIAL PRIMARY KEY,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	-- Recurrence: type and JSON config for recurrence parameters
	recurrence_type TEXT NULL,
	recurrence_config JSONB NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status);
