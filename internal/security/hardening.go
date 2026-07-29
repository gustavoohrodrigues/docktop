package security

import (
	"errors"
	"fmt"
	"strings"
)

type HardeningControlID string

const (
	ControlNoNewPrivileges    HardeningControlID = "no-new-privileges"
	ControlDisablePrivileged  HardeningControlID = "disable-privileged"
	ControlDropCapabilities   HardeningControlID = "drop-all-capabilities"
	ControlReadOnlyRootFS     HardeningControlID = "read-only-root-filesystem"
	ControlNonRootUser        HardeningControlID = "non-root-user"
	ControlPIDLimit           HardeningControlID = "pid-limit"
	ControlMemoryLimit        HardeningControlID = "memory-limit"
	ControlPrivatePID         HardeningControlID = "private-pid-namespace"
	ControlPrivateIPC         HardeningControlID = "private-ipc-namespace"
	ControlPrivateNetwork     HardeningControlID = "private-network-namespace"
	ControlRemoveDockerSocket HardeningControlID = "remove-docker-socket"
	ControlRemoveDevices      HardeningControlID = "remove-device-mappings"
	ControlCPULimit           HardeningControlID = "cpu-limit"
	ControlSwapLimit          HardeningControlID = "swap-limit"
	ControlNoFileLimit        HardeningControlID = "nofile-limit"
	ControlDefaultSeccomp     HardeningControlID = "default-seccomp"
	ControlAppArmor           HardeningControlID = "apparmor-profile"
	ControlPrivateUserNS      HardeningControlID = "private-user-namespace"
	ControlTmpfsTmp           HardeningControlID = "tmpfs-tmp"
	ControlTmpfsRun           HardeningControlID = "tmpfs-run"
	ControlReadOnlySensitive  HardeningControlID = "read-only-sensitive-mounts"
	ControlRemovePorts        HardeningControlID = "remove-published-ports"
)

type ControlState string

const (
	ControlNotApplied ControlState = "not-applied"
	ControlApplied    ControlState = "applied"
	ControlPartial    ControlState = "partial"
)

type HardeningControl struct {
	ID                      HardeningControlID
	Title                   string
	CurrentValue            string
	ProposedValue           string
	Benefit                 string
	CompatibilityRisk       string
	OperatingSystemSupport  string
	RecreationRequired      bool
	ProbableIncompatibility bool
	Selected                bool
	State                   ControlState
}

type ConfigurationDiff struct {
	Property, Before, After string
}

type CompatibilityWarning struct {
	Control HardeningControlID
	Message string
}

type HardeningPlan struct {
	ContainerID, ContainerName, Profile string
	ComposeManaged                      bool
	Controls                            []HardeningControl
	Diff                                []ConfigurationDiff
	Warnings                            []CompatibilityWarning
}

type HardeningResult struct {
	ContainerID, ContainerName, BackupContainerName string
	AppliedControls                                 []HardeningControlID
	Running, Healthy                                bool
	RollbackApplied                                 bool
}

