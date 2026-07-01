// Command clientsignals prints the client signals that would be attached to
// outbound Fly API requests from the current process/environment.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/superfly/fly-go/pkg/clientsignals"
)

func main() {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(clientsignals.Detect()); err != nil {
		fmt.Fprintln(os.Stderr, "error encoding signals:", err)
		os.Exit(1)
	}
}
