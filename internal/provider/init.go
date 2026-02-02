package provider

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/config"
	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/fetcher"
	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/resolver"
	pb "github.com/autonomous-bits/nomos-provider-environment-variables/proto/providerv1"
)

// Init initializes the provider with configuration
func (p *Provider) Init(_ context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("initializing provider with alias: %s", req.Alias)
	p.setState(StateInitializing)

	// Parse configuration
	cfg, err := config.ParseConfig(req.Config)
	if err != nil {
		p.setState(StateUninitialized)
		p.logger.Error("config parse failed: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "config parse failed: %v", err)
	}

	// Validate configuration
	if err := config.ValidateConfig(cfg); err != nil {
		p.setState(StateUninitialized)
		p.logger.Error("config validation failed: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "config validation failed: %v", err)
	}

	// Validate required variables exist
	if len(cfg.RequiredVariables) > 0 {
		var missing []string
		for _, varName := range cfg.RequiredVariables {
			if _, exists := os.LookupEnv(varName); !exists {
				missing = append(missing, varName)
			}
		}
		if len(missing) > 0 {
			p.setState(StateUninitialized)
			errMsg := fmt.Sprintf("required environment variables missing: %v", missing)
			p.logger.Error("%s", errMsg)
			return nil, status.Error(codes.InvalidArgument, errMsg)
		}
	}

	// Store configuration and alias
	p.config = cfg
	p.alias = req.Alias

	// Create fetcher if not exists
	if p.fetcher == nil {
		p.fetcher = fetcher.New()
	}

	// Create resolver with configured separator, case transformation, prefix, and prefix mode
	p.resolver = resolver.NewResolver(cfg.Separator, cfg.CaseTransform, cfg.Prefix, cfg.PrefixMode)

	p.setState(StateReady)
	p.logger.Info("provider initialized successfully")

	return &pb.InitResponse{}, nil
}
