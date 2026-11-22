//go:build !windows

package flaps

import "syscall"

const errConnectionRefused = syscall.ECONNREFUSED
