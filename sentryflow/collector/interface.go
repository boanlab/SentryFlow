package collector

import "google.golang.org/grpc"

// collectorInterface Interface
type collectorInterface interface {
	RegisterService(server *grpc.Server)
}
