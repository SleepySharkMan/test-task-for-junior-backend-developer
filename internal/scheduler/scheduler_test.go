package scheduler

import (
	"context"
	"reflect"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

// mockRepo is a tiny in-memory repository used for scheduler tests.
type mockRepo struct {
    tasks []*taskdomain.Task
    next  int64
}

func newMockRepo(parents []*taskdomain.Task) *mockRepo {
    r := &mockRepo{next: 1}
    for _, t := range parents {
        t.ID = r.next
        r.next++
        r.tasks = append(r.tasks, t)
    }
    return r
}

func (r *mockRepo) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
    task.ID = r.next
    r.next++
    // copy
    copyTask := *task
    r.tasks = append(r.tasks, &copyTask)
    return &copyTask, nil
}

func (r *mockRepo) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
    for _, t := range r.tasks {
        if t.ID == id {
            return t, nil
        }
    }
    return nil, nil
}

func (r *mockRepo) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
    for i, t := range r.tasks {
        if t.ID == task.ID {
            r.tasks[i] = task
            return task, nil
        }
    }
    return nil, nil
}

func (r *mockRepo) Delete(ctx context.Context, id int64) error { return nil }

func (r *mockRepo) List(ctx context.Context) ([]taskdomain.Task, error) {
    out := make([]taskdomain.Task, 0, len(r.tasks))
    for _, t := range r.tasks {
        out = append(out, *t)
    }
    return out, nil
}

func mustDate(s string) time.Time {
    t, _ := time.Parse("2006-01-02", s)
    return t
}

func TestShouldScheduleForDate(t *testing.T) {
    // daily every 2 days, parent created 2026-04-01
    parent := &taskdomain.Task{
        CreatedAt: mustDate("2026-04-01"),
        Recurrence: &taskdomain.Recurrence{
            Type:     taskdomain.RecurrenceDaily,
            Interval: 2,
        },
    }

    // last scheduled nil -> base is CreatedAt
    if !shouldScheduleForDate(parent, mustDate("2026-04-03")) {
        t.Fatalf("expected schedule on 2026-04-03")
    }
    if shouldScheduleForDate(parent, mustDate("2026-04-02")) {
        t.Fatalf("did not expect schedule on 2026-04-02")
    }

    // monthly
    parent2 := &taskdomain.Task{
        CreatedAt: mustDate("2026-04-01"),
        Recurrence: &taskdomain.Recurrence{
            Type:      taskdomain.RecurrenceMonthly,
            MonthDays: []int{1,15},
        },
    }
    if !shouldScheduleForDate(parent2, mustDate("2026-04-15")) {
        t.Fatalf("expected schedule on 15")
    }

    // specific dates
    parent3 := &taskdomain.Task{
        Recurrence: &taskdomain.Recurrence{
            Type:  taskdomain.RecurrenceSpecificDates,
            Dates: []string{"2026-05-01"},
        },
    }
    if !shouldScheduleForDate(parent3, mustDate("2026-05-01")) {
        t.Fatalf("expected schedule on specific date")
    }

    // even/odd
    p4 := &taskdomain.Task{Recurrence: &taskdomain.Recurrence{Type: taskdomain.RecurrenceEvenDays}}
    if !shouldScheduleForDate(p4, mustDate("2026-04-02")) || shouldScheduleForDate(p4, mustDate("2026-04-03")) {
        t.Fatalf("even/odd logic failed")
    }
}

func TestRunCreatesOccurrencesWithParentAndDueDate(t *testing.T) {
    now := mustDate("2026-04-01")
    // parent with monthly on days 2 and 3
    parent := &taskdomain.Task{
        Title:     "parent",
        CreatedAt: now,
        Recurrence: &taskdomain.Recurrence{
            Type:      taskdomain.RecurrenceMonthly,
            MonthDays: []int{2, 3},
        },
    }

    repo := newMockRepo([]*taskdomain.Task{parent})
    s := New(repo)
    s.now = func() time.Time { return now }

    // create occurrences for next 3 days (2 and 3 should be created)
    created, err := s.Run(context.Background(), 3)
    if err != nil {
        t.Fatal(err)
    }
    if created != 2 {
        t.Fatalf("expected 2 occurrences created, got %d", created)
    }

    // find occurrences: should have ParentID set to parent's ID and DueDate set
    var occs []*taskdomain.Task
    for _, tsk := range repo.tasks {
        if tsk.ParentID != nil && *tsk.ParentID == parent.ID {
            occs = append(occs, tsk)
        }
    }
    if len(occs) != 2 {
        t.Fatalf("expected 2 occurrences in repo, got %d", len(occs))
    }

    // check due dates are 2026-04-02 and 2026-04-03
    dates := []string{occs[0].DueDate.Format("2006-01-02"), occs[1].DueDate.Format("2006-01-02")}
    expected := []string{"2026-04-02", "2026-04-03"}
    if !reflect.DeepEqual(expected, dates) && !reflect.DeepEqual([]string{dates[1], dates[0]}, expected) {
        t.Fatalf("due dates mismatch: %v", dates)
    }

    // parent should have LastScheduledAt set to last date
    p, _ := repo.GetByID(context.Background(), parent.ID)
    if p.LastScheduledAt == nil || p.LastScheduledAt.Format("2006-01-02") != "2026-04-03" {
        t.Fatalf("parent LastScheduledAt not updated, got %v", p.LastScheduledAt)
    }
}

func TestShouldScheduleMonthly31AsLastDay(t *testing.T) {
    // parent with monthly on 31
    parent := &taskdomain.Task{
        Recurrence: &taskdomain.Recurrence{
            Type:      taskdomain.RecurrenceMonthly,
            MonthDays: []int{31},
        },
    }

    // February 2026 has 28 days -> last day is 28
    if !shouldScheduleForDate(parent, mustDate("2026-02-28")) {
        t.Fatalf("expected schedule on 2026-02-28 for month_days=31 (last day)")
    }

    // April 2026 has 30 days -> last day 30, should schedule on 30
    if !shouldScheduleForDate(parent, mustDate("2026-04-30")) {
        t.Fatalf("expected schedule on 2026-04-30 for month_days=31 (last day)")
    }

    // May 2026 has 31 days -> schedule on 31
    if !shouldScheduleForDate(parent, mustDate("2026-05-31")) {
        t.Fatalf("expected schedule on 2026-05-31 for month_days=31")
    }
}
