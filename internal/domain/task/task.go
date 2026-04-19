package task

import (
	"time"
)

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Recurrence stores recurrence settings for periodic tasks. If nil, the task is one-off.
	Recurrence *Recurrence `json:"recurrence,omitempty"`
	// LastScheduledAt stores when the task's recurrence was last expanded.
	LastScheduledAt *time.Time `json:"last_scheduled_at,omitempty"`
	// ParentID for occurrences: points to original recurring task. Nil for parent tasks.
	ParentID *int64 `json:"parent_id,omitempty"`
	// DueDate for occurrences (date only, stored as time.Time with zeroed time component UTC).
	DueDate *time.Time `json:"due_date,omitempty"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

type RecurrenceType string

const (
	RecurrenceNone          RecurrenceType = "none"
	RecurrenceDaily         RecurrenceType = "daily"
	RecurrenceMonthly       RecurrenceType = "monthly"
	RecurrenceSpecificDates RecurrenceType = "specific_dates"
	RecurrenceEvenDays      RecurrenceType = "even"
	RecurrenceOddDays       RecurrenceType = "odd"
)

type Recurrence struct {
	Type RecurrenceType `json:"type"`
	// Daily: every N days (N >= 1)
	Interval int `json:"interval,omitempty"`
	// Monthly: list of month days (1..30)
	MonthDays []int `json:"month_days,omitempty"`
	// SpecificDates: list of ISO dates "YYYY-MM-DD"
	Dates []string `json:"dates,omitempty"`
	// Even/odd is represented by Type being "even" or "odd"; field kept for completeness
	EvenOdd string `json:"even_odd,omitempty"`
}

func (t RecurrenceType) Valid() bool {
	switch t {
	case RecurrenceNone, RecurrenceDaily, RecurrenceMonthly, RecurrenceSpecificDates, RecurrenceEvenDays, RecurrenceOddDays:
		return true
	default:
		return false
	}
}
