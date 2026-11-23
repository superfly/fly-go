package flaps

import (
	"context"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
)

func TestWinsockReadReset(t *testing.T) {
	// A Windows socket read returns Winsock error 10054, wrapped by net and os.
	reset := &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("wsarecv", syscall.Errno(10054))}
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			tr := &scriptedTripper{steps: []step{{err: reset}, {body: `{}`}}}
			c := newTestFlapsClient(t, tr)
			err := c._sendRequest(context.Background(), method, "/apps", nil, nil, nil)
			if method == http.MethodGet {
				if err != nil || tr.callCount() != 2 {
					t.Fatalf("GET: %d attempts, %v", tr.callCount(), err)
				}
			} else if err == nil || tr.callCount() != 1 {
				t.Fatalf("%s: %d attempts, %v", method, tr.callCount(), err)
			}
		})
	}
}
