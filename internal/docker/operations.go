package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
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
