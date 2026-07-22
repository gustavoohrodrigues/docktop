package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	containertypes "github.com/docker/docker/api/types/container"
	"strings"
	"time"
)

func (s *SDK) Start(c context.Context, id string) error {
	return s.cli.ContainerStart(c, id, containertypes.StartOptions{})
}
func (s *SDK) Stop(c context.Context, id string) error {
	t := 10
	return s.cli.ContainerStop(c, id, containertypes.StopOptions{Timeout: &t})
}
func (s *SDK) Restart(c context.Context, id string) error {
	t := 10
	return s.cli.ContainerRestart(c, id, containertypes.StopOptions{Timeout: &t})
}
func (s *SDK) Pause(c context.Context, id string) error   { return s.cli.ContainerPause(c, id) }
func (s *SDK) Unpause(c context.Context, id string) error { return s.cli.ContainerUnpause(c, id) }
func (s *SDK) Remove(c context.Context, id string, force bool) error {
	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel()
	return s.cli.ContainerRemove(ctx, id, containertypes.RemoveOptions{Force: force})
}

// UpdateImage recreates a container with the newest version of its configured
// image. Docker cannot change a container image in place, therefore the old
// container is retained under a temporary name until the replacement starts.
func (s *SDK) UpdateImage(ctx context.Context, id string, progress func(string)) (string, error) {
	old, err := s.cli.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}
	if old.Config != nil && old.Config.Labels["com.docker.swarm.service.id"] != "" {
		return "", errors.New("container pertence a um serviço Swarm; atualize a imagem na tela Services para preservar o estado desejado")
	}
	ref := strings.TrimSpace(old.Config.Image)
	if ref == "" {
		return "", errors.New("container não possui referência de imagem atualizável")
	}
	if progress != nil {
		progress("baixando " + ref)
	}
	if err = s.Pull(ctx, ref, progress); err != nil {
		return "", fmt.Errorf("pull de %s: %w", ref, err)
	}
	latest, _, err := s.cli.ImageInspectWithRaw(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("inspecionar imagem baixada: %w", err)
	}
	if old.Image == latest.ID {
		return "imagem já estava atualizada: " + ref, nil
	}

	// Clone the daemon response before changing Image; this also prevents a
	// concurrent refresh from observing partially changed SDK structures.
	var cfg containertypes.Config
	var host containertypes.HostConfig
	if err = cloneJSON(old.Config, &cfg); err != nil {
		return "", err
	}
	if err = cloneJSON(old.HostConfig, &host); err != nil {
		return "", err
	}
	preserveVolumeMounts(old.Mounts, &host)
	cfg.Image = ref
	name := strings.TrimPrefix(old.Name, "/")
	backup := name + ".docktop-backup-" + time.Now().Format("20060102-150405")
	wasRunning := old.State != nil && old.State.Running
	if wasRunning {
		t := 10
		if err = s.cli.ContainerStop(ctx, id, containertypes.StopOptions{Timeout: &t}); err != nil {
			return "", fmt.Errorf("parar container antigo: %w", err)
		}
	}
	if err = s.cli.ContainerRename(ctx, id, backup); err != nil {
		if wasRunning {
			_ = s.cli.ContainerStart(context.WithoutCancel(ctx), id, containertypes.StartOptions{})
		}
		return "", fmt.Errorf("preparar substituição: %w", err)
	}

	created, createErr := s.cli.ContainerCreate(ctx, &cfg, &host, nil, nil, name)
	if createErr != nil {
		s.rollbackUpdate(id, "", name, wasRunning)
		return "", fmt.Errorf("criar substituto; rollback aplicado: %w", createErr)
	}
	if wasRunning {
		if err = s.cli.ContainerStart(ctx, created.ID, containertypes.StartOptions{}); err != nil {
			s.rollbackUpdate(id, created.ID, name, true)
			return "", fmt.Errorf("iniciar substituto; rollback aplicado: %w", err)
		}
	}
	if err = s.cli.ContainerRemove(ctx, id, containertypes.RemoveOptions{}); err != nil {
		return created.ID, fmt.Errorf("container atualizado, mas backup %s não pôde ser removido: %w", backup, err)
	}
	return created.ID, nil
}

func preserveVolumeMounts(mounts []containertypes.MountPoint, host *containertypes.HostConfig) {
	targets := make(map[string]bool)
	for _, bind := range host.Binds {
		parts := strings.Split(bind, ":")
		if len(parts) > 1 {
			targets[parts[1]] = true
		}
	}
	for _, mount := range mounts {
		if string(mount.Type) != "volume" || mount.Name == "" || targets[mount.Destination] {
			continue
		}
		bind := mount.Name + ":" + mount.Destination
		if !mount.RW {
			bind += ":ro"
		}
		host.Binds = append(host.Binds, bind)
		targets[mount.Destination] = true
	}
}

func cloneJSON(src, dst any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("copiar configuração do container: %w", err)
	}
	if err = json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("copiar configuração do container: %w", err)
	}
	return nil
}

func (s *SDK) rollbackUpdate(oldID, newID, originalName string, restart bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if newID != "" {
		_ = s.cli.ContainerRemove(ctx, newID, containertypes.RemoveOptions{Force: true})
	}
	_ = s.cli.ContainerRename(ctx, oldID, originalName)
	if restart {
		_ = s.cli.ContainerStart(ctx, oldID, containertypes.StartOptions{})
	}
}
