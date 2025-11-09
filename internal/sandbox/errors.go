package sandbox

import "errors"

// ErrUnsupported indicates that a sandbox runner is unavailable on this system
// or cannot be initialized due to missing permissions.
var ErrUnsupported = errors.New("sandbox unsupported")
