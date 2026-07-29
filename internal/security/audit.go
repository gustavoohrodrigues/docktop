package security

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Severity string

const (
	SeverityCritical      Severity = "Critical"
	SeverityHigh          Severity = "High"
	SeverityMedium        Severity = "Medium"
	SeverityLow           Severity = "Low"
	SeverityInformational Severity = "Informational"
)

type SecurityFinding struct {
	Title               string
	Severity            Severity
	CurrentValue        string
	Risk                string
	Remediation         string
	Automatic           bool
	RecreationRequired  bool
	CompatibilityImpact string
	Property            string
}

type ScoreDeduction struct {
	Condition         string
	Points            int
	Reason            string
	Evidence          string
	Remediation       string
	Automatic         bool
	CompatibilityRisk string
}

type SecurityScore struct {
	Value      int
	Deductions []ScoreDeduction
}

type ContainerSecurityReport struct {
	ContainerID   string
	ContainerName string
	Image         string
	GeneratedAt   time.Time
	Compose       bool
	Findings      []SecurityFinding
	Controls      []HardeningControl
	RuntimeScore  SecurityScore
}

type Mount struct {
	Type, Source, Destination string
	ReadWrite                 bool
}

type AuditInput struct {
	ContainerID, ContainerName, ImageReference, User       string
	Privileged, ReadonlyRootFS                             bool
	NoNewPrivileges                                        bool
	CapAdd, CapDrop, SecurityOptions                       []string
	Mounts                                                 []Mount
	Devices                                                []string
	PIDMode, IPCMode, NetworkMode, UserNSMode              string
	MemoryLimit, MemorySwap, NanoCPUs, CPUQuota, CPUShares int64
	PidsLimit                                              int64
	PublishedPorts                                         []string
	Tmpfs                                                  map[string]string
	Ulimits                                                []Ulimit
	HasHealthcheck                                         bool
	AppArmorProfile                                        string
	Environment                                            []string
	Labels                                                 map[string]string
}

type Ulimit struct {
	Name       string
	Soft, Hard int64
}