func GenerateHardeningPlan(in AuditInput, selected []HardeningControlID) HardeningPlan {
	chosen := make(map[HardeningControlID]bool, len(selected))
	for _, id := range selected {
		chosen[id] = true
	}
	plan := HardeningPlan{
		ContainerID: in.ContainerID, ContainerName: in.ContainerName,
		Profile: "Custom", ComposeManaged: isCompose(in.Labels),
	}
	add := func(control HardeningControl) {
		control.State = hardeningControlState(control.ID, in)
		control.Selected = chosen[control.ID] && control.State != ControlApplied
		plan.Controls = append(plan.Controls, control)
		if control.Selected {
			plan.Diff = append(plan.Diff, ConfigurationDiff{Property: propertyForControl(control.ID), Before: control.CurrentValue, After: control.ProposedValue})
			if control.ID == ControlSwapLimit && in.MemoryLimit == 0 {
				plan.Diff = append(plan.Diff, ConfigurationDiff{Property: "HostConfig.Memory", Before: "0", After: fmt.Sprint(int64(512 << 20))})
				plan.Warnings = append(plan.Warnings, CompatibilityWarning{Control: control.ID, Message: "Docker requires a memory limit before a swap limit; 512 MiB memory will also be applied."})
			}
			if control.ProbableIncompatibility {
				plan.Warnings = append(plan.Warnings, CompatibilityWarning{Control: control.ID, Message: control.CompatibilityRisk})
			}
		}
	}
	control := func(id HardeningControlID, title, current, proposed, benefit, risk string, incompatible bool) HardeningControl {
		return HardeningControl{ID: id, Title: title, CurrentValue: current, ProposedValue: proposed, Benefit: benefit,
			CompatibilityRisk: risk, OperatingSystemSupport: "Linux Docker Engine", RecreationRequired: true, ProbableIncompatibility: incompatible}
	}
	add(control(ControlNoNewPrivileges, "Enable no-new-privileges", boolText(in.NoNewPrivileges), "true",
		"Prevents setuid, setgid, and file capabilities from granting new privileges.",
		"Software intentionally elevating privileges during startup may fail.", false))
	add(control(ControlDisablePrivileged, "Disable privileged mode", boolText(in.Privileged), "false",
		"Removes broad host and device access.",
		"Privileged, nested-container, or hardware-management workloads may fail.", in.Privileged))
	add(control(ControlDropCapabilities, "Drop all Linux capabilities", capabilityState(in.CapAdd, in.CapDrop).value, "drop ALL",
		"Minimizes kernel privileges available to processes.",
		"Any operation requiring a Linux capability will fail until that capability is added explicitly.", len(in.CapAdd) > 0 || !contains(upper(in.CapDrop), "ALL")))
	add(control(ControlReadOnlyRootFS, "Use a read-only root filesystem", boolText(in.ReadonlyRootFS), "true",
		"Prevents modification of the container root filesystem.",
		"Applications writing outside declared volumes or tmpfs paths will fail.", !in.ReadonlyRootFS))
	add(control(ControlNonRootUser, "Run as a non-root user", displayUser(in.User), "65532:65532",
		"Reduces the privileges of a compromised process.",
		"File ownership, low ports, startup scripts, and package operations may be incompatible.", runsAsRoot(in.User)))
	add(control(ControlPIDLimit, "Set PID limit", fmt.Sprint(in.PidsLimit), "512",
		"Limits process-fork exhaustion.",
		"Highly concurrent workloads may exceed the limit.", false))
	add(control(ControlMemoryLimit, "Set memory limit", fmt.Sprint(in.MemoryLimit), fmt.Sprint(int64(512<<20)),
		"Limits host memory exhaustion.",
		"Workloads needing more than 512 MiB may be terminated by the kernel.", in.MemoryLimit == 0 || in.MemoryLimit > 512<<20))
	add(control(ControlPrivatePID, "Disable host PID namespace", emptyDefault(in.PIDMode, "private"), "private",
		"Restores process namespace isolation.",
		"Host monitoring and debugging workloads may fail.", strings.EqualFold(in.PIDMode, "host")))
	add(control(ControlPrivateIPC, "Disable host IPC namespace", emptyDefault(in.IPCMode, "private"), "private",
		"Restores IPC namespace isolation.",
		"Applications sharing host IPC resources may fail.", strings.EqualFold(in.IPCMode, "host")))
	add(control(ControlPrivateNetwork, "Disable host network namespace", emptyDefault(in.NetworkMode, "default"), "bridge/default",
		"Restores network namespace isolation.",
		"Host-network services and their port assumptions may fail.", strings.EqualFold(in.NetworkMode, "host")))
	socket := false
	for _, mount := range in.Mounts {
		socket = socket || isDockerSocket(mount)
	}
	add(control(ControlRemoveDockerSocket, "Remove Docker socket access", boolText(socket), "false",
		"Removes access commonly equivalent to host root.",
		"Docker-aware applications and nested-container workflows will fail.", socket))
	add(control(ControlRemoveDevices, "Remove host device mappings", fmt.Sprintf("%d", len(in.Devices)), "none",
		"Removes direct hardware and kernel attack surface.",
		"Hardware-dependent workloads will fail.", len(in.Devices) > 0))
	add(control(ControlCPULimit, "Set CPU limit", cpuCurrent(in), "1 CPU",
		"Limits sustained host CPU consumption.", "CPU-intensive workloads may be throttled.", false))
	add(control(ControlSwapLimit, "Set swap limit", fmt.Sprint(in.MemorySwap), fmt.Sprint(int64(512<<20)),
		"Bounds combined memory and swap consumption.",
		"Memory spikes may cause earlier out-of-memory termination; daemon swap support is required.", in.MemorySwap <= 0))
	add(control(ControlNoFileLimit, "Set file-descriptor limit", ulimitCurrent(in.Ulimits, "nofile"), "1024:4096",
		"Limits exhaustion of host file descriptors.",
		"High-connection or file-intensive workloads may exceed the limit.", false))
	add(control(ControlDefaultSeccomp, "Use default seccomp protection", seccompState(in.SecurityOptions), "default/daemon policy",
		"Filters dangerous system calls using Docker's default policy.",
		"Applications requiring filtered system calls may fail.", seccompState(in.SecurityOptions) == "unconfined"))
	add(control(ControlAppArmor, "Use docker-default AppArmor profile", emptyDefault(in.AppArmorProfile, "none"), "docker-default",
		"Adds mandatory access-control confinement on AppArmor-enabled Linux hosts.",
		"Unsupported hosts or applications needing denied operations may fail.", in.AppArmorProfile == ""))
	add(control(ControlPrivateUserNS, "Disable host user namespace", emptyDefault(in.UserNSMode, "daemon default"), "daemon default/private",
		"Allows daemon user-namespace isolation instead of forcing the host namespace.",
		"Bind-mount ownership may require migration.", strings.EqualFold(in.UserNSMode, "host")))
	add(control(ControlTmpfsTmp, "Add restricted tmpfs for /tmp", tmpfsCurrent(in.Tmpfs, "/tmp"), "rw,nosuid,nodev,noexec,size=64m",
		"Provides bounded temporary writable storage for a read-only root filesystem.",
		"Applications executing files from /tmp or needing more space may fail.", false))
	add(control(ControlTmpfsRun, "Add restricted tmpfs for /run", tmpfsCurrent(in.Tmpfs, "/run"), "rw,nosuid,nodev,noexec,size=16m",
		"Provides bounded runtime storage without making the root filesystem writable.",
		"Init systems or applications requiring executable or larger /run content may fail.", false))
	add(control(ControlReadOnlySensitive, "Make sensitive bind mounts read-only", sensitiveMountCurrent(in.Mounts), "all sensitive binds read-only",
		"Reduces modification of sensitive host data.",
		"Applications that write to those host paths will fail.", hasWritableSensitiveMount(in.Mounts)))
	add(control(ControlRemovePorts, "Remove all published ports", fmt.Sprintf("%d", len(in.PublishedPorts)), "0",
		"Removes externally published network attack surface.",
		"Services will no longer be reachable through current host port mappings.", len(in.PublishedPorts) > 0))
	return plan
}

