package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/docktop/docktop/internal/utils"
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
	path      string
	enabled   bool
	maxBytes  int64
	retention int
	mu        sync.Mutex
}

func New(enabled bool) (*Logger, error) { return NewWithOptions(enabled, "", 10, 5) }
func NewWithOptions(enabled bool, customPath string, maxSizeMB, retention int) (*Logger, error) {
	d, e := os.UserHomeDir()
	if e != nil {
		return nil, e
	}
	p := filepath.Join(d, ".local/share/docktop")
	path := filepath.Join(p, "audit.jsonl")
	if customPath != "" {
		path = customPath
		p = filepath.Dir(path)
	}
	if enabled {
		if e = os.MkdirAll(p, 0700); e != nil {
			return nil, e
		}
	}
	if maxSizeMB < 1 {
		maxSizeMB = 10
	}
	if retention < 1 {
		retention = 1
	}
	return &Logger{path: path, enabled: enabled, maxBytes: int64(maxSizeMB) << 20, retention: retention}, nil
}
func (l *Logger) Write(e Entry) error {
	if !l.enabled {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	e.Timestamp = time.Now().UTC()
	e.Error = utils.Sanitize(e.Error)
	if u, x := user.Current(); x == nil {
		e.User = u.Username
	}
	b, x := json.Marshal(e)
	if x != nil {
		return x
	}
	if x = l.rotate(); x != nil {
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

func (l *Logger) Read(limit int) ([]Entry, error) {
	if limit < 1 {
		limit = 500
	}
	f, err := os.Open(l.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := make([]Entry, 0, min(limit, 500))
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		var e Entry
		if json.Unmarshal(scanner.Bytes(), &e) == nil {
			out = append(out, e)
			if len(out) > limit {
				out = out[len(out)-limit:]
			}
		}
	}
	return out, scanner.Err()
}

func (l *Logger) rotate() error {
	info, err := os.Stat(l.path)
	if errorsIsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size() < l.maxBytes {
		return nil
	}
	_ = os.Remove(fmt.Sprintf("%s.%d", l.path, l.retention))
	for i := l.retention - 1; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", l.path, i)
		next := fmt.Sprintf("%s.%d", l.path, i+1)
		if err = os.Rename(old, next); err != nil && !errorsIsNotExist(err) {
			return err
		}
	}
	return os.Rename(l.path, l.path+".1")
}
func errorsIsNotExist(err error) bool { return err != nil && os.IsNotExist(err) }
