package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/docktop/docktop/internal/security"
)

func (s *SDK) metrics(ctx context.Context, containers []container.Summary) map[string]ContainerMetric {
	out := make(map[string]ContainerMetric)
	var mu sync.Mutex
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res, err := s.cli.ContainerStatsOneShot(ctx, id)
			if err != nil {
				return
			}
			defer res.Body.Close()
			var st container.StatsResponse
			if json.NewDecoder(res.Body).Decode(&st) != nil {
				return
			}
			cpuDelta := float64(st.CPUStats.CPUUsage.TotalUsage - st.PreCPUStats.CPUUsage.TotalUsage)
			sysDelta := float64(st.CPUStats.SystemUsage - st.PreCPUStats.SystemUsage)
			cpus := float64(st.CPUStats.OnlineCPUs)
			if cpus == 0 {
				cpus = 1
			}
			cpu := 0.0
			if sysDelta > 0 && cpuDelta > 0 {
				cpu = cpuDelta / sysDelta * cpus * 100
			}
			limit := st.MemoryStats.Limit
			mem := 0.0
			if limit > 0 {
				mem = float64(st.MemoryStats.Usage) / float64(limit) * 100
			}
			mu.Lock()
			out[id] = ContainerMetric{CPU: cpu, Memory: mem, MemoryBytes: st.MemoryStats.Usage}
			mu.Unlock()
		}(c.ID)
	}
	wg.Wait()
	return out
}

func (s *SDK) Logs(ctx context.Context, id string, tail int) (string, error) {
	r, err := s.cli.ContainerLogs(ctx, id, container.LogsOptions{ShowStdout: true, ShowStderr: true, Timestamps: true, Tail: fmt.Sprint(tail)})
	if err != nil {
		return "", err
	}
	defer r.Close()
	var stdout, stderr bytes.Buffer
	if _, err = stdcopy.StdCopy(&stdout, &stderr, r); err != nil {
		b, readErr := io.ReadAll(r)
		if readErr != nil {
			return "", err
		}
		stdout.Write(b)
	}
	return stdout.String() + stderr.String(), nil
}

func (s *SDK) Inspect(ctx context.Context, id string) (string, error) {
	v, err := s.cli.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	return string(b), err
}

// SecurityAudit is read-only. It inspects the selected container once, maps
// daemon-owned structures to the security package's stable input, and never
// starts the workload or invokes commands inside it.
func (s *SDK) SecurityAudit(ctx context.Context, id string) (security.ContainerSecurityReport, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	v, err := s.cli.ContainerInspect(ctx, id)
	if err != nil {
		return security.ContainerSecurityReport{}, friendly(err, s.endpoint)
	}
	if v.Config == nil || v.HostConfig == nil {
		return security.ContainerSecurityReport{}, errors.New("inspeção do container retornou configuração incompleta")
	}
	return security.AuditContainer(auditInputFromInspect(v), time.Now()), nil
}

