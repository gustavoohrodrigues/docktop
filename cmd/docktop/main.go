package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docktop/docktop/internal/app"
	"github.com/docktop/docktop/internal/config"
)

var version = "dev"

func main() {
	var path, contextName, theme string
	var readOnly, noMouse, debug, showVersion bool
	flag.StringVar(&path, "config", "", "arquivo de configuração")
	flag.StringVar(&contextName, "context", "", "contexto Docker")
	flag.StringVar(&theme, "theme", "", "tema")
	flag.BoolVar(&readOnly, "read-only", false, "bloqueia mutações")
	flag.BoolVar(&noMouse, "no-mouse", false, "desabilita mouse")
	flag.BoolVar(&debug, "debug", false, "diagnóstico detalhado")
	flag.BoolVar(&showVersion, "version", false, "exibe versão")
	flag.Parse()
	if showVersion {
		fmt.Println("docktop", version)
		return
	}
	cfg, err := config.Load(path)
	if err != nil {
		fatal(err)
	}
	if contextName != "" {
		cfg.DefaultContext = contextName
	}
	if theme != "" {
		cfg.Theme = theme
	}
	if readOnly {
		cfg.ReadOnly = true
	}
	if noMouse {
		cfg.MouseEnabled = false
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	m, closeFn, err := app.New(ctx, cfg, version, debug)
	if err != nil {
		fatal(err)
	}
	defer closeFn()
	opts := []tea.ProgramOption{tea.WithAltScreen(), tea.WithContext(ctx)}
	if cfg.MouseEnabled {
		opts = append(opts, tea.WithMouseCellMotion())
	}
	_, err = tea.NewProgram(m, opts...).Run()
	if err != nil && !errors.Is(err, context.Canceled) {
		fatal(err)
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "docktop:", err); os.Exit(1) }