func SelectedControlIDs(plan HardeningPlan) []HardeningControlID {
	var out []HardeningControlID
	for _, control := range plan.Controls {
		if control.Selected {
			out = append(out, control.ID)
		}
	}
	return out
}

func ValidateControlSelection(selected []HardeningControlID) error {
	if len(selected) == 0 {
		return errors.New("selecione ao menos um controle de hardening")
	}
	valid := map[HardeningControlID]bool{
		ControlNoNewPrivileges: true, ControlDisablePrivileged: true, ControlDropCapabilities: true,
		ControlReadOnlyRootFS: true, ControlNonRootUser: true, ControlPIDLimit: true,
		ControlMemoryLimit: true, ControlPrivatePID: true, ControlPrivateIPC: true,
		ControlPrivateNetwork: true, ControlRemoveDockerSocket: true, ControlRemoveDevices: true,
		ControlCPULimit: true, ControlSwapLimit: true, ControlNoFileLimit: true,
		ControlDefaultSeccomp: true, ControlAppArmor: true, ControlPrivateUserNS: true,
		ControlTmpfsTmp: true, ControlTmpfsRun: true, ControlReadOnlySensitive: true, ControlRemovePorts: true,
	}
	for _, id := range selected {
		if !valid[id] {
			return fmt.Errorf("controle de hardening desconhecido: %s", id)
		}
	}
	return nil
}

func propertyForControl(id HardeningControlID) string {
	return map[HardeningControlID]string{
		ControlNoNewPrivileges: "HostConfig.SecurityOpt", ControlDisablePrivileged: "HostConfig.Privileged",
		ControlDropCapabilities: "HostConfig.CapDrop/CapAdd", ControlReadOnlyRootFS: "HostConfig.ReadonlyRootfs",
		ControlNonRootUser: "Config.User", ControlPIDLimit: "HostConfig.PidsLimit",
		ControlMemoryLimit: "HostConfig.Memory", ControlPrivatePID: "HostConfig.PidMode",
		ControlPrivateIPC: "HostConfig.IpcMode", ControlPrivateNetwork: "HostConfig.NetworkMode",
		ControlRemoveDockerSocket: "HostConfig.Binds/Mounts", ControlRemoveDevices: "HostConfig.Devices",
		ControlCPULimit: "HostConfig.NanoCPUs/CPUQuota", ControlSwapLimit: "HostConfig.MemorySwap",
		ControlNoFileLimit: "HostConfig.Ulimits[nofile]", ControlDefaultSeccomp: "HostConfig.SecurityOpt[seccomp]",
		ControlAppArmor: "AppArmorProfile/HostConfig.SecurityOpt", ControlPrivateUserNS: "HostConfig.UsernsMode",
		ControlTmpfsTmp: "HostConfig.Tmpfs[/tmp]", ControlTmpfsRun: "HostConfig.Tmpfs[/run]",
		ControlReadOnlySensitive: "HostConfig.Binds/Mounts", ControlRemovePorts: "HostConfig.PortBindings",
	}[id]
}

