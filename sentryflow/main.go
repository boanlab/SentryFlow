// SPDX-License-Identifier: Apache-2.0

package main

import (
	core "github.com/5GSEC/sentryflow/core"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
)

// main is the entrypoint of this program
func main() {
	core.SentryFlow()
}
