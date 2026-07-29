package security

import (
	"strings"
	"testing"
	"time"
)

func TestAuditDetectsCoreRisksAndRedactsSecrets(t *testing.T) {
	input := AuditInput{
		ContainerID: "abc", ContainerName: "web", ImageReference: "example/web:latest",
		User: "", Privileged: true, SecurityOptions: []string{"seccomp=unconfined"},
		CapAdd: []string{"SYS_ADMIN"}, PIDMode: "host", IPCMode: "host", NetworkMode: "host", UserNSMode: "host",
		Mounts: []Mount{
			{Type: "bind", Source: "/var/run/docker.sock", Destination: "/var/run/docker.sock", ReadWrite: true},
			{Type: "bind", Source: "/etc", Destination: "/host/etc", ReadWrite: true},
		},
		Devices: []string{"/dev/kvm"}, PublishedPorts: []string{"0.0.0.0:80->80/tcp"},
		Environment: []string{"APP_ENV=prod", "DB_PASSWORD=do-not-display"},
		Labels:      map[string]string{"com.docker.compose.project": "demo", "com.docker.compose.service": "web"},
	}
	report := AuditContainer(input, time.Unix(1, 0))
	for _, title := range []string{
		"Container runs as root", "Privileged mode enabled", "Docker socket mounted",
		"Host PID namespace shared", "Host IPC namespace shared", "Host network namespace shared",
		"Resource limits are not configured", "Possible secret in environment",
		"Mutable image reference", "Container is managed by Docker Compose",
	} {
		if !hasFinding(report, title) {
			t.Errorf("missing finding %q", title)
		}
	}
	for _, finding := range report.Findings {
		if strings.Contains(finding.CurrentValue, "do-not-display") {
			t.Fatal("secret value leaked in finding")
		}
	}
	if report.RuntimeScore.Value != 0 {
		t.Fatalf("expected clamped score 0, got %d", report.RuntimeScore.Value)
	}
	if !report.Compose {
		t.Fatal("Compose labels not detected")
	}
}

func TestHardenedInputAvoidsCoreFindings(t *testing.T) {
	input := AuditInput{
		User: "65532:65532", ImageReference: "example/web@sha256:0123",
		NoNewPrivileges: true, ReadonlyRootFS: true, CapDrop: []string{"ALL"},
		MemoryLimit: 128 << 20, PidsLimit: 100, HasHealthcheck: true,
		AppArmorProfile: "docker-default", UserNSMode: "private",
	}
	report := AuditContainer(input, time.Now())
	for _, title := range []string{
		"Container runs as root", "Privileged mode enabled", "Excessive Linux capabilities",
		"Root filesystem is writable", "Resource limits are not configured", "PID limit is not configured",
		"Mutable image reference", "Health check is not configured",
	} {
		if hasFinding(report, title) {
			t.Errorf("unexpected finding %q", title)
		}
	}
	if report.RuntimeScore.Value != 100 {
		t.Fatalf("expected score 100, got %d", report.RuntimeScore.Value)
	}
}

func TestRootUserDetection(t *testing.T) {
	for _, user := range []string{"", "root", "0", "0:1000"} {
		if !runsAsRoot(user) {
			t.Errorf("%q should be root", user)
		}
	}
	for _, user := range []string{"1000", "1000:1000", "app"} {
		if runsAsRoot(user) {
			t.Errorf("%q should not be root", user)
		}
	}
}

func TestCapabilityAnalysis(t *testing.T) {
	if got := capabilityState([]string{"CAP_SYS_ADMIN"}, nil); !got.risky || got.severity != SeverityHigh {
		t.Fatalf("dangerous capability not classified: %#v", got)
	}
	if got := capabilityState(nil, []string{"ALL"}); got.risky {
		t.Fatalf("drop ALL should not be risky: %#v", got)
	}
}

func TestSensitiveMountAndSocketDetection(t *testing.T) {
	socket := Mount{Type: "bind", Source: "/var/run/docker.sock", Destination: "/sock"}
	if !isDockerSocket(socket) || !isSensitiveHostMount(socket) {
		t.Fatal("Docker socket mount not detected")
	}
	if isSensitiveHostMount(Mount{Type: "volume", Source: "data", Destination: "/data"}) {
		t.Fatal("named volume incorrectly classified as sensitive host mount")
	}
}

func TestSeverityOrder(t *testing.T) {
	report := AuditContainer(AuditInput{Privileged: true, User: "1000", ImageReference: "x@sha256:y", CapDrop: []string{"ALL"}, ReadonlyRootFS: true, NoNewPrivileges: true, MemoryLimit: 1, PidsLimit: 1, HasHealthcheck: true, AppArmorProfile: "default"}, time.Now())
	if len(report.Findings) == 0 || report.Findings[0].Severity != SeverityCritical {
		t.Fatalf("findings are not severity ordered: %#v", report.Findings)
	}
}

func TestAuditIncludesDynamicHardeningControlState(t *testing.T) {
	report := AuditContainer(AuditInput{
		NoNewPrivileges: true, ReadonlyRootFS: true, User: "1000",
		CapDrop: []string{"ALL"}, PidsLimit: 100, MemoryLimit: 128 << 20,
	}, time.Now())
	states := map[HardeningControlID]ControlState{}
	for _, control := range report.Controls {
		states[control.ID] = control.State
	}
	for _, id := range []HardeningControlID{
		ControlNoNewPrivileges, ControlReadOnlyRootFS, ControlNonRootUser,
		ControlDropCapabilities, ControlPIDLimit, ControlMemoryLimit,
	} {
		if states[id] != ControlApplied {
			t.Errorf("audit should report %s as applied, got %s", id, states[id])
		}
	}
	if states[ControlCPULimit] != ControlNotApplied {
		t.Fatalf("audit should report missing CPU limit, got %s", states[ControlCPULimit])
	}
}

func hasFinding(report ContainerSecurityReport, title string) bool {
	for _, finding := range report.Findings {
		if finding.Title == title {
			return true
		}
	}
	return false
}
