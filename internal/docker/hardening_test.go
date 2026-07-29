package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/docktop/docktop/internal/security"
)

func TestApplySelectedControlsAppliesOnlySelection(t *testing.T) {
	cfg := container.Config{User: "root"}
	host := container.HostConfig{Privileged: true, ReadonlyRootfs: false}
	applySelectedControls(&cfg, &host, []security.HardeningControlID{
		security.ControlNoNewPrivileges, security.ControlPIDLimit,
	})
	if !hasSecurityOption(host.SecurityOpt, "no-new-privileges") {
		t.Fatal("no-new-privileges not applied")
	}
	if host.PidsLimit == nil || *host.PidsLimit != 512 {
		t.Fatalf("PID limit not applied: %#v", host.PidsLimit)
	}
	if !host.Privileged || host.ReadonlyRootfs || cfg.User != "root" {
		t.Fatalf("unselected controls changed config: cfg=%#v host=%#v", cfg, host)
	}
}

func TestApplySelectedControlsCanApplySingleControl(t *testing.T) {
	cfg := container.Config{}
	host := container.HostConfig{CapAdd: []string{"NET_ADMIN"}}
	applySelectedControls(&cfg, &host, []security.HardeningControlID{security.ControlDropCapabilities})
	if len(host.CapAdd) != 0 || len(host.CapDrop) != 1 || host.CapDrop[0] != "ALL" {
		t.Fatalf("capabilities not restricted: add=%v drop=%v", host.CapAdd, host.CapDrop)
	}
}

func TestRemoveDockerSocketPreservesOtherMounts(t *testing.T) {
	cfg := container.Config{}
	host := container.HostConfig{
		Binds: []string{"/var/run/docker.sock:/var/run/docker.sock", "/srv/data:/data"},
		Mounts: []mount.Mount{
			{Source: "/run/docker.sock", Target: "/socket"},
			{Source: "/srv/config", Target: "/config"},
		},
	}
	applySelectedControls(&cfg, &host, []security.HardeningControlID{security.ControlRemoveDockerSocket})
	if len(host.Binds) != 1 || host.Binds[0] != "/srv/data:/data" {
		t.Fatalf("bind filtering failed: %#v", host.Binds)
	}
	if len(host.Mounts) != 1 || host.Mounts[0].Target != "/config" {
		t.Fatalf("mount filtering failed: %#v", host.Mounts)
	}
}

func TestApplyExpandedControls(t *testing.T) {
	cfg := container.Config{}
	host := container.HostConfig{
		SecurityOpt:  []string{"seccomp=unconfined"},
		PortBindings: map[nat.Port][]nat.PortBinding{"80/tcp": {{HostPort: "8080"}}},
		Binds:        []string{"/etc/app:/config:rw", "/srv/data:/data:rw"},
	}
	applySelectedControls(&cfg, &host, []security.HardeningControlID{
		security.ControlCPULimit, security.ControlSwapLimit, security.ControlNoFileLimit,
		security.ControlDefaultSeccomp, security.ControlAppArmor, security.ControlTmpfsTmp,
		security.ControlReadOnlySensitive, security.ControlRemovePorts,
	})
	if host.NanoCPUs != 1_000_000_000 || host.Memory != 512<<20 || host.MemorySwap != 512<<20 {
		t.Fatalf("resource limits not applied: cpu=%d memory=%d swap=%d", host.NanoCPUs, host.Memory, host.MemorySwap)
	}
	if len(host.Ulimits) != 1 || host.Ulimits[0].Name != "nofile" {
		t.Fatalf("nofile limit missing: %#v", host.Ulimits)
	}
	if hasSecurityOption(host.SecurityOpt, "seccomp=unconfined") || !hasSecurityOption(host.SecurityOpt, "apparmor=docker-default") {
		t.Fatalf("security profiles incorrect: %#v", host.SecurityOpt)
	}
	if host.Tmpfs["/tmp"] == "" || len(host.PortBindings) != 0 {
		t.Fatalf("tmpfs/ports controls missing: tmpfs=%v ports=%v", host.Tmpfs, host.PortBindings)
	}
	if host.Binds[0] != "/etc/app:/config:ro" || host.Binds[1] != "/srv/data:/data:rw" {
		t.Fatalf("sensitive bind conversion incorrect: %#v", host.Binds)
	}
}