func AuditContainer(in AuditInput, now time.Time) ContainerSecurityReport {
	report := ContainerSecurityReport{
		ContainerID:   in.ContainerID,
		ContainerName: in.ContainerName,
		Image:         in.ImageReference,
		GeneratedAt:   now.UTC(),
		Compose:       isCompose(in.Labels),
		RuntimeScore:  SecurityScore{Value: 100},
	}
	add := func(f SecurityFinding, points int) {
		report.Findings = append(report.Findings, f)
		if points > 0 {
			report.RuntimeScore.Deductions = append(report.RuntimeScore.Deductions, ScoreDeduction{
				Condition: f.Title, Points: points, Reason: f.Risk, Evidence: f.CurrentValue,
				Remediation: f.Remediation, Automatic: f.Automatic, CompatibilityRisk: f.CompatibilityImpact,
			})
			report.RuntimeScore.Value -= points
		}
	}

	if runsAsRoot(in.User) {
		add(finding("Container runs as root", SeverityHigh, displayUser(in.User),
			"A compromise may provide broad control inside the container and increase host impact.",
			"Configure and validate a non-root UID/GID in the image or container.", false, true,
			"May break file access, low-port binding, package installation, or startup scripts.", "Config.User"), 15)
	}
	if in.Privileged {
		add(finding("Privileged mode enabled", SeverityCritical, "true",
			"Privileged containers receive nearly unrestricted host access.",
			"Recreate the container without privileged mode and grant only required capabilities/devices.", true, true,
			"High: workloads using nested containers, devices, or kernel features may stop working.", "HostConfig.Privileged"), 25)
	}
	if !in.NoNewPrivileges {
		add(finding("No-new-privileges is not enabled", SeverityMedium, "false",
			"Processes may gain additional privileges through setuid, setgid, or file capabilities.",
			"Add the security option no-new-privileges=true.", true, true,
			"May affect software that intentionally elevates privileges during startup.", "HostConfig.SecurityOpt"), 8)
	}

	caps := capabilityState(in.CapAdd, in.CapDrop)
	if caps.risky {
		add(finding("Excessive Linux capabilities", caps.severity, caps.value,
			"Powerful or broadly retained capabilities can expand the impact of a compromise.",
			"Drop all capabilities and add back only capabilities proven necessary.", true, true,
			"Removing a required capability can prevent startup or runtime operations.", "HostConfig.CapAdd/CapDrop"), caps.points)
	}
	if !in.ReadonlyRootFS {
		add(finding("Root filesystem is writable", SeverityMedium, "writable",
			"An attacker or faulty process can modify files in the container filesystem.",
			"Use a read-only root filesystem with explicit volumes or tmpfs for required writes.", true, true,
			"Likely to break applications that write outside declared writable paths.", "HostConfig.ReadonlyRootfs"), 7)
	}

	for _, mount := range in.Mounts {
		if isDockerSocket(mount) {
			mode := "read-only"
			if mount.ReadWrite {
				mode = "read-write"
			}
			add(finding("Docker socket mounted", SeverityCritical, mount.Destination+" ("+mode+")",
				"Docker daemon access is commonly equivalent to root access on the host, even through a read-only socket mount.",
				"Remove Docker socket access or replace it with a narrowly scoped proxy.", false, true,
				"Docker-aware workloads and Docker-in-Docker workflows may stop working.", "Mounts["+mount.Destination+"]"), 25)
			continue
		}
		if isSensitiveHostMount(mount) {
			severity := SeverityHigh
			points := 12
			if !mount.ReadWrite {
				severity, points = SeverityMedium, 6
			}
			add(finding("Sensitive host path mounted", severity, mount.Source+" → "+mount.Destination+" ("+mountMode(mount)+")",
				"Host system or identity data is exposed to the container.",
				"Remove the bind mount, narrow its source, or make it read-only where compatible.", false, true,
				"The application may depend on the mounted host data.", "Mounts["+mount.Destination+"]"), points)
		}
		if mount.ReadWrite && isBroadWritableMount(mount) {
			add(finding("Broad writable mount", SeverityMedium, mount.Source+" → "+mount.Destination,
				"A broad writable mount increases the host or persistent-data impact of a compromise.",
				"Narrow the mounted path or change it to read-only.", false, true,
				"Writes outside the narrowed path will fail.", "Mounts["+mount.Destination+"]"), 5)
		}
	}

	if len(in.Devices) > 0 {
		add(finding("Host devices mapped", SeverityHigh, strings.Join(in.Devices, ", "),
			"Direct device access can expose host data or kernel attack surface.",
			"Remove unnecessary mappings and restrict permissions for required devices.", false, true,
			"Hardware-dependent workloads may fail.", "HostConfig.Devices"), 12)
	}
	namespaceFinding := func(title, mode, property string) {
		add(finding(title, SeverityHigh, mode, "Sharing a host namespace weakens container isolation.",
			"Recreate the container with a private namespace.", true, true,
			"Monitoring, debugging, networking, or tightly coupled workloads may fail.", property), 10)
	}
	if strings.EqualFold(in.PIDMode, "host") {
		namespaceFinding("Host PID namespace shared", in.PIDMode, "HostConfig.PidMode")
	}
	if strings.EqualFold(in.IPCMode, "host") {
		namespaceFinding("Host IPC namespace shared", in.IPCMode, "HostConfig.IpcMode")
	}
	if strings.EqualFold(in.NetworkMode, "host") {
		namespaceFinding("Host network namespace shared", in.NetworkMode, "HostConfig.NetworkMode")
	}
	if strings.EqualFold(in.UserNSMode, "host") {
		add(finding("User namespace isolation disabled", SeverityMedium, in.UserNSMode,
			"Host user namespace sharing removes UID remapping isolation for this container.",
			"Use daemon user-namespace remapping where supported and compatible.", false, true,
			"Ownership of bind mounts and files may need migration.", "HostConfig.UsernsMode"), 7)
	} else if strings.TrimSpace(in.UserNSMode) == "" {
		add(finding("User namespace mode requires verification", SeverityInformational, "daemon default",
			"Container inspection does not prove whether daemon-level user namespace remapping is enabled.",
			"Verify the Docker daemon userns-remap configuration.", false, false,
			"Enabling daemon-level remapping can affect bind-mount ownership.", "HostConfig.UsernsMode"), 0)
	}

	if !hasResourceLimit(in) {
		add(finding("Resource limits are not configured", SeverityMedium, "memory/cpu limits absent",
			"An uncontrolled workload can exhaust host CPU or memory.",
			"Set workload-appropriate memory and CPU limits after measuring normal usage.", true, true,
			"Limits that are too low can cause throttling or out-of-memory termination.", "HostConfig.Resources"), 7)
	}
	if in.PidsLimit <= 0 {
		add(finding("PID limit is not configured", SeverityMedium, fmt.Sprint(in.PidsLimit),
			"A process storm can exhaust host PID resources.",
			"Set a workload-appropriate positive PID limit.", true, true,
			"A limit that is too low can prevent worker creation or startup.", "HostConfig.PidsLimit"), 5)
	}
	if len(in.PublishedPorts) > 0 {
		add(finding("Published ports require review", SeverityLow, strings.Join(in.PublishedPorts, ", "),
			"Published services increase externally reachable attack surface; necessity cannot be inferred automatically.",
			"Remove ports that are not required and bind internal-only services to a restricted host address.", false, true,
			"Removing a required mapping makes the service unreachable through that port.", "NetworkSettings.Ports"), 2)
	}
	if !in.HasHealthcheck {
		add(finding("Health check is not configured", SeverityLow, "none",
			"Runtime failure may not be detected promptly by Docker or operators.",
			"Add an application-specific health check with safe timing and timeout values.", false, true,
			"An incorrect health check can mark a working application unhealthy.", "Config.Healthcheck"), 3)
	}

	seccomp := seccompState(in.SecurityOptions)
	if seccomp == "unconfined" {
		add(finding("Seccomp protection disabled", SeverityHigh, seccomp,
			"Unfiltered system calls expose more kernel attack surface.",
			"Use Docker's default seccomp profile or a reviewed custom profile.", true, true,
			"Applications requiring blocked system calls may fail.", "HostConfig.SecurityOpt"), 12)
	} else if seccomp == "default/daemon policy" {
		add(finding("Seccomp profile uses daemon policy", SeverityInformational, seccomp,
			"The effective profile depends on Docker daemon configuration.",
			"Verify that the daemon default seccomp profile is enabled.", false, false,
			"No impact from this read-only verification.", "HostConfig.SecurityOpt"), 0)
	}
	if strings.TrimSpace(in.AppArmorProfile) == "" && !hasSELinuxLabel(in.SecurityOptions) {
		add(finding("Mandatory access-control protection not detected", SeverityLow, "no AppArmor profile or SELinux label reported",
			"AppArmor or SELinux can limit the impact of a container escape or process compromise.",
			"Enable a reviewed AppArmor profile or SELinux labeling when the host supports it.", false, true,
			"Host support and workload-specific policy validation are required.", "AppArmorProfile/HostConfig.SecurityOpt"), 4)
	}

	for _, item := range in.Environment {
		name, _, ok := strings.Cut(item, "=")
		if ok && IsSensitiveName(name) {
			add(finding("Possible secret in environment", SeverityHigh, name+"=[REDACTED]",
				"Environment variables can be exposed through container inspection, process diagnostics, or application errors.",
				"Use Docker secrets or a dedicated secret manager and rotate the value if exposure is suspected.", false, true,
				"Changing secret delivery requires application and deployment changes.", "Config.Env["+name+"]"), 8)
		}
	}
	if isMutableImageReference(in.ImageReference) {
		add(finding("Mutable image reference", SeverityMedium, in.ImageReference,
			"A mutable tag can resolve to different image contents over time.",
			"Pin deployments to a reviewed immutable image digest while retaining a readable tag in deployment metadata.", false, true,
			"Digest pinning requires an explicit update workflow.", "Config.Image"), 5)
	}
	if report.Compose {
		add(finding("Container is managed by Docker Compose", SeverityInformational, composeValue(in.Labels),
			"Manual recreation can be overwritten by the next Compose reconciliation.",
			"Apply future hardening through a reviewed Compose override, not direct recreation.", false, true,
			"Compose configuration and service dependencies must be validated together.", "Config.Labels[com.docker.compose.*]"), 0)
	}

	if report.RuntimeScore.Value < 0 {
		report.RuntimeScore.Value = 0
	}
	sort.SliceStable(report.Findings, func(i, j int) bool {
		return severityRank(report.Findings[i].Severity) > severityRank(report.Findings[j].Severity)
	})
	report.Controls = GenerateHardeningPlan(in, nil).Controls
	return report
}

