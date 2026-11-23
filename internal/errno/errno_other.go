//go:build !windows

package errno

import "syscall"

const (
	ErrConnectionRefused = syscall.ECONNREFUSED
	ErrConnectionReset   = syscall.ECONNRESET
)
