package ratelimit

import (
    "context"
    "net"
    "strings"

    pb "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
    "google.golang.org/grpc"
    "k8s.io/klog/v2"
)

type GRPCServer struct {
    pb.UnimplementedRateLimitServiceServer
    manager *RateLimitManager
}

func NewGRPCServer(manager *RateLimitManager) *GRPCServer {
    return &GRPCServer{
        manager: manager,
    }
}

func (s *GRPCServer) ShouldRateLimit(ctx context.Context, req *pb.RateLimitRequest) (*pb.RateLimitResponse, error) {
    // Build key from request domain and descriptors
    key := buildKeyFromRequest(req)
    
    klog.V(4).Infof("gRPC rate limit request: domain=%s, key=%s", req.Domain, key)
    
    result, err := s.manager.Check(ctx, key)
    if err != nil {
        klog.Errorf("gRPC rate limit check failed: %v", err)
        return &pb.RateLimitResponse{
            OverallCode: pb.RateLimitResponse_UNKNOWN,
        }, nil
    }
    
    code := pb.RateLimitResponse_OK
    if !result.Allowed {
        code = pb.RateLimitResponse_OVER_LIMIT
    }
    
    return &pb.RateLimitResponse{
        OverallCode: code,
        Statuses:    []*pb.RateLimitResponse_DescriptorStatus{},
    }, nil
}

func buildKeyFromRequest(req *pb.RateLimitRequest) string {
    // Build key from domain and descriptors
    parts := []string{"domain=" + req.Domain}
    
    for _, desc := range req.Descriptors {
        for _, entry := range desc.Entries {
            parts = append(parts, entry.Key+"="+entry.Value)
        }
    }
    return strings.Join(parts, "|")
}

// StartGRPCServer starts gRPC server on specified port
func StartGRPCServer(port string, manager *RateLimitManager) (*grpc.Server, error) {
    lis, err := net.Listen("tcp", ":"+port)
    if err != nil {
        return nil, err
    }
    
    grpcServer := grpc.NewServer()
    pb.RegisterRateLimitServiceServer(grpcServer, NewGRPCServer(manager))
    
    go func() {
        if err := grpcServer.Serve(lis); err != nil {
            klog.Errorf("gRPC server error: %v", err)
        }
    }()
    
    klog.Infof("gRPC rate limit server listening on port %s", port)
    return grpcServer, nil
}