func finding(title string, severity Severity, value, risk, remediation string, automatic, recreation bool, impact, property string) SecurityFinding {
	return SecurityFinding{Title: title, Severity: severity, CurrentValue: value, Risk: risk, Remediation: remediation,
		Automatic: automatic, RecreationRequired: recreation, CompatibilityImpact: impact, Property: property}
}

func runsAsRoot(user string) bool {
	user = strings.TrimSpace(user)
	if user == "" || strings.EqualFold(user, "root") {
		return true
	}
	id, _, _ := strings.Cut(user, ":")
	return id == "0"
}

func displayUser(user string) string {
	if strings.TrimSpace(user) == "" {
		return "root (image default/empty Config.User)"
	}
	return user
}

type capabilityAssessment struct {
	risky    bool
	severity Severity
	value    string
	points   int
}

func capabilityState(add, drop []string) capabilityAssessment {
	add = upper(add)
	drop = upper(drop)
	dangerous := []string{"ALL", "SYS_ADMIN", "SYS_PTRACE", "SYS_MODULE", "DAC_READ_SEARCH", "NET_ADMIN"}
	for _, cap := range add {
		if contains(dangerous, cap) {
			return capabilityAssessment{true, SeverityHigh, "added: " + strings.Join(add, ", "), 12}
		}
	}
	if !contains(drop, "ALL") {
		value := "Docker default capability set retained"
		if len(add) > 0 {
			value += "; added: " + strings.Join(add, ", ")
		}
		if len(drop) > 0 {
			value += "; dropped: " + strings.Join(drop, ", ")
		}
		return capabilityAssessment{true, SeverityMedium, value, 6}
	}
	if len(add) > 0 {
		return capabilityAssessment{true, SeverityMedium, "drop ALL; added: " + strings.Join(add, ", "), 4}
	}
	return capabilityAssessment{}
}

