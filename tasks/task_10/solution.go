package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrMethodNotAllowed = errors.New("method not allowed")
	ErrNotFound         = errors.New("task not found")
	ErrInvalidTitle     = errors.New("invalid title")
	ErrSecondObject     = errors.New("an extra object in payload")
	ErrBadRequest       = errors.New("sent request is incorrect")
	ErrInternal         = errors.New("something went wrong")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{clock: clock, tasks: map[string]Task{}}
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {

	clearTitle := strings.TrimSpace(title)
	task := Task{Done: false}

	if len(clearTitle) == 0 {
		return task, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	task.Title = clearTitle
	task.ID = strconv.FormatUint(r.seq+1, 10)
	task.UpdatedAt = r.clock.Now()
	r.tasks[task.ID] = task
	r.seq += 1

	return task, nil

}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	if task, ok := r.tasks[id]; ok {
		return task, true
	}
	return Task{}, false
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := []Task{}

	for _, task := range r.tasks {
		if task.Done == done {
			results = append(results, task)
		}
	}

	return results
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]

	if ok {
		if task.Done != done {
			task.UpdatedAt = r.clock.Now()
		}
		task.Done = done
		r.tasks[id] = task
		return task, nil
	}

	return task, ErrNotFound

}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	switch {
	case len(pathParts) == 2 && pathParts[1] == "tasks":
		switch r.Method {
		case "POST":
			h.handleCreate(w, r)
		case "GET":
			h.handleList(w, r)
		default:
			http.Error(w, ErrMethodNotAllowed.Error(), http.StatusMethodNotAllowed)
			return
		}
	case len(pathParts) == 3 && pathParts[1] == "tasks" && pathParts[2] != "":
		switch r.Method {
		case "GET":
			h.handleGet(w, r, pathParts[2])
		case "PATCH":
			h.handlePatch(w, r, pathParts[2])
		default:
			http.Error(w, ErrMethodNotAllowed.Error(), http.StatusMethodNotAllowed)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {

	var taskTitle struct {
		Title *string `json:"title"`
	}

	if err := decodeStrictJSON(r.Body, &taskTitle); err != nil {
		http.Error(w, ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}

	if taskTitle.Title == nil {
		http.Error(w, ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}

	task, errCreatedTask := h.repo.Create(*taskTitle.Title)

	if errCreatedTask != nil {
		if errors.Is(errCreatedTask, ErrInvalidTitle) {
			http.Error(w, ErrInvalidTitle.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, ErrInternal.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, task)

}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, ok := h.repo.Get(id)

	if !ok {
		http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, task)

}

func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	var paramDone string = "false"

	if len(params["done"]) > 1 {
		http.Error(w, ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}

	if params.Has("done") {
		paramDone = params.Get("done")
	}

	if paramDone == "" {
		http.Error(w, ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}

	if paramDone != "true" && paramDone != "false" {
		http.Error(w, ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}

	paramDoneBool := (paramDone == "true")

	tasks := h.repo.List(paramDoneBool)

	if tasks == nil {
		tasks = []Task{}
	}

	slices.SortFunc(tasks, sortingTasks)

	writeJSON(w, http.StatusOK, tasks)

}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {

	var doneTask struct {
		Done *bool `json:"done"`
	}

	if err := decodeStrictJSON(r.Body, &doneTask); err != nil {
		http.Error(w, ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}

	if doneTask.Done == nil {
		http.Error(w, ErrBadRequest.Error(), http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *doneTask.Done)

	if err != nil {

		if errors.Is(err, ErrNotFound) {
			http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, ErrInternal.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, task)

}

func decodeStrictJSON(r io.Reader, v any) error {

	dec := json.NewDecoder(r)
	checkIncorrectPayload := map[string]string{}

	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}

	errIncorrectPayload := dec.Decode(&checkIncorrectPayload)

	if errIncorrectPayload == io.EOF {
		return nil
	}

	if errIncorrectPayload != nil {
		return errIncorrectPayload
	}

	return ErrSecondObject
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func sortingTasks(t1, t2 Task) int {
	diffUpdatedAt := t1.UpdatedAt.Compare(t2.UpdatedAt)
	if diffUpdatedAt == 0 {
		return cmp.Compare(t1.ID, t2.ID)
	}
	return -diffUpdatedAt
}
