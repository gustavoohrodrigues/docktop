package audit

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Timestamp                                   time.Time `json:"timestamp"`
	User, Host, Action, Resource, Result, Error string    `json:",omitempty"`
}
type Logger struct {
	path    string
	enabled bool
	mu      sync.Mutex
}

func New(enabled bool) (*Logger, error) {
	d, e := os.UserHomeDir()
	if e != nil {
		return nil, e
	}
	p := filepath.Join(d, ".local/share/docktop")
	if enabled {
		if e = os.MkdirAll(p, 0700); e != nil {
			return nil, e
		}
	}
	return &Logger{filepath.Join(p, "audit.jsonl"), enabled, sync.Mutex{}}, nil
}
func (l *Logger) Write(e Entry) error {
	if !l.enabled {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	e.Timestamp = time.Now().UTC()
	if u, x := user.Current(); x == nil {
		e.User = u.Username
	}
	b, x := json.Marshal(e)
	if x != nil {
		return x
	}
	f, x := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if x != nil {
		return x
	}
	defer f.Close()
	_, x = f.Write(append(b, '\n'))
	return x
}
