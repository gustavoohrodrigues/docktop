package docker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	containertypes "github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docktop/docktop/internal/config"
	"github.com/docktop/docktop/internal/security"
)

type Container struct{ ID, Name, Image, State, Status string }
type ContainerMetric struct {
	CPU, Memory float64
	MemoryBytes uint64
}
type CreateRequest struct {
	Name, Image, Command, Restart string
	Ports, Volumes, Env           []string
}
type Image struct {
	ID, Tags string
	Size     int64
}
type Volume struct{ Name, Driver, Scope string }
type Network struct{ ID, Name, Driver, Scope string }
type Service struct {
	ID, Name, Stack, Image, Mode, Update string
	Desired, Running                     uint64
}
type Task struct {
	ID, ServiceID, Service, NodeID, Node, Desired, State, Error string
	Slot                                                        int
}
type Node struct {
	ID, Hostname, Role, Availability, State, Manager, Engine string
	CPUs, Memory                                             int64
}
type Stack struct {
	Name                               string
	Services, Desired, Running, Failed int
}
type Event struct {
	Time                   time.Time
	Type, Action, ID, Name string
}
type Snapshot struct {
	Info       system.Info
	Version    string
	Containers []Container
	Images     []Image
	Volumes    []Volume
	Networks   []Network
	At         time.Time
	Metrics    map[string]ContainerMetric
	Services   []Service
	Tasks      []Task
	Nodes      []Node
	Stacks     []Stack
	Events     []Event
}
type Engine interface {
	Snapshot(context.Context) (Snapshot, error)
	Start(context.Context, string) error
	Stop(context.Context, string) error
	Restart(context.Context, string) error
	UpdateImage(context.Context, string, func(string)) (string, error)
	ScaleService(context.Context, string, uint64) error
	SetNodeAvailability(context.Context, string, string) error
	ClusterInspect(context.Context, string, string) (string, error)
	Pause(context.Context, string) error
	Unpause(context.Context, string) error
	Remove(context.Context, string, bool) error
	Logs(context.Context, string, int) (string, error)
	Inspect(context.Context, string) (string, error)
	SecurityAudit(context.Context, string) (security.ContainerSecurityReport, error)
	PrepareHardening(context.Context, string, []security.HardeningControlID) (security.HardeningPlan, error)
	ApplyHardening(context.Context, string, []security.HardeningControlID) (security.HardeningResult, error)
	Processes(context.Context, string) (string, error)
	Pull(context.Context, string, func(string)) error
	CreateContainer(context.Context, CreateRequest) (string, error)
	RemoveImage(context.Context, string, bool) error
	CreateVolume(context.Context, string, string) error
	RemoveVolume(context.Context, string, bool) error
	CreateNetwork(context.Context, string, string) error
	RemoveNetwork(context.Context, string) error
	ShellCommand(context.Context, string) (*exec.Cmd, error)
	Close() error
	Endpoint() string
}
type SDK struct {
	cli      *client.Client
	endpoint string
	tls      config.TLS
}

func New(c config.Context) (*SDK, error) {
	if c.Host == "" {
		return nil, errors.New("endpoint Docker vazio")
	}
	opts := []client.Opt{client.WithHost(c.Host), client.WithAPIVersionNegotiation()}
	if c.TLS.Enabled {
		ca, err := os.ReadFile(c.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("ler CA TLS: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(ca) {
			return nil, errors.New("CA TLS inválida")
		}
		cert, err := tls.LoadX509KeyPair(c.TLS.CertFile, c.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("certificado TLS inválido: %w", err)
		}
		h := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool, Certificates: []tls.Certificate{cert}}}}
		opts = append(opts, client.WithHTTPClient(h), client.WithTLSClientConfig(c.TLS.CAFile, c.TLS.CertFile, c.TLS.KeyFile))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}
	return &SDK{cli: cli, endpoint: c.Host, tls: c.TLS}, nil
}
func (s *SDK) Endpoint() string { return s.endpoint }
func (s *SDK) Close() error     { return s.cli.Close() }
func (s *SDK) Snapshot(ctx context.Context) (Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	info, err := s.cli.Info(ctx)
	if err != nil {
		return Snapshot{}, friendly(err, s.endpoint)
	}
	v, err := s.cli.ServerVersion(ctx)
	if err != nil {
		return Snapshot{}, friendly(err, s.endpoint)
	}
	cs, err := s.cli.ContainerList(ctx, containertypes.ListOptions{All: true})
	if err != nil {
		return Snapshot{}, err
	}
	is, err := s.cli.ImageList(ctx, imagetypes.ListOptions{All: true})
	if err != nil {
		return Snapshot{}, err
	}
	vs, err := s.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return Snapshot{}, err
	}
	ns, err := s.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return Snapshot{}, err
	}
	o := Snapshot{Info: info, Version: v.Version, At: time.Now(), Metrics: map[string]ContainerMetric{}}
	for _, x := range cs {
		name := strings.TrimPrefix(first(x.Names), "/")
		o.Containers = append(o.Containers, Container{x.ID, name, x.Image, x.State, x.Status})
	}
	for _, x := range is {
		o.Images = append(o.Images, Image{x.ID, strings.Join(x.RepoTags, ", "), x.Size})
	}
	for _, x := range vs.Volumes {
		o.Volumes = append(o.Volumes, Volume{x.Name, x.Driver, x.Scope})
	}
	for _, x := range ns {
		o.Networks = append(o.Networks, Network{x.ID, x.Name, x.Driver, x.Scope})
	}
	o.Metrics = s.metrics(ctx, cs)
	if info.Swarm.ControlAvailable {
		s.collectSwarm(ctx, &o)
	}
	o.Events = s.recentEvents(ctx, 100)
	return o, nil
}
func first(v []string) string {
	if len(v) > 0 {
		return v[0]
	}
	return "-"
}
func friendly(err error, host string) error {
	m := strings.ToLower(err.Error())
	switch {
	case strings.Contains(m, "permission denied"):
		return fmt.Errorf("usuário não possui acesso ao endpoint %s", host)
	case strings.Contains(m, "no such file"):
		return errors.New("Docker daemon não está ativo ou o socket não existe")
	case strings.Contains(m, "certificate"):
		return errors.New("falha na validação TLS do endpoint remoto")
	default:
		return fmt.Errorf("Docker indisponível: %w", err)
	}
}
