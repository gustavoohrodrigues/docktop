package app

import (
	"context"
	"fmt"
	"github.com/docktop/docktop/internal/audit"
	"github.com/docktop/docktop/internal/config"
	dock "github.com/docktop/docktop/internal/docker"
	"github.com/docktop/docktop/internal/ui"
)

func New(ctx context.Context, c config.Config, version string, debug bool) (*ui.Model, func(), error) {
	cc, ok := c.Contexts[c.DefaultContext]
	if !ok {
		return nil, nil, fmt.Errorf("contexto %q não configurado", c.DefaultContext)
	}
	e, err := dock.New(cc)
	if err != nil {
		return nil, nil, err
	}
	a, err := audit.NewWithOptions(c.Audit.Enabled, c.Audit.Path, c.Audit.MaxSizeMB, c.Audit.Retention)
	if err != nil {
		e.Close()
		return nil, nil, err
	}
	m := ui.New(ctx, c, e, a, version)
	return m, func() { e.Close() }, nil
}
