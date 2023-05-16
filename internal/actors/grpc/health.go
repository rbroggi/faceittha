package grpc

import (
	"context"

	pb "github.com/rbroggi/faceittha/pkg/sdk/v1"
)

// HealthService implements the User service gRPC methods.
type HealthService struct {
	pb.UnimplementedHealthServiceServer
}

// Healthz is the health endpoint for the server
func (*HealthService) Healthz(context.Context, *pb.HealthzRequest) (*pb.HealthzResponse, error) {
	return &pb.HealthzResponse{Status: "Ok"}, nil
}