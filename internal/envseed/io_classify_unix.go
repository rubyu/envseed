//go:build unix || darwin

package envseed

import (
	"errors"
	"os"
	"syscall"
)

// classifyStatDetail maps a stat-phase error into an EVE-102 detail code.
// Unix/macOS implementation: classifies ENOENT/ENOTDIR/ELOOP/ENAMETOOLONG
// under EVE-102-B0. Unknown errno fall back to EVE-102-203.
func classifyStatDetail(err error) string {
	if os.IsNotExist(err) {
		return "EVE-102-1"
	}
	if errors.Is(err, syscall.ENOTDIR) {
		return "EVE-102-3"
	}
	if errors.Is(err, syscall.ELOOP) {
		return "EVE-102-4"
	}
	if errors.Is(err, syscall.ENAMETOOLONG) {
		return "EVE-102-5"
	}
	return "EVE-102-203"
}