func auditInputFromInspect(v container.InspectResponse) security.AuditInput {
	in := security.AuditInput{
		ContainerID:     v.ID,
		ContainerName:   strings.TrimPrefix(v.Name, "/"),
		ImageReference:  v.Config.Image,
		User:            v.Config.User,
		Privileged:      v.HostConfig.Privileged,
		ReadonlyRootFS:  v.HostConfig.ReadonlyRootfs,
		CapAdd:          append([]string(nil), v.HostConfig.CapAdd...),
		CapDrop:         append([]string(nil), v.HostConfig.CapDrop...),
		SecurityOptions: append([]string(nil), v.HostConfig.SecurityOpt...),
		PIDMode:         string(v.HostConfig.PidMode),
		IPCMode:         string(v.HostConfig.IpcMode),
		NetworkMode:     string(v.HostConfig.NetworkMode),
		UserNSMode:      string(v.HostConfig.UsernsMode),
		MemoryLimit:     v.HostConfig.Memory,
		MemorySwap:      v.HostConfig.MemorySwap,
		NanoCPUs:        v.HostConfig.NanoCPUs,
		CPUQuota:        v.HostConfig.CPUQuota,
		CPUShares:       v.HostConfig.CPUShares,
		HasHealthcheck:  v.Config.Healthcheck != nil,
		AppArmorProfile: v.AppArmorProfile,
		Environment:     append([]string(nil), v.Config.Env...),
		Labels:          cloneStrings(v.Config.Labels),
		Tmpfs:           cloneStrings(v.HostConfig.Tmpfs),
	}
	for _, limit := range v.HostConfig.Ulimits {
		if limit != nil {
			in.Ulimits = append(in.Ulimits, security.Ulimit{Name: limit.Name, Soft: limit.Soft, Hard: limit.Hard})
		}
	}
	for _, option := range v.HostConfig.SecurityOpt {
		if strings.EqualFold(option, "no-new-privileges") || strings.EqualFold(option, "no-new-privileges=true") {
			in.NoNewPrivileges = true
		}
	}
	if v.HostConfig.PidsLimit != nil {
		in.PidsLimit = *v.HostConfig.PidsLimit
	}
	for _, mount := range v.Mounts {
		in.Mounts = append(in.Mounts, security.Mount{
			Type: string(mount.Type), Source: mount.Source, Destination: mount.Destination, ReadWrite: mount.RW,
		})
	}
	for _, device := range v.HostConfig.Devices {
		in.Devices = append(in.Devices, device.PathOnHost+" → "+device.PathInContainer)
	}
	if v.NetworkSettings != nil {
		for port, bindings := range v.NetworkSettings.Ports {
			if len(bindings) == 0 {
				continue
			}
			for _, binding := range bindings {
				host := binding.HostIP
				if host == "" {
					host = "*"
				}
				in.PublishedPorts = append(in.PublishedPorts, host+":"+binding.HostPort+"→"+string(port))
			}
		}
	}
	sort.Strings(in.PublishedPorts)
	return in
}

