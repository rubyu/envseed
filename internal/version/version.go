package version

import (
	"fmt"
	"strings"
	"time"
)

// Version is filled at build time via -ldflags. Leave empty for fallback.
var Version string

// String returns the effective version string, applying the spec-required fallback
// when no build-time value is supplied.
func String() string {
	v := strings.TrimSpace(Version)
	if v != "" {
		return v
	}
	return fmt.Sprintf("v0.0.0-dev+%s.unknown", time.Now().UTC().Format("20060102"))
}