func upper(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strings.ToUpper(strings.TrimPrefix(value, "CAP_")))
	}
	return out
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func IsSensitiveName(name string) bool {
	name = strings.ToLower(name)
	for _, marker := range []string{"password", "passwd", "token", "secret", "credential", "api_key", "api-key", "private_key", "private-key"} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func isDockerSocket(m Mount) bool {
	return strings.HasSuffix(m.Source, "/docker.sock") || strings.HasSuffix(m.Destination, "/docker.sock")
}

func isSensitiveHostMount(m Mount) bool {
	if m.Type != "bind" {
		return false
	}
	source := strings.TrimSuffix(m.Source, "/")
	for _, path := range []string{"/", "/boot", "/dev", "/etc", "/proc", "/root", "/run", "/sys", "/var/run", "/var/lib/docker", "/home"} {
		if source == path || (path != "/" && strings.HasPrefix(source, path+"/")) {
			return true
		}
	}
	return false
}

func isBroadWritableMount(m Mount) bool {
	if !m.ReadWrite || m.Type != "bind" {
		return false
	}
	source := strings.TrimSuffix(m.Source, "/")
	return source == "/" || source == "/etc" || source == "/home" || source == "/root" || source == "/usr" || source == "/var"
}

func mountMode(m Mount) string {
	if m.ReadWrite {
		return "read-write"
	}
	return "read-only"
}

func hasResourceLimit(in AuditInput) bool {
	return in.MemoryLimit > 0 || in.NanoCPUs > 0 || in.CPUQuota > 0 || in.CPUShares > 0
}

func seccompState(options []string) string {
	for _, option := range options {
		value := strings.ToLower(option)
		if strings.Contains(value, "seccomp=unconfined") {
			return "unconfined"
		}
		if strings.HasPrefix(value, "seccomp=") {
			return strings.TrimPrefix(option, "seccomp=")
		}
	}
	return "default/daemon policy"
}

func hasSELinuxLabel(options []string) bool {
	for _, option := range options {
		if strings.HasPrefix(strings.ToLower(option), "label=") {
			return true
		}
	}
	return false
}

func isMutableImageReference(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return true
	}
	return !strings.Contains(ref, "@sha256:")
}

func isCompose(labels map[string]string) bool {
	return labels["com.docker.compose.project"] != "" || labels["com.docker.compose.service"] != ""
}

func composeValue(labels map[string]string) string {
	project, service := labels["com.docker.compose.project"], labels["com.docker.compose.service"]
	if project == "" && service == "" {
		return "Compose labels detected"
	}
	return fmt.Sprintf("project=%s service=%s", project, service)
}

func severityRank(severity Severity) int {
	return map[Severity]int{SeverityCritical: 5, SeverityHigh: 4, SeverityMedium: 3, SeverityLow: 2, SeverityInformational: 1}[severity]
}