func cloneStrings(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func (s *SDK) PrepareHardening(ctx context.Context, id string, selected []security.HardeningControlID) (security.HardeningPlan, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	v, err := s.cli.ContainerInspect(ctx, id)
	if err != nil {
		return security.HardeningPlan{}, friendly(err, s.endpoint)
	}
	if v.Config == nil || v.HostConfig == nil {
		return security.HardeningPlan{}, errors.New("inspeção do container retornou configuração incompleta")
	}
	plan := security.GenerateHardeningPlan(auditInputFromInspect(v), selected)
	if plan.ComposeManaged {
		return plan, errors.New("container gerenciado por Docker Compose; hardening direto foi bloqueado para não divergir do projeto Compose")
	}
	if v.Config.Labels["com.docker.swarm.service.id"] != "" {
		return plan, errors.New("container pertence a um serviço Swarm; altere o service spec em vez de recriar a task")
	}
	return plan, nil
}

func (s *SDK) ApplyHardening(ctx context.Context, expectedID string, selected []security.HardeningControlID) (security.HardeningResult, error) {
	if err := security.ValidateControlSelection(selected); err != nil {
		return security.HardeningResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	old, err := s.cli.ContainerInspect(ctx, expectedID)
	if err != nil {
		return security.HardeningResult{}, friendly(err, s.endpoint)
	}
	if old.ID != expectedID {
		return security.HardeningResult{}, errors.New("container mudou desde a inspeção; execute o hardening novamente")
	}
	if old.Config == nil || old.HostConfig == nil {
		return security.HardeningResult{}, errors.New("inspeção do container retornou configuração incompleta")
	}
	plan := security.GenerateHardeningPlan(auditInputFromInspect(old), selected)
	if plan.ComposeManaged {
		return security.HardeningResult{}, errors.New("container gerenciado por Docker Compose; gere um override antes de aplicar hardening")
	}
	if old.Config.Labels["com.docker.swarm.service.id"] != "" {
		return security.HardeningResult{}, errors.New("container pertence a um serviço Swarm; hardening deve alterar o service spec")
	}
	selected = security.SelectedControlIDs(plan)
	if len(selected) == 0 {
		return security.HardeningResult{}, errors.New("os controles selecionados já estão aplicados")
	}

	var cfg container.Config
	var host container.HostConfig
	if err = cloneJSON(old.Config, &cfg); err != nil {
		return security.HardeningResult{}, err
	}
	if err = cloneJSON(old.HostConfig, &host); err != nil {
		return security.HardeningResult{}, err
	}
	preserveVolumeMounts(old.Mounts, &host)
	applySelectedControls(&cfg, &host, selected)

	name := strings.TrimPrefix(old.Name, "/")
	backup := name + ".docktop-before-hardening-" + time.Now().Format("20060102-150405")
	wasRunning := old.State != nil && old.State.Running
	if wasRunning {
		stopTimeout := 10
		if err = s.cli.ContainerStop(ctx, old.ID, container.StopOptions{Timeout: &stopTimeout}); err != nil {
			return security.HardeningResult{}, fmt.Errorf("parar container original: %w", err)
		}
	}
	latest, inspectErr := s.cli.ContainerInspect(ctx, old.ID)
	if inspectErr != nil || latest.ID != expectedID {
		if wasRunning {
			_ = s.cli.ContainerStart(context.WithoutCancel(ctx), old.ID, container.StartOptions{})
		}
		return security.HardeningResult{}, errors.New("container mudou durante a operação; configuração original preservada")
	}
	if err = s.cli.ContainerRename(ctx, old.ID, backup); err != nil {
		if wasRunning {
			_ = s.cli.ContainerStart(context.WithoutCancel(ctx), old.ID, container.StartOptions{})
		}
		return security.HardeningResult{}, fmt.Errorf("preservar container original: %w", err)
	}

	networking := preservedNetworking(old, string(host.NetworkMode))
	created, createErr := s.cli.ContainerCreate(ctx, &cfg, &host, networking, nil, name)
	if createErr != nil {
		rollbackErr := s.rollbackHardening(old.ID, "", name, wasRunning)
		return security.HardeningResult{BackupContainerName: backup, RollbackApplied: rollbackErr == nil},
			hardeningFailure("criar substituto", createErr, rollbackErr)
	}
	if wasRunning {
		if err = s.cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
			rollbackErr := s.rollbackHardening(old.ID, created.ID, name, true)
			return security.HardeningResult{ContainerID: created.ID, BackupContainerName: backup, RollbackApplied: rollbackErr == nil},
				hardeningFailure("iniciar substituto", err, rollbackErr)
		}
	}
	running, healthy, validationErr := s.validateReplacement(ctx, created.ID, wasRunning, cfg.Healthcheck != nil)
	if validationErr != nil {
		rollbackErr := s.rollbackHardening(old.ID, created.ID, name, wasRunning)
		return security.HardeningResult{ContainerID: created.ID, BackupContainerName: backup, RollbackApplied: rollbackErr == nil},
			hardeningFailure("validar substituto", validationErr, rollbackErr)
	}
	return security.HardeningResult{
		ContainerID: created.ID, ContainerName: name, BackupContainerName: backup,
		AppliedControls: append([]security.HardeningControlID(nil), selected...), Running: running, Healthy: healthy,
	}, nil
}

func applySelectedControls(cfg *container.Config, host *container.HostConfig, selected []security.HardeningControlID) {
	for _, id := range selected {
		switch id {
		case security.ControlNoNewPrivileges:
			if !hasSecurityOption(host.SecurityOpt, "no-new-privileges") {
				host.SecurityOpt = append(host.SecurityOpt, "no-new-privileges=true")
			}
		case security.ControlDisablePrivileged:
			host.Privileged = false
		case security.ControlDropCapabilities:
			host.CapDrop = []string{"ALL"}
			host.CapAdd = nil
		case security.ControlReadOnlyRootFS:
			host.ReadonlyRootfs = true
		case security.ControlNonRootUser:
			cfg.User = "65532:65532"
		case security.ControlPIDLimit:
			limit := int64(512)
			host.PidsLimit = &limit
		case security.ControlMemoryLimit:
			host.Memory = 512 << 20
			if host.MemorySwap > 0 && host.MemorySwap < host.Memory {
				host.MemorySwap = host.Memory
			}
		case security.ControlPrivatePID:
			host.PidMode = ""
		case security.ControlPrivateIPC:
			host.IpcMode = ""
		case security.ControlPrivateNetwork:
			host.NetworkMode = "default"
		case security.ControlRemoveDockerSocket:
			host.Binds = filterDockerSocketBinds(host.Binds)
			host.Mounts = filterDockerSocketMounts(host.Mounts)
		case security.ControlRemoveDevices:
			host.Devices = nil
			host.DeviceCgroupRules = nil
			host.DeviceRequests = nil
		case security.ControlCPULimit:
			if host.NanoCPUs <= 0 && host.CPUQuota <= 0 {
				host.NanoCPUs = 1_000_000_000
			}
		case security.ControlSwapLimit:
			if host.Memory <= 0 {
				host.Memory = 512 << 20
			}
			host.MemorySwap = 512 << 20
		case security.ControlNoFileLimit:
			setUlimit(&host.Ulimits, "nofile", 1024, 4096)
		case security.ControlDefaultSeccomp:
			host.SecurityOpt = filterSecurityOption(host.SecurityOpt, "seccomp=")
		case security.ControlAppArmor:
			host.SecurityOpt = filterSecurityOption(host.SecurityOpt, "apparmor=")
			host.SecurityOpt = append(host.SecurityOpt, "apparmor=docker-default")
		case security.ControlPrivateUserNS:
			host.UsernsMode = ""
		case security.ControlTmpfsTmp:
			if host.Tmpfs == nil {
				host.Tmpfs = map[string]string{}
			}
			host.Tmpfs["/tmp"] = "rw,nosuid,nodev,noexec,size=64m"
		case security.ControlTmpfsRun:
			if host.Tmpfs == nil {
				host.Tmpfs = map[string]string{}
			}
			host.Tmpfs["/run"] = "rw,nosuid,nodev,noexec,size=16m"
		case security.ControlReadOnlySensitive:
			host.Binds = readOnlySensitiveBinds(host.Binds)
			for index := range host.Mounts {
				if sensitiveHostPath(host.Mounts[index].Source) {
					host.Mounts[index].ReadOnly = true
				}
			}
		case security.ControlRemovePorts:
			host.PortBindings = nil
		}
	}
}

func setUlimit(values *[]*units.Ulimit, name string, soft, hard int64) {
	for _, item := range *values {
		if item != nil && item.Name == name {
			item.Soft, item.Hard = soft, hard
			return
		}
	}
	*values = append(*values, &units.Ulimit{Name: name, Soft: soft, Hard: hard})
}

func filterSecurityOption(options []string, prefix string) []string {
	out := options[:0]
	for _, option := range options {
		if strings.HasPrefix(strings.ToLower(option), prefix) {
			continue
		}
		out = append(out, option)
	}
	return out
}

func readOnlySensitiveBinds(binds []string) []string {
	out := make([]string, 0, len(binds))
	for _, bind := range binds {
		parts := strings.Split(bind, ":")
		if len(parts) < 2 || !sensitiveHostPath(parts[0]) {
			out = append(out, bind)
			continue
		}
		mode := "ro"
		if len(parts) > 2 {
			options := strings.Split(parts[2], ",")
			filtered := options[:0]
			for _, option := range options {
				if option != "rw" && option != "ro" {
					filtered = append(filtered, option)
				}
			}
			filtered = append(filtered, "ro")
			mode = strings.Join(filtered, ",")
		}
		out = append(out, parts[0]+":"+parts[1]+":"+mode)
	}
	return out
}

func sensitiveHostPath(source string) bool {
	source = strings.TrimSuffix(source, "/")
	for _, path := range []string{"/", "/boot", "/dev", "/etc", "/proc", "/root", "/run", "/sys", "/var/run", "/var/lib/docker", "/home"} {
		if source == path || (path != "/" && strings.HasPrefix(source, path+"/")) {
			return true
		}
	}
	return false
}

func hasSecurityOption(options []string, wanted string) bool {
	for _, option := range options {
		if strings.EqualFold(option, wanted) || strings.EqualFold(option, wanted+"=true") {
			return true
		}
	}
	return false
}

func filterDockerSocketBinds(binds []string) []string {
	out := binds[:0]
	for _, bind := range binds {
		parts := strings.Split(bind, ":")
		if len(parts) > 0 && strings.HasSuffix(parts[0], "/docker.sock") {
			continue
		}
		if len(parts) > 1 && strings.HasSuffix(parts[1], "/docker.sock") {
			continue
		}
		out = append(out, bind)
	}
	return out
}

func filterDockerSocketMounts(mounts []mount.Mount) []mount.Mount {
	out := mounts[:0]
	for _, item := range mounts {
		if strings.HasSuffix(item.Source, "/docker.sock") || strings.HasSuffix(item.Target, "/docker.sock") {
			continue
		}
		out = append(out, item)
	}
	return out
}

func preservedNetworking(old container.InspectResponse, networkMode string) *network.NetworkingConfig {
	out := &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{}}
	if old.NetworkSettings == nil {
		return out
	}
	for name, endpoint := range old.NetworkSettings.Networks {
		if name == "host" || name == "none" || networkMode == "host" || networkMode == "none" {
			continue
		}
		if endpoint == nil {
			continue
		}
		out.EndpointsConfig[name] = &network.EndpointSettings{
			Aliases: append([]string(nil), endpoint.Aliases...), Links: append([]string(nil), endpoint.Links...),
			DriverOpts: cloneStrings(endpoint.DriverOpts), GwPriority: endpoint.GwPriority,
		}
	}
	return out
}

