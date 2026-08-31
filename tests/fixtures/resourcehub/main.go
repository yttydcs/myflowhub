package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	filesystemfeature "github.com/yttydcs/myflowhub/feature/filesystem"
	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

type mountOption struct {
	name  string
	root  string
	label string
}

type options struct {
	stateDirectory string
	listen         string
	nodeID         uint64
	mounts         [3]mountOption
}

type runningFixture struct {
	hub        *hub.Hub
	filesystem *filesystemfeature.Registration
	resources  []string
	closeOnce  sync.Once
	closeErr   error
}

type startupResult struct {
	Version   int      `json:"version"`
	NodeID    string   `json:"node_id"`
	Endpoints []string `json:"endpoints"`
	Resources []string `json:"resources"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := execute(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if ctx == nil {
		return errors.New("fixture context is required")
	}
	if stdout == nil || stderr == nil {
		return errors.New("fixture output streams are required")
	}
	config, err := parseOptions(args, stderr)
	if err != nil {
		return err
	}
	fixture, err := startFixture(ctx, config)
	if err != nil {
		return err
	}
	result := startupResult{Version: 1, NodeID: strconv.FormatUint(uint64(fixture.hub.Node.ID()), 10), Resources: append([]string(nil), fixture.resources...)}
	for _, endpoint := range fixture.hub.Endpoints {
		result.Endpoints = append(result.Endpoints, string(endpoint))
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		return errors.Join(fmt.Errorf("write fixture startup result: %w", err), fixture.Close())
	}
	<-ctx.Done()
	return fixture.Close()
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var config options
	flags := flag.NewFlagSet("resourcehub", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&config.stateDirectory, "state", "", "required absolute isolated Hub state directory")
	flags.StringVar(&config.listen, "listen", "127.0.0.1:0", "loopback TCP listen endpoint")
	flags.Uint64Var(&config.nodeID, "id", 1, "non-zero Hub node ID")
	defaults := []mountOption{
		{name: "storage/allowed", label: "Allowed files"},
		{name: "storage/forbidden", label: "Forbidden files"},
		{name: "storage/misc", label: "Miscellaneous files"},
	}
	keys := []string{"allowed", "forbidden", "misc"}
	for index, key := range keys {
		config.mounts[index] = defaults[index]
		flags.StringVar(&config.mounts[index].root, key+"-root", "", "required absolute "+key+" mount root")
		flags.StringVar(&config.mounts[index].name, key+"-name", defaults[index].name, key+" Resource name")
		flags.StringVar(&config.mounts[index].label, key+"-label", defaults[index].label, key+" presentation label")
	}
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected fixture arguments: %s", strings.Join(flags.Args(), " "))
	}
	if err := config.validate(); err != nil {
		return options{}, err
	}
	return config, nil
}

func (o options) validate() error {
	if strings.TrimSpace(o.stateDirectory) == "" {
		return errors.New("fixture state directory is required")
	}
	if !filepath.IsAbs(o.stateDirectory) {
		return errors.New("fixture state directory must be absolute")
	}
	nodeID := protocol.NodeID(o.nodeID)
	if err := nodeID.Validate(); err != nil {
		return fmt.Errorf("fixture Hub identity: %w", err)
	}
	if err := validateLoopbackEndpoint(o.listen); err != nil {
		return err
	}
	names := make(map[string]struct{}, len(o.mounts))
	for index, mount := range o.mounts {
		if mount.root == "" {
			return fmt.Errorf("fixture mount %d root is required", index)
		}
		if !filepath.IsAbs(mount.root) {
			return fmt.Errorf("fixture mount %d root must be absolute", index)
		}
		if strings.TrimSpace(mount.name) != mount.name || mount.name == "" {
			return fmt.Errorf("fixture mount %d Resource name is required without surrounding whitespace", index)
		}
		if err := (protocol.ResourceID{Owner: nodeID, Name: mount.name}).Validate(); err != nil {
			return fmt.Errorf("fixture mount %d Resource name: %w", index, err)
		}
		if _, exists := names[mount.name]; exists {
			return fmt.Errorf("fixture mount %d duplicates Resource name %q", index, mount.name)
		}
		names[mount.name] = struct{}{}
		if strings.TrimSpace(mount.label) != mount.label || mount.label == "" {
			return fmt.Errorf("fixture mount %d label is required without surrounding whitespace", index)
		}
	}
	return nil
}

func validateLoopbackEndpoint(value string) error {
	host, portText, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("fixture listen endpoint must use loopback host:port: %w", err)
	}
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return errors.New("fixture listen endpoint must use a loopback address")
		}
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 || port > 65535 {
		return errors.New("fixture listen endpoint port must be between 0 and 65535")
	}
	return nil
}

func startFixture(ctx context.Context, config options) (*runningFixture, error) {
	if ctx == nil {
		return nil, errors.New("fixture context is required")
	}
	if err := config.validate(); err != nil {
		return nil, err
	}
	runtime, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: config.stateDirectory,
		NodeID:         protocol.NodeID(config.nodeID),
		Listeners: []hub.ListenerConfig{{
			Driver: tcp.Driver{}, Endpoint: link.Endpoint(config.listen),
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("start resource Hub fixture: %w", err)
	}
	mounts := make([]filesystemfeature.Mount, 0, len(config.mounts))
	resources := make([]string, 0, len(config.mounts))
	for _, mount := range config.mounts {
		mounts = append(mounts, filesystemfeature.Mount{Name: mount.name, Root: mount.root, Label: mount.label})
		resources = append(resources, mount.name)
	}
	registration, err := filesystemfeature.Register(runtime.Node, mounts)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("register filesystem fixture: %w", err), runtime.Close())
	}
	return &runningFixture{hub: runtime, filesystem: registration, resources: resources}, nil
}

func (f *runningFixture) Close() error {
	if f == nil {
		return nil
	}
	f.closeOnce.Do(func() {
		var filesystemErr, hubErr error
		if f.filesystem != nil {
			filesystemErr = f.filesystem.Close()
		}
		if f.hub != nil {
			hubErr = f.hub.Close()
		}
		f.closeErr = errors.Join(filesystemErr, hubErr)
	})
	return f.closeErr
}
