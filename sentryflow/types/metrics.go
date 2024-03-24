// SPDX-License-Identifier: Apache-2.0

package types

// PerAPICount Structure
type PerAPICount struct {
	Api   string
	Count int
}

type DbAccessLogType struct {
	Labels      []byte
	Annotations []byte
	AccessLog   []byte
}
