// SPDX-License-Identifier: Apache-2.0

package types

// k8sResources const
const (
	K8sResourceTypeUnknown = 0
	K8sResourceTypePod     = 1
	K8sResourceTypeService = 2
)

// K8sNetworkedResource Structure
type K8sNetworkedResource struct {
	Name      string
	Namespace string
	Labels    map[string]string
	Type      uint8
}

// K8sResourceTypeToString Function
func K8sResourceTypeToString(t uint8) string {
	switch t {
	case K8sResourceTypePod:
		return "Pod"
	case K8sResourceTypeService:
		return "Service"
	case K8sResourceTypeUnknown:
	default:
		return "Unknown"
	}

	return "Unknown"
}
