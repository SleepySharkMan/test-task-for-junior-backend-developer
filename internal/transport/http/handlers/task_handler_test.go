package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
	"github.com/gorilla/mux"
)

// custom mock to return given values
type mockOccUsecase struct{
    created *taskdomain.Task
    err error
}
func (m *mockOccUsecase) Create(ctx context.Context, input taskusecase.CreateInput) (*taskdomain.Task, error) { return nil, nil }
func (m *mockOccUsecase) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) { return nil, nil }
func (m *mockOccUsecase) Update(ctx context.Context, id int64, input taskusecase.UpdateInput) (*taskdomain.Task, error) { return nil, nil }
func (m *mockOccUsecase) Delete(ctx context.Context, id int64) error { return nil }
func (m *mockOccUsecase) List(ctx context.Context) ([]taskdomain.Task, error) { return nil, nil }
func (m *mockOccUsecase) CreateOccurrence(ctx context.Context, parentID int64, input taskusecase.OccurrenceInput) (*taskdomain.Task, error) {
    return m.created, m.err
}

func TestCreateOccurrenceHandler_Success(t *testing.T) {
    occ := &taskdomain.Task{ID: 123}
    pid := int64(1)
    d, _ := time.Parse(time.RFC3339, "2026-05-01T00:00:00Z")
    occ.ParentID = &pid
    occ.DueDate = &d

    mu := &mockOccUsecase{created: occ, err: nil}
    h := NewTaskHandler(mu)

    body := map[string]interface{}{"title":"occ","description":"d","status":"new","due_date":"2026-05-01T00:00:00Z"}
    b, _ := json.Marshal(body)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/1/occurrences", bytes.NewReader(b))
    req = mux.SetURLVars(req, map[string]string{"parent_id":"1"})
    req.Header.Set("Content-Type","application/json")
    rr := httptest.NewRecorder()

    h.CreateOccurrence(rr, req)

    if rr.Code != http.StatusCreated {
        t.Fatalf("expected 201 created, got %d body=%s", rr.Code, rr.Body.String())
    }
}

func TestCreateOccurrenceHandler_Duplicate(t *testing.T) {
    mu := &mockOccUsecase{created: nil, err: taskdomain.ErrAlreadyExists}
    h := NewTaskHandler(mu)

    body := map[string]interface{}{"title":"occ","description":"d","status":"new","due_date":"2026-05-01T00:00:00Z"}
    b, _ := json.Marshal(body)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/1/occurrences", bytes.NewReader(b))
    req = mux.SetURLVars(req, map[string]string{"parent_id":"1"})
    req.Header.Set("Content-Type","application/json")
    rr := httptest.NewRecorder()

    h.CreateOccurrence(rr, req)

    if rr.Code != http.StatusBadRequest {
        t.Fatalf("expected 400 bad request for duplicate, got %d body=%s", rr.Code, rr.Body.String())
    }
}

// helper to set mux vars on request for handler tests
// no helper needed; use mux.SetURLVars
