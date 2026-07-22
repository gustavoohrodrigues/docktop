package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type State string

const (
	Queued    State = "queued"
	Running   State = "running"
	Succeeded State = "succeeded"
	Failed    State = "failed"
	Cancelled State = "cancelled"
)

type Job struct {
	ID, Type, Resource string
	State              State
	Progress           float64
	Message, Error     string
	Started, Finished  time.Time
}
type Runner func(context.Context, func(float64, string)) error
type Manager struct {
	mu      sync.RWMutex
	jobs    map[string]Job
	cancels map[string]context.CancelFunc
	seq     atomic.Uint64
	limit   chan struct{}
}

func New(concurrency int) *Manager {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Manager{jobs: map[string]Job{}, cancels: map[string]context.CancelFunc{}, limit: make(chan struct{}, concurrency)}
}
func (m *Manager) Start(parent context.Context, kind, resource string, run Runner) string {
	id := fmt.Sprintf("job-%06d", m.seq.Add(1))
	ctx, cancel := context.WithCancel(parent)
	m.mu.Lock()
	m.jobs[id] = Job{ID: id, Type: kind, Resource: resource, State: Queued}
	m.cancels[id] = cancel
	m.mu.Unlock()
	go func() {
		select {
		case m.limit <- struct{}{}:
		case <-ctx.Done():
			m.finish(id, Cancelled, ctx.Err())
			return
		}
		defer func() { <-m.limit }()
		m.update(id, func(j *Job) { j.State = Running; j.Started = time.Now() })
		err := run(ctx, func(p float64, msg string) {
			m.update(id, func(j *Job) {
				if p < 0 {
					p = 0
				}
				if p > 1 {
					p = 1
				}
				j.Progress = p
				j.Message = msg
			})
		})
		state := Succeeded
		if errors.Is(err, context.Canceled) {
			state = Cancelled
		} else if err != nil {
			state = Failed
		}
		m.finish(id, state, err)
	}()
	return id
}
func (m *Manager) Cancel(id string) bool {
	m.mu.RLock()
	c, ok := m.cancels[id]
	m.mu.RUnlock()
	if ok {
		c()
	}
	return ok
}
func (m *Manager) Get(id string) (Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[id]
	return j, ok
}
func (m *Manager) List() []Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j)
	}
	return out
}
func (m *Manager) Active() int {
	n := 0
	for _, j := range m.List() {
		if j.State == Queued || j.State == Running {
			n++
		}
	}
	return n
}
func (m *Manager) update(id string, fn func(*Job)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j := m.jobs[id]
	fn(&j)
	m.jobs[id] = j
}
func (m *Manager) finish(id string, state State, err error) {
	m.update(id, func(j *Job) {
		j.State = state
		j.Finished = time.Now()
		if err != nil {
			j.Error = err.Error()
		}
	})
	m.mu.Lock()
	delete(m.cancels, id)
	m.mu.Unlock()
}
