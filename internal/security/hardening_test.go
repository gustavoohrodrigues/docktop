package security

import "testing"

func TestGenerateHardeningPlanSelectsOnlyRequestedControls(t *testing.T) {
	plan := GenerateHardeningPlan(AuditInput{
		ContainerID: "id", ContainerName: "web", User: "root", PIDMode: "host",
	}, []HardeningControlID{ControlNoNewPrivileges, ControlPrivatePID})
	selected := SelectedControlIDs(plan)
	if len(selected) != 2 || selected[0] != ControlNoNewPrivileges || selected[1] != ControlPrivatePID {
		t.Fatalf("unexpected selection: %#v", selected)
	}
	if len(plan.Diff) != 2 {
		t.Fatalf("expected two diff entries, got %#v", plan.Diff)
	}
}

func TestGenerateHardeningPlanMarksCompatibilityRisks(t *testing.T) {
	plan := GenerateHardeningPlan(AuditInput{
		User: "root", Privileged: true, NetworkMode: "host",
		Mounts: []Mount{{Source: "/var/run/docker.sock", Destination: "/var/run/docker.sock"}},
	}, []HardeningControlID{ControlNonRootUser, ControlDisablePrivileged, ControlPrivateNetwork, ControlRemoveDockerSocket})
	if len(plan.Warnings) != 4 {
		t.Fatalf("expected four compatibility warnings, got %#v", plan.Warnings)
	}
}

func TestValidateControlSelection(t *testing.T) {
	if err := ValidateControlSelection(nil); err == nil {
		t.Fatal("empty selection should fail")
	}
	if err := ValidateControlSelection([]HardeningControlID{"unknown"}); err == nil {
		t.Fatal("unknown control should fail")
	}
	if err := ValidateControlSelection([]HardeningControlID{ControlPIDLimit}); err != nil {
		t.Fatalf("valid selection rejected: %v", err)
	}
}

func TestComposePlanIsMarkedManaged(t *testing.T) {
	plan := GenerateHardeningPlan(AuditInput{Labels: map[string]string{
		"com.docker.compose.project": "demo", "com.docker.compose.service": "web",
	}}, nil)
	if !plan.ComposeManaged {
		t.Fatal("Compose plan not marked")
	}
}

func TestPlanReportsAlreadyAppliedAndDoesNotSelectIt(t *testing.T) {
	plan := GenerateHardeningPlan(AuditInput{
		NoNewPrivileges: true, ReadonlyRootFS: true, User: "1000",
		CapDrop: []string{"ALL"}, PidsLimit: 100, MemoryLimit: 128 << 20,
		NanoCPUs: 500_000_000, MemorySwap: 256 << 20,
		Tmpfs:   map[string]string{"/tmp": "rw,nosuid,nodev,noexec,size=64m"},
		Ulimits: []Ulimit{{Name: "nofile", Soft: 1024, Hard: 2048}},
	}, []HardeningControlID{ControlNoNewPrivileges, ControlCPULimit, ControlTmpfsTmp})
	states := map[HardeningControlID]ControlState{}
	for _, control := range plan.Controls {
		states[control.ID] = control.State
	}
	for _, id := range []HardeningControlID{ControlNoNewPrivileges, ControlReadOnlyRootFS, ControlNonRootUser, ControlDropCapabilities, ControlCPULimit, ControlSwapLimit, ControlNoFileLimit, ControlTmpfsTmp} {
		if states[id] != ControlApplied {
			t.Errorf("%s should be applied, got %s", id, states[id])
		}
	}
	if len(SelectedControlIDs(plan)) != 0 || len(plan.Diff) != 0 {
		t.Fatalf("already-applied controls produced changes: selected=%v diff=%v", SelectedControlIDs(plan), plan.Diff)
	}
}

func TestPlanMarksPartialCapabilities(t *testing.T) {
	plan := GenerateHardeningPlan(AuditInput{CapDrop: []string{"ALL"}, CapAdd: []string{"NET_BIND_SERVICE"}}, nil)
	for _, control := range plan.Controls {
		if control.ID == ControlDropCapabilities && control.State != ControlPartial {
			t.Fatalf("expected partial capability state, got %s", control.State)
		}
	}
}

func TestExpandedControlsAreAvailable(t *testing.T) {
	plan := GenerateHardeningPlan(AuditInput{NetworkMode: "host", UserNSMode: "host"}, nil)
	available := map[HardeningControlID]bool{}
	for _, control := range plan.Controls {
		available[control.ID] = true
	}
	for _, id := range []HardeningControlID{
		ControlCPULimit, ControlSwapLimit, ControlNoFileLimit, ControlDefaultSeccomp,
		ControlAppArmor, ControlPrivateUserNS, ControlTmpfsTmp, ControlTmpfsRun,
		ControlReadOnlySensitive, ControlRemovePorts,
	} {
		if !available[id] {
			t.Errorf("expanded control %s missing", id)
		}
	}
}
