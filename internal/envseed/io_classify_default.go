//go:build !unix && !darwin

package envseed

import "os"

// classifyStatDetail provides a conservative fallback mapping on non-Unix builds.
func classifyStatDetail(err error) string {
	if os.IsNotExist(err) {
		return "EVE-102-1"
	}
	return "EVE-102-203"
}
