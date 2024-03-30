// SPDX-License-Identifier: Apache-2.0

package types

import (
	"github.com/5GSEC/SentryFlow/protobuf"
)

// PerAPICount Structure
type PerAPICount struct {
	Api   string
	Count int
}

type DbAccessLogType struct {
	Namespace string
	Labels    string
	AccessLog *protobuf.APILog
}
