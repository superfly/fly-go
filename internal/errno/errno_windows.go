package errno

import "golang.org/x/sys/windows"

const (
	ConnectionRefused = windows.WSAECONNREFUSED
	ConnectionReset   = windows.WSAECONNRESET
)
