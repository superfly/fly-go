package flaps

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

// flapsErrorWithBody builds the FlapsError the client produces for a given
// response body, which is what every classifier here reads.
func flapsErrorWithBody(status int, body string) *FlapsError {
	return &FlapsError{
		OriginalError:      handleAPIError(status, []byte(body)),
		ResponseStatusCode: status,
		ResponseBody:       []byte(body),
	}
}

func TestIsNameTakenError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "status code set by the API",
			err:  flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"Validation failed: Name has already been taken","status":"name_taken"}`),
			want: true,
		},
		{
			// An API that predates the status code, or a create that never
			// went through the Machines API at all.
			name: "message only",
			err:  flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"Validation failed: Name has already been taken"}`),
			want: true,
		},
		{
			name: "bare error carrying the legacy message",
			err:  errors.New("Validation failed: Name has already been taken"),
			want: true,
		},
		{
			name: "wrapped",
			err:  fmt.Errorf("provisioning tigris: %w", flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"nope","status":"name_taken"}`)),
			want: true,
		},
		{
			// The generic message a collision gets when it races past the
			// validation and trips the database constraint instead. It covers
			// every unique constraint the platform has, so a client must not
			// read it as an app-name collision on its own.
			name: "generic uniqueness violation is not enough",
			err:  flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"uniqueness constraint violated"}`),
			want: false,
		},
		{
			name: "a different validation failure",
			err:  flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"Validation failed: Name under 63 chars using numbers, lowercase letters and dashes"}`),
			want: false,
		},
		{
			name: "another status code",
			err:  flapsErrorWithBody(http.StatusServiceUnavailable, `{"error":"out of capacity","status":"insufficient_capacity"}`),
			want: false,
		},
		{
			name: "nil",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNameTakenError(tc.err); got != tc.want {
				t.Errorf("IsNameTakenError() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFlapsErrorSuggestion(t *testing.T) {
	err := flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"Validation failed: Name has already been taken","status":"name_taken"}`)

	// flyctl prints this through flyerr.GetErrorSuggestion, so an empty string
	// here is the difference between the user being told why the name is
	// unavailable and being left to guess.
	if got := err.Suggestion(); got == "" {
		t.Fatal("Suggestion() is empty for a name_taken error")
	}

	// The message alone must not produce one: the suggestion is keyed off the
	// status code, and claiming otherwise would hide that the API never set it.
	noCode := flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"Validation failed: Name has already been taken"}`)
	if got := noCode.Suggestion(); got != "" {
		t.Errorf("Suggestion() = %q for an error with no status code, want empty", got)
	}
}
