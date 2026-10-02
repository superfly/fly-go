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
			// A code the API set outranks the message. Without that, an error
			// that aggregates or quotes another one could carry the legacy
			// text and be read as a collision it is not.
			name: "another status code whose message quotes the legacy one",
			err:  flapsErrorWithBody(http.StatusServiceUnavailable, `{"error":"retry gave up after: Validation failed: Name has already been taken","status":"insufficient_capacity"}`),
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

func TestFlapsErrorSuggestionVolumePlacementCapacity(t *testing.T) {
	err := flapsErrorWithBody(http.StatusPreconditionFailed, `{"error":"insufficient resources to create new machine with existing volume id 'vol_123'","status":"volume_placement_capacity"}`)

	if got := err.Suggestion(); got == "" {
		t.Fatal("Suggestion() is empty for a volume_placement_capacity error")
	}
}

func TestCapacityScopeString(t *testing.T) {
	cases := []struct {
		scope CapacityScope
		want  string
	}{
		{CapacityScopeNone, "none"},
		{CapacityScopeHost, "host"},
		{CapacityScopeVolumePlacement, "volume_placement"},
		{CapacityScopeRegion, "region"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.scope.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCapacityScopeOf(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want CapacityScope
	}{
		{
			name: "422 insufficient_capacity is region scope",
			err:  flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"could not find a candidate host","status":"insufficient_capacity"}`),
			want: CapacityScopeRegion,
		},
		{
			name: "412 volume_placement_capacity is volume placement scope",
			err:  flapsErrorWithBody(http.StatusPreconditionFailed, `{"error":"insufficient resources to create new machine with existing volume id '123'","status":"volume_placement_capacity"}`),
			want: CapacityScopeVolumePlacement,
		},
		{
			name: "409 CPU sentinel wrapped by gate is host scope",
			err:  flapsErrorWithBody(http.StatusConflict, `{"error":"aborted: could not reserve resource for machine: insufficient CPUs available to fulfill request on the current host"}`),
			want: CapacityScopeHost,
		},
		{
			name: "409 memory sentinel wrapped by gate is host scope",
			err:  flapsErrorWithBody(http.StatusConflict, `{"error":"aborted: could not reserve resource for machine: insufficient memory available to fulfill request on the current host"}`),
			want: CapacityScopeHost,
		},
		{
			name: "409 bare IPs sentinel, no gate, is host scope",
			err:  flapsErrorWithBody(http.StatusConflict, `{"error":"aborted: insufficient IPs available to fulfill request"}`),
			want: CapacityScopeHost,
		},
		{
			name: "409 create-path resource wrapper is host scope",
			err:  flapsErrorWithBody(http.StatusConflict, `{"error":"aborted: insufficient resources available to fulfill request: insufficient memory available to fulfill request"}`),
			want: CapacityScopeHost,
		},
		{
			name: "409 no capacity is region scope",
			err:  flapsErrorWithBody(http.StatusConflict, `{"error":"aborted: no capacity"}`),
			want: CapacityScopeRegion,
		},
		{
			name: "409 concurrent update is not a capacity error",
			err:  flapsErrorWithBody(http.StatusConflict, `{"error":"aborted: machine update failed due to concurrent update"}`),
			want: CapacityScopeNone,
		},
		{
			name: "409 name collision is not a capacity error",
			err:  flapsErrorWithBody(http.StatusConflict, `{"error":"already_exists: unique machine name violation, ..."}`),
			want: CapacityScopeNone,
		},
		{
			name: "409 empty body is not a capacity error",
			err:  flapsErrorWithBody(http.StatusConflict, ``),
			want: CapacityScopeNone,
		},
		{
			name: "409 non-JSON body still matches on raw text",
			err:  flapsErrorWithBody(http.StatusConflict, `insufficient CPUs available to fulfill request`),
			want: CapacityScopeHost,
		},
		{
			name: "500 with a matching phrase is not consulted at all",
			err:  flapsErrorWithBody(http.StatusInternalServerError, `{"error":"insufficient CPUs available to fulfill request"}`),
			want: CapacityScopeNone,
		},
		{
			// A present code, even one this package doesn't recognize, wins
			// over the text: reaching for the message after the API has
			// already classified the error can only misclassify.
			name: "a present but unrecognized status code outranks a matching phrase",
			err:  flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"insufficient CPUs available to fulfill request","status":"unknown"}`),
			want: CapacityScopeNone,
		},
		{
			name: "wrapped error is still classified",
			err:  fmt.Errorf("failed to update VM 123: %w", flapsErrorWithBody(http.StatusConflict, `{"error":"aborted: could not reserve resource for machine: insufficient CPUs available to fulfill request on the current host"}`)),
			want: CapacityScopeHost,
		},
		{
			name: "non-FlapsError is not a capacity error",
			err:  errors.New("insufficient CPUs available to fulfill request"),
			want: CapacityScopeNone,
		},
		{
			name: "nil",
			err:  nil,
			want: CapacityScopeNone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CapacityScopeOf(tc.err); got != tc.want {
				t.Errorf("CapacityScopeOf() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsCapacityError(t *testing.T) {
	capacityErr := flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"could not find a candidate host","status":"insufficient_capacity"}`)
	if !IsCapacityError(capacityErr) {
		t.Error("IsCapacityError() = false, want true for a region-scoped error")
	}

	nameErr := flapsErrorWithBody(http.StatusUnprocessableEntity, `{"error":"Validation failed: Name has already been taken","status":"name_taken"}`)
	if IsCapacityError(nameErr) {
		t.Error("IsCapacityError() = true, want false for a name_taken error")
	}
}
