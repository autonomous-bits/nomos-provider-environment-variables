package provider

import (
	"context"

	pb "github.com/autonomous-bits/nomos-provider-environment-variables/proto/providerv1"
)

// Info returns provider metadata
func (p *Provider) Info(_ context.Context, _ *pb.InfoRequest) (*pb.InfoResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return &pb.InfoResponse{
		Alias:   p.alias,
		Version: Version,
		Type:    "environment-variables",
	}, nil
}
