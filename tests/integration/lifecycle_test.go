//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/autonomous-bits/nomos-provider-environment-variables/proto/providerv1"
)

// T020: Integration test for Health RPC state transitions
func TestHealthTransitions(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Before Init: should be DEGRADED
	resp, err := client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	if resp.Status != pb.HealthStatus_STATUS_DEGRADED {
		t.Errorf("expected STATUS_DEGRADED before init, got %v", resp.Status)
	}

	// After Init: should be OK
	configStruct, _ := structpb.NewStruct(map[string]interface{}{})
	_, err = client.Init(ctx, &pb.InitRequest{
		Alias:  "test-env",
		Config: configStruct,
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	resp, err = client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	if resp.Status != pb.HealthStatus_STATUS_OK {
		t.Errorf("expected STATUS_OK after init, got %v", resp.Status)
	}

	// After Shutdown: should be DEGRADED
	_, err = client.Shutdown(ctx, &pb.ShutdownRequest{})
	if err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	resp, err = client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	if resp.Status != pb.HealthStatus_STATUS_DEGRADED {
		t.Errorf("expected STATUS_DEGRADED after shutdown, got %v", resp.Status)
	}
}
