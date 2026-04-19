package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, created_at, updated_at, parent_id, due_date, recurrence_type, recurrence_config, last_scheduled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, title, description, status, created_at, updated_at, parent_id, due_date, recurrence_type, recurrence_config, last_scheduled_at
	`

	var cfg []byte
	var rtype string
	var lastScheduled interface{}
	var parentID interface{}
	var dueDate interface{}
	if task.Recurrence != nil {
		b, err := json.Marshal(task.Recurrence)
		if err != nil {
			return nil, err
		}
		cfg = b
		rtype = string(task.Recurrence.Type)
	}

	if task.LastScheduledAt != nil {
		lastScheduled = *task.LastScheduledAt
	}

	if task.ParentID != nil {
		parentID = *task.ParentID
	}

	if task.DueDate != nil {
		// store date-only value
		dueDate = task.DueDate.Format("2006-01-02")
	}

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt, parentID, dueDate, rtype, cfg, lastScheduled)
	created, err := scanTask(row)
	if err != nil {
		// translate unique-violation into domain error
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return nil, taskdomain.ErrAlreadyExists
			}
		}
		return nil, err
	}

	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at, parent_id, due_date, recurrence_type, recurrence_config, last_scheduled_at
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			status = $3,
			updated_at = $4,
			recurrence_type = $5,
			recurrence_config = $6
			, last_scheduled_at = $7
			, parent_id = $8,
			due_date = $9
		WHERE id = $10
		RETURNING id, title, description, status, created_at, updated_at, parent_id, due_date, recurrence_type, recurrence_config, last_scheduled_at
	`

	var cfg []byte
	var rtype string
	var lastScheduled interface{}
	var parentID interface{}
	var dueDate interface{}
	if task.Recurrence != nil {
		b, err := json.Marshal(task.Recurrence)
		if err != nil {
			return nil, err
		}
		cfg = b
		rtype = string(task.Recurrence.Type)
	}

	if task.LastScheduledAt != nil {
		lastScheduled = *task.LastScheduledAt
	}

	if task.ParentID != nil {
		parentID = *task.ParentID
	}

	if task.DueDate != nil {
		dueDate = task.DueDate.Format("2006-01-02")
	}

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.UpdatedAt, rtype, cfg, lastScheduled, parentID, dueDate, task.ID)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at, parent_id, due_date, recurrence_type, recurrence_config, last_scheduled_at
		FROM tasks
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task          taskdomain.Task
		status        string
		rtype         *string
		cfg           []byte
		lastScheduled *time.Time
		parentID      *int64
		dueDate       *time.Time
	)

	// Try to scan with recurrence fields. Some queries might not include them,
	// so handle both possibilities by attempting the larger scan first.
	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
		&parentID,
		&dueDate,
		&rtype,
		&cfg,
		&lastScheduled,
	); err != nil {
		// Fallback to older shape without recurrence
		if err := scanner.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&status,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, err
		}
	}

	task.Status = taskdomain.Status(status)

	if rtype != nil && *rtype != "" && len(cfg) > 0 {
		var rec taskdomain.Recurrence
		if err := json.Unmarshal(cfg, &rec); err == nil {
			task.Recurrence = &rec
		}
	}

	if parentID != nil {
		task.ParentID = parentID
	}

	if dueDate != nil {
		task.DueDate = dueDate
	}

	if lastScheduled != nil {
		task.LastScheduledAt = lastScheduled
	}

	return &task, nil
}
