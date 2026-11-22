//go:build !windows

package flaps

import "syscall"

const connectionRefused = syscall.ECONNREFUSED
