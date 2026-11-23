package errno

import "golang.org/x/sys/windows"

const (
	ErrConnectionRefused = windows.WSAECONNREFUSED
	ErrConnectionReset   = windows.WSAECONNRESET
)
