package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/swarm"
)

func (s *SDK) collectSwarm(ctx context.Context, out *Snapshot) {
	services, err := s.cli.ServiceList(ctx, swarm.ServiceListOptions{Status: true})
	if err != nil {
		return
	}
	tasks, _ := s.cli.TaskList(ctx, swarm.TaskListOptions{})
	nodes, _ := s.cli.NodeList(ctx, swarm.NodeListOptions{})
	serviceNames := map[string]string{}
	nodeNames := map[string]string{}
	for _, n := range nodes {
		manager := "-"
		if n.ManagerStatus != nil {
			manager = string(n.ManagerStatus.Reachability)
			if n.ManagerStatus.Leader {
				manager = "leader"
			}
		}
		out.Nodes = append(out.Nodes, Node{ID: n.ID, Hostname: n.Description.Hostname, Role: string(n.Spec.Role), Availability: string(n.Spec.Availability), State: string(n.Status.State), Manager: manager, Engine: n.Description.Engine.EngineVersion, CPUs: n.Description.Resources.NanoCPUs / 1e9, Memory: n.Description.Resources.MemoryBytes})
		nodeNames[n.ID] = n.Description.Hostname
	}
	stackMap := map[string]*Stack{}
	for _, svc := range services {
		mode := "global"
		var desired, running uint64
		if svc.Spec.Mode.Replicated != nil {
			mode = "replicated"
			if svc.Spec.Mode.Replicated.Replicas != nil {
				desired = *svc.Spec.Mode.Replicated.Replicas
			}
		}
		if svc.ServiceStatus != nil {
			desired = svc.ServiceStatus.DesiredTasks
			running = svc.ServiceStatus.RunningTasks
		}
		image := "-"
		if svc.Spec.TaskTemplate.ContainerSpec != nil {
			image = svc.Spec.TaskTemplate.ContainerSpec.Image
		}
		stack := svc.Spec.Labels["com.docker.stack.namespace"]
		update := "-"
		if svc.UpdateStatus != nil {
			update = string(svc.UpdateStatus.State)
		}
		out.Services = append(out.Services, Service{ID: svc.ID, Name: svc.Spec.Name, Stack: stack, Image: image, Mode: mode, Desired: desired, Running: running, Update: update})
		serviceNames[svc.ID] = svc.Spec.Name
		if stack != "" {
			st := stackMap[stack]
			if st == nil {
				st = &Stack{Name: stack}
				stackMap[stack] = st
			}
			st.Services++
			st.Desired += int(desired)
			st.Running += int(running)
		}
	}
	for _, task := range tasks {
		errText := task.Status.Err
		out.Tasks = append(out.Tasks, Task{ID: task.ID, ServiceID: task.ServiceID, Service: serviceNames[task.ServiceID], NodeID: task.NodeID, Node: nodeNames[task.NodeID], Desired: string(task.DesiredState), State: string(task.Status.State), Error: errText, Slot: task.Slot})
		if task.Status.State == swarm.TaskStateFailed || task.Status.State == swarm.TaskStateRejected {
			for _, st := range stackMap {
				if strings.HasPrefix(serviceNames[task.ServiceID], st.Name+"_") {
					st.Failed++
				}
			}
		}
	}
	for _, st := range stackMap {
		out.Stacks = append(out.Stacks, *st)
	}
	sort.Slice(out.Stacks, func(i, j int) bool { return out.Stacks[i].Name < out.Stacks[j].Name })
}

func (s *SDK) recentEvents(ctx context.Context, limit int) []Event {
	until := time.Now()
	since := until.Add(-10 * time.Minute)
	ch, errs := s.cli.Events(ctx, events.ListOptions{Since: since.Format(time.RFC3339Nano), Until: until.Format(time.RFC3339Nano)})
	out := make([]Event, 0, limit)
	for ch != nil || errs != nil {
		select {
		case ev, ok := <-ch:
			if !ok {
				ch = nil
				continue
			}
			name := ev.Actor.Attributes["name"]
			out = append(out, Event{Time: time.Unix(ev.Time, ev.TimeNano%1e9), Type: string(ev.Type), Action: string(ev.Action), ID: ev.Actor.ID, Name: name})
			if len(out) > limit {
				out = out[len(out)-limit:]
			}
		case _, ok := <-errs:
			if !ok {
				errs = nil
			}
		case <-ctx.Done():
			return out
		}
	}
	return out
}

func (s *SDK) ScaleService(ctx context.Context, id string, replicas uint64) error {
	svc, _, err := s.cli.ServiceInspectWithRaw(ctx, id, swarm.ServiceInspectOptions{})
	if err != nil {
		return err
	}
	if svc.Spec.Mode.Replicated == nil {
		return errors.New("apenas serviços replicated podem ser escalados")
	}
	svc.Spec.Mode.Replicated.Replicas = &replicas
	_, err = s.cli.ServiceUpdate(ctx, id, svc.Version, svc.Spec, swarm.ServiceUpdateOptions{})
	return err
}
func (s *SDK) SetNodeAvailability(ctx context.Context, id, value string) error {
	n, _, err := s.cli.NodeInspectWithRaw(ctx, id)
	if err != nil {
		return err
	}
	switch value {
	case "active":
		n.Spec.Availability = swarm.NodeAvailabilityActive
	case "pause":
		n.Spec.Availability = swarm.NodeAvailabilityPause
	case "drain":
		n.Spec.Availability = swarm.NodeAvailabilityDrain
	default:
		return errors.New("availability inválida")
	}
	return s.cli.NodeUpdate(ctx, id, n.Version, n.Spec)
}
func (s *SDK) ClusterInspect(ctx context.Context, kind, id string) (string, error) {
	var v any
	var err error
	switch kind {
	case "service":
		v, _, err = s.cli.ServiceInspectWithRaw(ctx, id, swarm.ServiceInspectOptions{})
	case "task":
		v, _, err = s.cli.TaskInspectWithRaw(ctx, id)
	case "node":
		v, _, err = s.cli.NodeInspectWithRaw(ctx, id)
	default:
		return "", fmt.Errorf("tipo de recurso desconhecido: %s", kind)
	}
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	return string(b), err
}
