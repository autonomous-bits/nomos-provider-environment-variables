package provider

import (
	"context"

	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
)

// Shutdown gracefully shuts down the provider
func (p *Provider) Shutdown(_ context.Context, _ *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("shutting down provider")
	p.setState(StateShuttingDown)

	// Clear cache
	if p.fetcher != nil {
		p.fetcher.Clear()
	}

	p.setState(StateStopped)
	p.logger.Info("provider shut down successfully")

	return &pb.ShutdownResponse{}, nil
}
