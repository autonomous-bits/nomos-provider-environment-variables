package provider

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/fetcher"
	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/resolver"
	pb "github.com/autonomous-bits/nomos-provider-environment-variables/proto/providerv1"
)

// Fetch retrieves configuration data at the specified path
func (p *Provider) Fetch(_ context.Context, req *pb.FetchRequest) (*pb.FetchResponse, error) {
	// Check if initialized
	if p.GetState() != StateReady {
		p.logger.Error("fetch called before initialization")
		return nil, status.Error(codes.FailedPrecondition, "provider not initialized")
	}

	// Validate path
	if len(req.Path) == 0 {
		p.logger.Error("fetch called with empty path")
		return nil, status.Error(codes.InvalidArgument, "path cannot be empty")
	}

	for i, segment := range req.Path {
		if strings.TrimSpace(segment) == "" {
			p.logger.Error("fetch called with empty path segment at index %d", i)
			return nil, status.Errorf(codes.InvalidArgument, "path[%d] cannot be empty string", i)
		}
	}

	// Determine the variable name to fetch
	var varName string
	var err error

	if len(req.Path) == 1 {
		// Single-segment path: direct environment variable access
		varName = req.Path[0]
		p.logger.Debug("fetching environment variable (direct): %s", varName)
	} else {
		// Multi-segment path: transform using resolver
		varName, err = p.resolver.Transform(req.Path)
		if err != nil {
			p.logger.Error("path transformation failed for %v: %v", req.Path, err)
			return nil, status.Errorf(codes.InvalidArgument, "path transformation failed: %v", err)
		}
		p.logger.Debug("fetching environment variable (transformed): %s from path %v", varName, req.Path)
	}

	// In filter_only mode, check if the variable passes the prefix filter
	// This prevents access to variables that don't have the required prefix
	if p.config.PrefixMode == "filter_only" && p.config.Prefix != "" {
		if !resolver.FilterByPrefix(varName, p.config.Prefix) {
			p.logger.Warn("environment variable does not match prefix filter: %s (prefix: %s)", varName, p.config.Prefix)
			return nil, status.Errorf(codes.NotFound, "environment variable not found: %s", varName)
		}
	}

	// Fetch from environment
	value, err := p.fetcher.Fetch(varName)
	if err != nil {
		if errors.Is(err, fetcher.ErrNotFound) {
			p.logger.Warn("environment variable not found: %s", varName)
			return nil, status.Errorf(codes.NotFound, "environment variable not found: %s", varName)
		}
		if errors.Is(err, fetcher.ErrValueTooLarge) {
			p.logger.Error("environment variable value too large: %s", varName)
			return nil, status.Errorf(codes.InvalidArgument, "environment variable value exceeds maximum size of %d bytes", fetcher.MaxValueSize)
		}
		p.logger.Error("fetch failed for %s: %v", varName, err)
		return nil, status.Errorf(codes.Internal, "fetch failed: %v", err)
	}

	// Apply type conversion if enabled
	var convertedValue interface{} = value
	if p.config.EnableTypeConversion || p.config.EnableJSONParsing {
		var converted interface{}
		converted, err = p.convertValue(value)
		if err != nil {
			p.logger.Error("type conversion failed for %s: %v", varName, err)
			return nil, status.Errorf(codes.InvalidArgument, "type conversion failed: %v", err)
		}
		convertedValue = converted
	}

	// Convert value to protobuf Value
	protoValue, err := toProtoValue(convertedValue)
	if err != nil {
		p.logger.Error("failed to convert value to protobuf: %v", err)
		return nil, status.Errorf(codes.Internal, "value conversion failed: %v", err)
	}

	// Wrap in a struct with "value" field
	valueStruct, err := structpb.NewStruct(map[string]interface{}{
		"value": protoValue,
	})
	if err != nil {
		p.logger.Error("failed to create protobuf struct: %v", err)
		return nil, status.Errorf(codes.Internal, "struct creation failed: %v", err)
	}

	p.logger.Debug("successfully fetched %s", varName)

	return &pb.FetchResponse{
		Value: valueStruct,
	}, nil
}