func (s *SDK) validateReplacement(ctx context.Context, id string, shouldRun, hasHealthcheck bool) (bool, bool, error) {
	if !shouldRun {
		return false, false, nil
	}
	started := time.Now()
	for {
		state, err := s.cli.ContainerInspect(ctx, id)
		if err != nil {
			return false, false, err
		}
		if state.State == nil || !state.State.Running {
			message := "container substituto não permaneceu em execução"
			if state.State != nil && state.State.Error != "" {
				message += ": " + state.State.Error
			}
			return false, false, errors.New(message)
		}
		if !hasHealthcheck || state.State.Health == nil {
			if time.Since(started) >= 2*time.Second {
				return true, false, nil
			}
		} else {
			switch state.State.Health.Status {
			case "healthy":
				return true, true, nil
			case "unhealthy":
				return true, false, errors.New("health check do substituto retornou unhealthy")
			}
		}
		select {
		case <-ctx.Done():
			return true, false, errors.New("tempo esgotado aguardando health check do substituto")
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *SDK) rollbackHardening(oldID, failedID, originalName string, restart bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var problems []string
	if failedID != "" {
		_ = s.cli.ContainerStop(ctx, failedID, container.StopOptions{})
		diagnosticName := originalName + ".docktop-hardening-failed-" + time.Now().Format("20060102-150405")
		if err := s.cli.ContainerRename(ctx, failedID, diagnosticName); err != nil {
			if removeErr := s.cli.ContainerRemove(ctx, failedID, container.RemoveOptions{Force: true}); removeErr != nil {
				problems = append(problems, "preservar/remover substituto falho: "+removeErr.Error())
			}
		}
	}
	if err := s.cli.ContainerRename(ctx, oldID, originalName); err != nil {
		problems = append(problems, "restaurar nome original: "+err.Error())
	}
	if restart {
		if err := s.cli.ContainerStart(ctx, oldID, container.StartOptions{}); err != nil {
			problems = append(problems, "reiniciar original: "+err.Error())
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func hardeningFailure(stage string, operationErr, rollbackErr error) error {
	if rollbackErr != nil {
		return fmt.Errorf("%s falhou: %v; rollback também falhou: %v", stage, operationErr, rollbackErr)
	}
	return fmt.Errorf("%s falhou; configuração original restaurada: %w", stage, operationErr)
}

func (s *SDK) Processes(ctx context.Context, id string) (string, error) {
	v, err := s.cli.ContainerTop(ctx, id, nil)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(strings.Join(v.Titles, "  ") + "\n")
	for _, p := range v.Processes {
		b.WriteString(strings.Join(p, "  ") + "\n")
	}
	return b.String(), nil
}

func (s *SDK) Pull(ctx context.Context, ref string, progress func(string)) error {
	if !strings.Contains(ref, ":") {
		ref += ":latest"
	}
	r, err := s.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return err
	}
	defer r.Close()
	dec := json.NewDecoder(r)
	for {
		var event struct{ Status, ID, Error string }
		if err = dec.Decode(&event); errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if event.Error != "" {
			return errors.New(event.Error)
		}
		line := strings.TrimSpace(event.ID + " " + event.Status)
		if line != "" && progress != nil {
			progress(line)
		}
	}
}

func (s *SDK) CreateContainer(ctx context.Context, req CreateRequest) (string, error) {
	if strings.TrimSpace(req.Image) == "" {
		return "", errors.New("imagem é obrigatória")
	}
	if strings.TrimSpace(req.Name) == "" {
		return "", errors.New("nome do container é obrigatório")
	}
	if _, _, err := s.cli.ImageInspectWithRaw(ctx, req.Image); err != nil {
		if err = s.Pull(ctx, req.Image, nil); err != nil {
			return "", fmt.Errorf("imagem não existe localmente e o pull falhou: %w", err)
		}
	}
	cmd := strings.Fields(req.Command)
	exposed, bindings, err := nat.ParsePortSpecs(req.Ports)
	if err != nil {
		return "", fmt.Errorf("portas inválidas: %w", err)
	}
	restart := req.Restart
	if restart == "" {
		restart = "no"
	}
	validRestart := map[string]bool{"no": true, "always": true, "unless-stopped": true, "on-failure": true}
	if !validRestart[restart] {
		return "", errors.New("restart deve ser no, always, unless-stopped ou on-failure")
	}
	host := &container.HostConfig{Binds: req.Volumes, PortBindings: bindings, RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyMode(restart)}}
	conf := &container.Config{Image: req.Image, Cmd: cmd, Tty: false, Env: req.Env, ExposedPorts: exposed}
	res, err := s.cli.ContainerCreate(ctx, conf, host, nil, nil, req.Name)
	if err != nil {
		return "", err
	}
	if err = s.cli.ContainerStart(ctx, res.ID, container.StartOptions{}); err != nil {
		return res.ID, fmt.Errorf("criado, mas não iniciado: %w", err)
	}
	return res.ID, nil
}

func (s *SDK) RemoveImage(ctx context.Context, id string, force bool) error {
	_, err := s.cli.ImageRemove(ctx, id, image.RemoveOptions{Force: force, PruneChildren: true})
	return err
}
func (s *SDK) CreateVolume(ctx context.Context, name, driver string) error {
	if driver == "" {
		driver = "local"
	}
	_, err := s.cli.VolumeCreate(ctx, volume.CreateOptions{Name: name, Driver: driver})
	return err
}
func (s *SDK) RemoveVolume(ctx context.Context, name string, force bool) error {
	return s.cli.VolumeRemove(ctx, name, force)
}
func (s *SDK) CreateNetwork(ctx context.Context, name, driver string) error {
	if driver == "" {
		driver = "bridge"
	}
	_, err := s.cli.NetworkCreate(ctx, name, network.CreateOptions{Driver: driver})
	return err
}
func (s *SDK) RemoveNetwork(ctx context.Context, id string) error {
	return s.cli.NetworkRemove(ctx, id)
}

func (s *SDK) ShellCommand(ctx context.Context, id string) (*exec.Cmd, error) {
	inspect, err := s.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}
	if !inspect.State.Running {
		return nil, errors.New("container não está em execução")
	}
	bin, err := exec.LookPath("docker")
	if err != nil {
		return nil, errors.New("Docker CLI é necessário para o terminal interativo")
	}
	shells := []string{"/bin/bash", "/bin/sh", "bash", "sh", "ash"}
	for _, shell := range shells {
		probe, probeErr := s.cli.ContainerExecCreate(ctx, id, container.ExecOptions{Cmd: []string{shell, "-c", "exit 0"}})
		if probeErr != nil {
			continue
		}
		if probeErr = s.cli.ContainerExecStart(ctx, probe.ID, container.ExecStartOptions{Detach: true}); probeErr != nil {
			continue
		}
		for range 10 {
			state, inspectErr := s.cli.ContainerExecInspect(ctx, probe.ID)
			if inspectErr == nil && !state.Running {
				if state.ExitCode == 0 {
					// O CLI recebe argumentos estruturados; ele cuida do raw TTY e devolve o terminal à TUI.
					args := []string{"--host", s.endpoint}
					if s.tls.Enabled {
						args = append(args, "--tlsverify", "--tlscacert", s.tls.CAFile, "--tlscert", s.tls.CertFile, "--tlskey", s.tls.KeyFile)
					}
					args = append(args, "exec", "-it", id, shell)
					return exec.CommandContext(ctx, bin, args...), nil
				}
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	return nil, errors.New("container não possui bash, sh ou ash")
}
