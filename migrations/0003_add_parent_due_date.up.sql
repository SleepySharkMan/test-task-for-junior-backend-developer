ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS parent_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS due_date DATE NULL;

-- Unique index to prevent duplicate occurrences for the same parent+due_date
CREATE UNIQUE INDEX IF NOT EXISTS ux_tasks_parent_due_date ON tasks (parent_id, due_date) WHERE parent_id IS NOT NULL AND due_date IS NOT NULL;

-- Optionally add a foreign key (no cascade)
ALTER TABLE tasks
    ADD CONSTRAINT IF NOT EXISTS fk_tasks_parent FOREIGN KEY (parent_id) REFERENCES tasks(id);
