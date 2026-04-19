package task

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	CreateOccurrence(ctx context.Context, parentID int64, input OccurrenceInput) (*taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
}

type CreateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
	Recurrence  *taskdomain.Recurrence
	DueDate     *time.Time
}

type UpdateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
	Recurrence  *taskdomain.Recurrence
	DueDate     *time.Time
}

type OccurrenceInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
	DueDate     *time.Time
}
