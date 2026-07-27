package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docktop/docktop/internal/config"
	"github.com/docktop/docktop/internal/docker"
)

func TestMouseSelectsVisibleTab(t *testing.T) {
	m := &Model{w: 120, tab: 0}
	start := 120 - helpWidth(m.helpLabel()) - 1
	total := 0
	for i := 0; i < 9; i++ {
		total += len(m.tabName(i)) + 2
	}
	x := (start-total)/2 + 1
	model, _ := m.updateMouse(tea.MouseMsg{X: x, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if model.(*Model).tab != 0 {
		t.Fatalf("aba incorreta: %d", model.(*Model).tab)
	}
	secondX := x + len(m.tabName(0)) + 2
	model, _ = m.updateMouse(tea.MouseMsg{X: secondX, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if model.(*Model).tab != 1 {
		t.Fatalf("clique não selecionou Containers: %d", model.(*Model).tab)
	}
}

func TestMouseSelectsContainerRowAndWheel(t *testing.T) {
	m := &Model{w: 100, h: 30, tab: 1, snap: docker.Snapshot{Containers: []docker.Container{{Name: "one"}, {Name: "two"}, {Name: "three"}}}}
	model, _ := m.updateMouse(tea.MouseMsg{X: 5, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if model.(*Model).cursor != 1 {
		t.Fatalf("linha incorreta: %d", model.(*Model).cursor)
	}
	model, _ = m.updateMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	if model.(*Model).cursor != 0 {
		t.Fatalf("wheel não moveu seleção: %d", model.(*Model).cursor)
	}
}

func TestLongContainerListKeepsSelectionVisible(t *testing.T) {
	containers := make([]docker.Container, 30)
	for i := range containers {
		containers[i].Name = fmt.Sprintf("container-%02d", i)
	}
	m := &Model{
		w:    100,
		h:    20,
		tab:  1,
		snap: docker.Snapshot{Containers: containers},
	}

	model, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	got := model.(*Model)
	if got.cursor != 29 || got.listOffset == 0 {
		t.Fatalf("fim da lista não ficou visível: cursor=%d offset=%d", got.cursor, got.listOffset)
	}
	view := got.containers()
	if !strings.Contains(view, "container-29") || strings.Contains(view, "container-00") {
		t.Fatalf("viewport incorreta no fim da lista: %s", view)
	}
	if !strings.Contains(view, "30/30") {
		t.Fatalf("progresso da lista ausente: %s", view)
	}
}

func TestPageNavigationAndMouseUseListOffset(t *testing.T) {
	containers := make([]docker.Container, 30)
	for i := range containers {
		containers[i].Name = fmt.Sprintf("container-%02d", i)
	}
	m := &Model{
		w:          100,
		h:          20,
		tab:        1,
		cursor:     20,
		listOffset: 18,
		snap:       docker.Snapshot{Containers: containers},
	}

	model, _ := m.updateKeys(tea.KeyMsg{Type: tea.KeyPgDown})
	got := model.(*Model)
	if got.cursor != 28 || got.listOffset == 18 {
		t.Fatalf("PgDn não rolou a viewport: cursor=%d offset=%d", got.cursor, got.listOffset)
	}

	model, _ = got.updateMouse(tea.MouseMsg{X: 5, Y: 6, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	got = model.(*Model)
	if got.cursor != got.listOffset+1 {
		t.Fatalf("clique não considerou offset: cursor=%d offset=%d", got.cursor, got.listOffset)
	}
}

func TestUpdateRoutesKeyboardEvents(t *testing.T) {
	m := &Model{w: 100, h: 30}
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if model.(*Model).tab != 1 {
		t.Fatalf("Tab não foi encaminhado para updateKeys: %d", model.(*Model).tab)
	}
	model, _ = model.(*Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if model.(*Model).mode != "help" {
		t.Fatalf("atalho de ajuda não foi processado: %q", model.(*Model).mode)
	}
}

func TestAllMainScreensUseSelectedLanguage(t *testing.T) {
	m := &Model{cfg: config.Config{Language: "en-US"}}
	if got := m.containers(); !strings.Contains(got, "NAME") || !strings.Contains(got, "No containers") {
		t.Fatalf("Containers não traduzido: %s", got)
	}
	if got := m.services(); !strings.Contains(got, "REPLICAS") || !strings.Contains(got, "No services") {
		t.Fatalf("Services não traduzido: %s", got)
	}
	m.cfg.Language = "es"
	if got := m.nodes(); !strings.Contains(got, "ESTADO") || !strings.Contains(got, "Los nodos") {
		t.Fatalf("Nodes não traduzido: %s", got)
	}
	if !strings.Contains(helpManual("es"), "ATAJOS GLOBALES") || !strings.Contains(helpManual("en-US"), "GLOBAL SHORTCUTS") {
		t.Fatal("manuais não traduzidos")
	}
}
