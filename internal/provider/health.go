package provider

import (
	"context"

	pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
)

// Health returns the health status of the provider
func (p *Provider) Health(_ context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	state := p.GetState()

	var status pb.HealthResponse_Status
	var message string

	switch state {
	case StateReady:
		status = pb.HealthResponse_STATUS_OK
		message = "provider is ready"
	case StateInitializing:
		status = pb.HealthResponse_STATUS_STARTING
		message = "provider is initializing"
	default:
		status = pb.HealthResponse_STATUS_DEGRADED
		message = "provider is not ready"
	}

	return &pb.HealthResponse{
		Status:  status,
		Message: message,
	}, nil
}
