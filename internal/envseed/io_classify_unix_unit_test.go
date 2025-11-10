//go:build unix || darwin

package envseed

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

// [EVT-MIU-1]
func TestClassifyStatDetail_UnixMapping(t *testing.T) {
	t.Parallel()

	mkPathErr := func(errno syscall.Errno) error {
		return &os.PathError{Op: "lstat", Path: "/tmp/x", Err: errno}
	}

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "ENOENT", err: os.ErrNotExist, want: "EVE-102-1"},
		{name: "ENOTDIR", err: mkPathErr(syscall.ENOTDIR), want: "EVE-102-3"},
		{name: "ELOOP", err: mkPathErr(syscall.ELOOP), want: "EVE-102-4"},
		{name: "ENAMETOOLONG", err: mkPathErr(syscall.ENAMETOOLONG), want: "EVE-102-5"},
		{name: "OTHER", err: errors.New("weird"), want: "EVE-102-203"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := classifyStatDetail(tc.err)
			if got != tc.want {
				t.Fatalf("classifyStatDetail(%v)=%s want %s", tc.err, got, tc.want)
			}
		})
	}
}
