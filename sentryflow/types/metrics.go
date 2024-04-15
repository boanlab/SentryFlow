// SPDX-License-Identifier: Apache-2.0

package types

import (
	"github.com/5GSEC/SentryFlow/protobuf"
)

// PerAPICount Structure
type PerAPICount struct {
	API   string
	Count uint64
}

// DbAccessLogType Structure
type DbAccessLogType struct {
	Namespace string
	Labels    string
	AccessLog *protobuf.APILog
}
