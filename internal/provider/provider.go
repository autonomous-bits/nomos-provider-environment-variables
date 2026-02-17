package provider

import (
	"sync"
	"sync/atomic"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/config"
	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/fetcher"
	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/logger"
	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/resolver"
	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
)

// State represents the lifecycle state of the provider
type State int32

const (
	// StateUninitialized indicates the provider has not been initialized.
	StateUninitialized State = iota
	// StateInitializing indicates initialization is in progress.
	StateInitializing
	// StateReady indicates the provider is ready to serve requests.
	StateReady
	// StateShuttingDown indicates shutdown is in progress.
	StateShuttingDown
	// StateStopped indicates the provider has stopped.
	StateStopped
)

// Provider implements the ProviderService gRPC contract
type Provider struct {
	pb.UnimplementedProviderServiceServer

	alias    string
	config   *config.Config
	fetcher  *fetcher.Fetcher
	resolver *resolver.Resolver
	// cache   sync.Map // Reserved for future use
	state  atomic.Int32
	logger *logger.Logger
	mu     sync.RWMutex
}

// New creates a new Provider instance
func New(log *logger.Logger) *Provider {
	p := &Provider{
		logger: log,
	}
	p.state.Store(int32(StateUninitialized))
	return p
}

// GetState returns the current provider state
func (p *Provider) GetState() State {
	return State(p.state.Load())
}

// setState atomically updates the provider state
func (p *Provider) setState(state State) {
	p.state.Store(int32(state))
	p.logger.Info("state transition: %v", state)
}

// Version is injected at build time
var Version = "dev"
