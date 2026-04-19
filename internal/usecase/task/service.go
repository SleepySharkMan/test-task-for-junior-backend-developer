package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		Recurrence:  normalized.Recurrence,
		DueDate:     normalized.DueDate,
	}
	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		UpdatedAt:   s.now(),
		Recurrence:  normalized.Recurrence,
		DueDate:     normalized.DueDate,
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

// CreateOccurrence creates an occurrence task linked to a parent recurring task.
func (s *Service) CreateOccurrence(ctx context.Context, parentID int64, input OccurrenceInput) (*taskdomain.Task, error) {
	if parentID <= 0 {
		return nil, fmt.Errorf("%w: parent id must be positive", ErrInvalidInput)
	}

	parent, err := s.repo.GetByID(ctx, parentID)
	if err != nil {
		if errors.Is(err, taskdomain.ErrNotFound) {
			return nil, fmt.Errorf("%w: parent not found", ErrInvalidInput)
		}
		return nil, err
	}

	if parent.Recurrence == nil || parent.Recurrence.Type == taskdomain.RecurrenceNone {
		return nil, fmt.Errorf("%w: parent task is not recurring", ErrInvalidInput)
	}

	if input.DueDate == nil {
		return nil, fmt.Errorf("%w: due_date is required for occurrence", ErrInvalidInput)
	}

	// normalize due date to date-only UTC
	d := time.Date(input.DueDate.Year(), input.DueDate.Month(), input.DueDate.Day(), 0, 0, 0, 0, time.UTC)

	model := &taskdomain.Task{
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		Status:      input.Status,
		CreatedAt:   s.now(),
		UpdatedAt:   s.now(),
		ParentID:    &parentID,
		DueDate:     &d,
	}

	if model.Title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		if errors.Is(err, taskdomain.ErrAlreadyExists) || strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "ux_tasks_parent_due_date") || strings.Contains(err.Error(), "23505") {
			return nil, fmt.Errorf("%w: occurrence already exists", ErrInvalidInput)
		}
		return nil, err
	}

	return created, nil
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if input.Recurrence != nil {
		if err := validateRecurrence(input.Recurrence); err != nil {
			return CreateInput{}, err
		}
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if input.Recurrence != nil {
		if err := validateRecurrence(input.Recurrence); err != nil {
			return UpdateInput{}, err
		}
	}

	return input, nil
}

func validateRecurrence(r *taskdomain.Recurrence) error {
	if r.Type == "" {
		return fmt.Errorf("%w: recurrence type is required", ErrInvalidInput)
	}

	if !r.Type.Valid() {
		return fmt.Errorf("%w: invalid recurrence type", ErrInvalidInput)
	}

	switch r.Type {
	case taskdomain.RecurrenceDaily:
		if r.Interval < 1 {
			return fmt.Errorf("%w: interval must be >= 1 for daily recurrence", ErrInvalidInput)
		}
	case taskdomain.RecurrenceMonthly:
		if len(r.MonthDays) == 0 {
			return fmt.Errorf("%w: month_days must be provided for monthly recurrence", ErrInvalidInput)
		}
		for _, d := range r.MonthDays {
			if d < 1 || d > 31 {
				return fmt.Errorf("%w: month_days must be between 1 and 31", ErrInvalidInput)
			}
		}
	case taskdomain.RecurrenceSpecificDates:
		if len(r.Dates) == 0 {
			return fmt.Errorf("%w: dates must be provided for specific_dates recurrence", ErrInvalidInput)
		}
		for _, ds := range r.Dates {
			if _, err := time.Parse("2006-01-02", ds); err != nil {
				return fmt.Errorf("%w: invalid date format in dates, expected YYYY-MM-DD", ErrInvalidInput)
			}
		}
	case taskdomain.RecurrenceEvenDays, taskdomain.RecurrenceOddDays:
		// no extra params required
	}

	return nil
}
