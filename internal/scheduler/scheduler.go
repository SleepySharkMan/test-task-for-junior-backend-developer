package scheduler

import (
	"context"
	"fmt"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskrepo "example.com/taskservice/internal/usecase/task"
)

type Scheduler struct {
	repo taskrepo.Repository
	now  func() time.Time
}

func New(repo taskrepo.Repository) *Scheduler {
	return &Scheduler{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// Run scans recurring tasks and creates occurrences for the next `daysAhead` days (including today).
// Pass daysAhead=1 to keep previous RunOnce behaviour.
func (s *Scheduler) Run(ctx context.Context, daysAhead int) (int, error) {
	if daysAhead < 1 {
		daysAhead = 1
	}

	tasks, err := s.repo.List(ctx)
	if err != nil {
		return 0, err
	}

	now := s.now().UTC()
	today := dateOnly(now)
	created := 0

	// For each parent task, attempt to create occurrences for the date window [today, today+daysAhead-1]
	for i := range tasks {
		t := tasks[i]
		if t.Recurrence == nil || t.Recurrence.Type == taskdomain.RecurrenceNone {
			continue
		}

		var lastScheduled dateOnlyWrapper
		if t.LastScheduledAt != nil {
			lastScheduled = dateOnlyWrapper{dateOnly(*t.LastScheduledAt)}
		}

		maxScheduled := lastScheduled.t

		for d := 0; d < daysAhead; d++ {
			target := dateOnly(today.AddDate(0, 0, d))

			// skip dates already scheduled
			if !maxScheduled.IsZero() && !maxScheduled.Before(target) {
				continue
			}

			if shouldScheduleForDate(&t, target) {
				// prepare occurrence with parent link and due date
				pid := t.ID
				d := target
				occ := &taskdomain.Task{
					Title:       t.Title,
					Description: t.Description,
					Status:      taskdomain.StatusNew,
					CreatedAt:   now,
					UpdatedAt:   now,
					ParentID:    &pid,
					DueDate:     &d,
				}

				if _, err := s.repo.Create(ctx, occ); err != nil {
					fmt.Println("scheduler: create occurrence error:", err)
					continue
				}

				// update maxScheduled to the latest date we've scheduled
				if maxScheduled.IsZero() || maxScheduled.Before(target) {
					maxScheduled = target
				}

				created++
			}
		}

		// if we created anything, persist LastScheduledAt = maxScheduled
		if !maxScheduled.IsZero() {
			t.LastScheduledAt = &maxScheduled
			if _, err := s.repo.Update(ctx, &t); err != nil {
				fmt.Println("scheduler: update parent last_scheduled_at error:", err)
			}
		}
	}

	return created, nil
}

// RunOnce kept for compatibility: create occurrences for today only
func (s *Scheduler) RunOnce(ctx context.Context) (int, error) {
	return s.Run(ctx, 1)
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Helper wrapper so we can check zero value conveniently
type dateOnlyWrapper struct{ t time.Time }

func (w dateOnlyWrapper) IsZero() bool { return w.t.IsZero() }

func shouldScheduleForDate(t *taskdomain.Task, target time.Time) bool {
	r := t.Recurrence
	switch r.Type {
	case taskdomain.RecurrenceDaily:
		interval := r.Interval
		if interval < 1 {
			interval = 1
		}

		var last time.Time
		if t.LastScheduledAt != nil {
			last = dateOnly(*t.LastScheduledAt)
		} else {
			last = dateOnly(t.CreatedAt)
		}

		// If target is before or equal to last scheduled, skip
		if !last.Before(target) {
			return false
		}

		// compute number of days between last (exclusive) and target (inclusive)
		days := int(target.Sub(last).Hours() / 24)
		return days%interval == 0

	case taskdomain.RecurrenceMonthly:
		if len(r.MonthDays) == 0 {
			return false
		}
		d := target.Day()
		for _, md := range r.MonthDays {
			if md == d {
				return true
			}
			// interpret 31 as last day of month when month has less than 31 days
			if md == 31 {
				firstOfNext := time.Date(target.Year(), target.Month()+1, 1, 0, 0, 0, 0, time.UTC)
				lastDay := firstOfNext.AddDate(0, 0, -1).Day()
				if d == lastDay {
					return true
				}
			}
		}
		return false

	case taskdomain.RecurrenceSpecificDates:
		targetStr := target.Format("2006-01-02")
		for _, ds := range r.Dates {
			if ds == targetStr {
				return true
			}
		}
		return false

	case taskdomain.RecurrenceEvenDays:
		return target.Day()%2 == 0
	case taskdomain.RecurrenceOddDays:
		return target.Day()%2 == 1
	default:
		return false
	}
}
