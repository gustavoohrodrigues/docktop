package docker

import (
	"context"
	containertypes "github.com/docker/docker/api/types/container"
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
