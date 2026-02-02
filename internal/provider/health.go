package provider

import (
	"context"

	pb "github.com/autonomous-bits/nomos-provider-environment-variables/proto/providerv1"
)

// Health returns the health status of the provider
func (p *Provider) Health(_ context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	state := p.GetState()

	var status pb.HealthStatus
	var message string

	switch state {
	case StateReady:
		status = pb.HealthStatus_STATUS_OK
		message = "provider is ready"
	case StateInitializing:
		status = pb.HealthStatus_STATUS_STARTING
		message = "provider is initializing"
	default:
		status = pb.HealthStatus_STATUS_DEGRADED
		message = "provider is not ready"
	}

	return &pb.HealthResponse{
		Status:  status,
		Message: message,
	}, nil
}
