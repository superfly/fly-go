//go:build !windows

package errno

import "syscall"

const (
	ConnectionRefused = syscall.ECONNREFUSED
	ConnectionReset   = syscall.ECONNRESET
)
