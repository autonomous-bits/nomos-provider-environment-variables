// Package main implements the environment variables provider gRPC server.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/logger"
	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/provider"
	pb "github.com/autonomous-bits/nomos-provider-environment-variables/proto/providerv1"
)

var version = "dev"

func main() {
	// Create logger (writes to stderr)
	log := logger.New(logger.INFO)

	// Create provider instance
	prov := provider.New(log)

	// Set version from build
	provider.Version = version

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024), // 10MB max message size
		grpc.MaxSendMsgSize(10*1024*1024),
	)

	// Register provider service
	pb.RegisterProviderServiceServer(grpcServer, prov)

	// Listen on random port (loopback only)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Error("failed to listen: %v", err)
		os.Exit(1)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	// Print PORT announcement to stdout (required by CLI)
	fmt.Printf("PROVIDER_PORT=%d\n", port)
	if err := os.Stdout.Sync(); err != nil {
		log.Error("failed to flush stdout: %v", err)
	}

	// Log startup to stderr
	log.Info("environment-variables provider starting")
	log.Info("version: %s", version)
	log.Info("listening on: 127.0.0.1:%d", port)

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// Start gRPC server in background
	errCh := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigCh:
		log.Info("received signal: %v", sig)
	case err := <-errCh:
		log.Error("server error: %v", err)
		os.Exit(1)
	}

	// Graceful shutdown
	log.Info("shutting down gracefully")

	// Call provider shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := prov.Shutdown(ctx, &pb.ShutdownRequest{}); err != nil {
		log.Error("error during shutdown: %v", err)
	}

	// Stop gRPC server
	grpcServer.GracefulStop()
	log.Info("shutdown complete")
}
