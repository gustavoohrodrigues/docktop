package jobs

import (
	"context"
	"testing"
	"time"
)

func TestJobCompletes(t *testing.T) {
	m := New(1)
	id := m.Start(context.Background(), "pull", "nginx", func(_ context.Context, p func(float64, string)) error { p(1, "ok"); return nil })
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		j, _ := m.Get(id)
		if j.State == Succeeded {
			if j.Progress != 1 {
				t.Fatal(j.Progress)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("job não terminou")
}
