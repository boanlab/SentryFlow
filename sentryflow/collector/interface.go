// SPDX-License-Identifier: Apache-2.0

package collector

import "google.golang.org/grpc"

// collectorInterface Interface
type collectorInterface interface {
	registerService(server *grpc.Server)
}
