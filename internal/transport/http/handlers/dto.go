package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	Recurrence  *recurrenceDTO    `json:"recurrence,omitempty"`
	DueDate     *time.Time        `json:"due_date,omitempty"`
}

type taskDTO struct {
	ID              int64             `json:"id"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Status          taskdomain.Status `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Recurrence      *recurrenceDTO    `json:"recurrence,omitempty"`
	LastScheduledAt *time.Time        `json:"last_scheduled_at,omitempty"`
	ParentID        *int64            `json:"parent_id,omitempty"`
	DueDate         *time.Time        `json:"due_date,omitempty"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	return taskDTO{
		ID:              task.ID,
		Title:           task.Title,
		Description:     task.Description,
		Status:          task.Status,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
		Recurrence:      toRecurrenceDTO(task.Recurrence),
		LastScheduledAt: task.LastScheduledAt,
		ParentID:        task.ParentID,
		DueDate:         task.DueDate,
	}
}

type recurrenceDTO struct {
	Type      taskdomain.RecurrenceType `json:"type"`
	Interval  int                       `json:"interval,omitempty"`
	MonthDays []int                     `json:"month_days,omitempty"`
	Dates     []string                  `json:"dates,omitempty"`
	EvenOdd   string                    `json:"even_odd,omitempty"`
}

func toRecurrenceDTO(r *taskdomain.Recurrence) *recurrenceDTO {
	if r == nil {
		return nil
	}

	return &recurrenceDTO{
		Type:      r.Type,
		Interval:  r.Interval,
		MonthDays: r.MonthDays,
		Dates:     r.Dates,
		EvenOdd:   r.EvenOdd,
	}
}

func toDomainRecurrence(r *recurrenceDTO) *taskdomain.Recurrence {
	if r == nil {
		return nil
	}
	return &taskdomain.Recurrence{
		Type:      r.Type,
		Interval:  r.Interval,
		MonthDays: r.MonthDays,
		Dates:     r.Dates,
		EvenOdd:   r.EvenOdd,
	}
}
