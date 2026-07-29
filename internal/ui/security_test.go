package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docktop/docktop/internal/config"
	"github.com/docktop/docktop/internal/docker"
	"github.com/docktop/docktop/internal/security"
)

type auditOnlyEngine struct {
	docker.Engine
	auditCalls               int
	lastID                   string
	prepareCalls, applyCalls int
}

func (e *auditOnlyEngine) SecurityAudit(_ context.Context, id string) (security.ContainerSecurityReport, error) {
	e.auditCalls++
	e.lastID = id
	return security.ContainerSecurityReport{
		ContainerID: id, ContainerName: "web", Image: "example/web:1",
		GeneratedAt: time.Unix(1, 0), RuntimeScore: security.SecurityScore{Value: 88},
	}, nil
}

func (e *auditOnlyEngine) PrepareHardening(_ context.Context, id string, selected []security.HardeningControlID) (security.HardeningPlan, error) {
	e.prepareCalls++
	e.lastID = id
	return security.GenerateHardeningPlan(security.AuditInput{ContainerID: id, ContainerName: "web"}, selected), nil
}

func (e *auditOnlyEngine) ApplyHardening(_ context.Context, id string, selected []security.HardeningControlID) (security.HardeningResult, error) {
	e.applyCalls++
	e.lastID = id
	return security.HardeningResult{ContainerID: "new-id", ContainerName: "web", AppliedControls: selected}, nil
}

func TestSecurityAuditActionUsesReadOnlyEngineMethod(t *testing.T) {
	engine := &auditOnlyEngine{}
	m := &Model{
		ctx: context.Background(), cfg: config.Config{Language: "en-US"}, engine: engine, tab: 1,
		snap: docker.Snapshot{Containers: []docker.Container{{ID: "container-id", Name: "web"}}},
	}
	cmd := m.containerSecurityAudit()
	if cmd == nil {
		t.Fatal("audit command was not created")
	}
	msg, ok := cmd().(securityAuditMsg)
	if !ok {
		t.Fatalf("unexpected message type %T", msg)
	}
	if msg.e != nil || engine.auditCalls != 1 || engine.lastID != "container-id" {
		t.Fatalf("unexpected audit invocation: calls=%d id=%q err=%v", engine.auditCalls, engine.lastID, msg.e)
	}
}

func TestSecurityReportRendersRequiredMetadataWithoutSecretValue(t *testing.T) {
	m := &Model{
		cfg: config.Config{Language: "en-US"},
		securityReport: security.ContainerSecurityReport{
			ContainerID: "abcdef1234567890", ContainerName: "web", Image: "example/web:latest",
			GeneratedAt: time.Unix(1, 0), RuntimeScore: security.SecurityScore{Value: 72},
			Findings: []security.SecurityFinding{{
				Title: "Possible secret in environment", Severity: security.SeverityHigh,
				CurrentValue: "DB_PASSWORD=[REDACTED]", Risk: "risk", Remediation: "use secrets",
				RecreationRequired: true, CompatibilityImpact: "validate", Property: "Config.Env[DB_PASSWORD]",
			}},
		},
	}
	got := m.renderSecurityReport()
	for _, wanted := range []string{"CONTAINER SECURITY AUDIT", "72/100", "DB_PASSWORD=[REDACTED]", "recreation required", "Config.Env[DB_PASSWORD]"} {
		if !strings.Contains(got, wanted) {
			t.Errorf("report missing %q:\n%s", wanted, got)
		}
	}
	if strings.Contains(got, "hunter2") {
		t.Fatal("secret value leaked")
	}
}

func TestSecurityReportStripsTerminalControlCharacters(t *testing.T) {
	m := &Model{
		cfg: config.Config{Language: "en-US"},
		securityReport: security.ContainerSecurityReport{
			ContainerID: "id", ContainerName: "web\x1b[31m", RuntimeScore: security.SecurityScore{Value: 100},
		},
	}
	if got := m.renderSecurityReport(); strings.ContainsRune(got, '\x1b') {
		t.Fatalf("terminal escape was not stripped: %q", got)
	}
}

func TestHardeningSelectionCanApplyOnlyOneControl(t *testing.T) {
	engine := &auditOnlyEngine{}
	m := &Model{
		ctx: context.Background(), cfg: config.Config{Language: "en-US"}, engine: engine, tab: 1,
		snap: docker.Snapshot{Containers: []docker.Container{{ID: "container-id", Name: "web"}}},
	}
	msg := m.beginHardening()().(hardeningPlanMsg)
	if msg.e != nil {
		t.Fatal(msg.e)
	}
	m.hardeningPlan, m.mode = msg.plan, "hardening-select"
	_, _ = m.updateOverlay(key(" "))
	if got := security.SelectedControlIDs(m.hardeningPlan); len(got) != 1 || got[0] != security.ControlNoNewPrivileges {
		t.Fatalf("unexpected selected controls: %#v", got)
	}
	_, reviewCmd := m.updateOverlay(key("enter"))
	review := reviewCmd().(hardeningPlanMsg)
	if len(review.plan.Diff) != 1 || engine.prepareCalls != 2 {
		t.Fatalf("single-control plan was not prepared: %#v", review.plan.Diff)
	}
	if engine.applyCalls != 0 {
		t.Fatal("review must not apply hardening")
	}
}

