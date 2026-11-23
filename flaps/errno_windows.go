package flaps

import "golang.org/x/sys/windows"

const (
	errConnectionRefused = windows.WSAECONNREFUSED
	errConnectionReset   = windows.WSAECONNRESET
)