func hardeningControlState(id HardeningControlID, in AuditInput) ControlState {
	switch id {
	case ControlNoNewPrivileges:
		return appliedState(in.NoNewPrivileges)
	case ControlDisablePrivileged:
		return appliedState(!in.Privileged)
	case ControlDropCapabilities:
		if contains(upper(in.CapDrop), "ALL") && len(in.CapAdd) == 0 {
			return ControlApplied
		}
		if contains(upper(in.CapDrop), "ALL") {
			return ControlPartial
		}
	case ControlReadOnlyRootFS:
		return appliedState(in.ReadonlyRootFS)
	case ControlNonRootUser:
		return appliedState(!runsAsRoot(in.User))
	case ControlPIDLimit:
		return appliedState(in.PidsLimit > 0)
	case ControlMemoryLimit:
		return appliedState(in.MemoryLimit > 0)
	case ControlPrivatePID:
		return appliedState(!strings.EqualFold(in.PIDMode, "host"))
	case ControlPrivateIPC:
		return appliedState(!strings.EqualFold(in.IPCMode, "host"))
	case ControlPrivateNetwork:
		return appliedState(!strings.EqualFold(in.NetworkMode, "host"))
	case ControlRemoveDockerSocket:
		for _, item := range in.Mounts {
			if isDockerSocket(item) {
				return ControlNotApplied
			}
		}
		return ControlApplied
	case ControlRemoveDevices:
		return appliedState(len(in.Devices) == 0)
	case ControlCPULimit:
		return appliedState(in.NanoCPUs > 0 || in.CPUQuota > 0)
	case ControlSwapLimit:
		return appliedState(in.MemorySwap > 0)
	case ControlNoFileLimit:
		for _, item := range in.Ulimits {
			if item.Name == "nofile" {
				if item.Soft <= 1024 && item.Hard <= 4096 {
					return ControlApplied
				}
				return ControlPartial
			}
		}
	case ControlDefaultSeccomp:
		state := seccompState(in.SecurityOptions)
		if state == "default/daemon policy" {
			return ControlApplied
		}
		if state != "unconfined" {
			return ControlPartial
		}
	case ControlAppArmor:
		if in.AppArmorProfile == "docker-default" {
			return ControlApplied
		}
		if strings.TrimSpace(in.AppArmorProfile) != "" {
			return ControlPartial
		}
	case ControlPrivateUserNS:
		return appliedState(!strings.EqualFold(in.UserNSMode, "host"))
	case ControlTmpfsTmp:
		return tmpfsState(in.Tmpfs["/tmp"], "64m")
	case ControlTmpfsRun:
		return tmpfsState(in.Tmpfs["/run"], "16m")
	case ControlReadOnlySensitive:
		return appliedState(!hasWritableSensitiveMount(in.Mounts))
	case ControlRemovePorts:
		return appliedState(len(in.PublishedPorts) == 0)
	}
	return ControlNotApplied
}

func tmpfsState(value, size string) ControlState {
	if value == "" {
		return ControlNotApplied
	}
	for _, required := range []string{"nosuid", "nodev", "noexec", "size=" + size} {
		if !strings.Contains(value, required) {
			return ControlPartial
		}
	}
	return ControlApplied
}

func appliedState(applied bool) ControlState {
	if applied {
		return ControlApplied
	}
	return ControlNotApplied
}

func cpuCurrent(in AuditInput) string {
	if in.NanoCPUs > 0 {
		return fmt.Sprintf("%.2f CPU", float64(in.NanoCPUs)/1e9)
	}
	if in.CPUQuota > 0 {
		return fmt.Sprintf("quota=%d", in.CPUQuota)
	}
	return "unlimited"
}

func hasUlimit(values []Ulimit, name string) bool {
	for _, item := range values {
		if item.Name == name {
			return true
		}
	}
	return false
}

func ulimitCurrent(values []Ulimit, name string) string {
	for _, item := range values {
		if item.Name == name {
			return fmt.Sprintf("%d:%d", item.Soft, item.Hard)
		}
	}
	return "unlimited"
}

func tmpfsCurrent(values map[string]string, path string) string {
	if values[path] == "" {
		return "none"
	}
	return values[path]
}

func hasWritableSensitiveMount(mounts []Mount) bool {
	for _, item := range mounts {
		if item.ReadWrite && isSensitiveHostMount(item) {
			return true
		}
	}
	return false
}

func sensitiveMountCurrent(mounts []Mount) string {
	count := 0
	for _, item := range mounts {
		if item.ReadWrite && isSensitiveHostMount(item) {
			count++
		}
	}
	return fmt.Sprintf("%d writable", count)
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func emptyDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