func TestHardeningSelectionCancelMakesNoApplyCall(t *testing.T) {
	engine := &auditOnlyEngine{}
	m := &Model{
		ctx: context.Background(), cfg: config.Config{Language: "en-US"}, engine: engine,
		mode: "hardening-select", hardeningPlan: security.GenerateHardeningPlan(security.AuditInput{}, nil),
	}
	_, cmd := m.updateOverlay(key("esc"))
	if cmd != nil || m.mode != "" || engine.applyCalls != 0 {
		t.Fatalf("cancel had side effects: mode=%q calls=%d", m.mode, engine.applyCalls)
	}
}

func TestSecurityAndHardeningContentFollowSelectedLanguage(t *testing.T) {
	m := &Model{
		cfg: config.Config{Language: "pt-BR"},
		securityReport: security.ContainerSecurityReport{
			RuntimeScore: security.SecurityScore{Value: 80},
			Findings: []security.SecurityFinding{{
				Title: "Container runs as root", Severity: security.SeverityHigh,
				CurrentValue:        "root (image default/empty Config.User)",
				Risk:                "A compromise may provide broad control inside the container and increase host impact.",
				Remediation:         "Configure and validate a non-root UID/GID in the image or container.",
				CompatibilityImpact: "May break file access, low-port binding, package installation, or startup scripts.",
			}},
		},
	}
	report := m.renderSecurityReport()
	for _, wanted := range []string{"Container executa como root", "Um comprometimento pode", "Configure e valide", "Pode quebrar acesso"} {
		if !strings.Contains(report, wanted) {
			t.Errorf("relatório pt-BR não contém %q:\n%s", wanted, report)
		}
	}
	if strings.Contains(report, "Container runs as root") {
		t.Fatalf("relatório pt-BR manteve título em inglês:\n%s", report)
	}

	m.cfg.Language = "es"
	m.hardeningPlan = security.GenerateHardeningPlan(security.AuditInput{}, nil)
	selector := m.renderHardeningSelector()
	for _, wanted := range []string{"CONTROLES INDIVIDUALES", "Deshabilitar modo privilegiado", "beneficio"} {
		if !strings.Contains(selector, wanted) {
			t.Errorf("selector es no contiene %q:\n%s", wanted, selector)
		}
	}
}

func TestHardeningSelectorShowsAndProtectsAppliedState(t *testing.T) {
	m := &Model{
		cfg: config.Config{Language: "en-US"}, mode: "hardening-select",
		hardeningPlan: security.GenerateHardeningPlan(security.AuditInput{NoNewPrivileges: true}, nil),
	}
	if got := m.renderHardeningSelector(); !strings.Contains(got, "already applied") || !strings.Contains(got, "[✓]") {
		t.Fatalf("applied state not visible:\n%s", got)
	}
	_, _ = m.updateOverlay(key(" "))
	if m.hardeningPlan.Controls[0].Selected {
		t.Fatal("already-applied control must not become selected")
	}
}

func TestAuditReportExplainsDynamicHardeningStates(t *testing.T) {
	m := &Model{
		cfg: config.Config{Language: "en-US"},
		securityReport: security.AuditContainer(security.AuditInput{
			ContainerName: "web", NoNewPrivileges: true,
		}, time.Now()),
	}
	got := m.renderSecurityReport()
	for _, wanted := range []string{"HARDENING CONTROL STATUS", "[✓]", "already applied", "Enable no-new-privileges", "benefit"} {
		if !strings.Contains(got, wanted) {
			t.Errorf("dynamic audit report missing %q:\n%s", wanted, got)
		}
	}
}

func TestContainerHintPlacesSecurityActionsLast(t *testing.T) {
	m := &Model{cfg: config.Config{Language: "en-US"}}
	got := m.containers()
	remove := strings.LastIndex(got, "d remove")
	audit := strings.LastIndex(got, "a audit")
	hardening := strings.LastIndex(got, "H apply hardening")
	if remove < 0 || audit <= remove || hardening <= audit {
		t.Fatalf("security actions are not last: %q", got)
	}
}

func key(value string) tea.KeyMsg {
	if value == "enter" {
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	if value == "esc" {
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}
