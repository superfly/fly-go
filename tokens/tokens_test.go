package tokens

import (
	"go/build"
	"testing"
)

// TestPackageNeverImportsFlyGo guards against this package's HTTP client
// (used to discharge third-party macaroon caveats, which may talk to
// non-Fly auth locations) ever gaining the ability to attach the
// fly-go client-signal headers/User-Agent suffix. Those signals must only
// ever be sent to Fly's own API surfaces (the GraphQL and flaps clients),
// never to arbitrary third parties.
func TestPackageNeverImportsFlyGo(t *testing.T) {
	pkg, err := build.ImportDir(".", 0)
	if err != nil {
		t.Fatalf("build.ImportDir: %v", err)
	}

	for _, imp := range pkg.Imports {
		if imp == "github.com/superfly/fly-go" {
			t.Fatal("tokens package must never import github.com/superfly/fly-go: " +
				"its discharge HTTP client can talk to third-party locations and must never receive Fly-Client-* signal headers")
		}
	}
}

func TestAuthorizationHeader(t *testing.T) {
	check := func(macaroonAndUserTokens bool, input, expectedOutput string) {
		t.Helper()
		if tok := Parse(input).normalized(macaroonAndUserTokens, false); tok != expectedOutput {
			t.Fatalf("expected token to be '%s', got '%s'", expectedOutput, tok)
		}
	}

	// scheme stripping
	check(true, "foobar", "foobar")
	check(true, "Bearer foobar", "foobar")
	check(true, "FlyV1 foobar", "foobar")
	check(true, "Bearer FlyV1 foobar", "foobar")
	check(true, "FlyV1 Bearer foobar", "foobar")
	check(true, "BEARER FLYV1 foobar", "foobar")

	// api access token
	check(true, "fm2_foobar,foobar", "fm2_foobar,foobar")
	check(true, "foobar,fm2_foobar", "fm2_foobar,foobar")
	check(true, "foobar", "foobar")
	check(true, "fm2_foobar", "fm2_foobar")

	// non-api access token
	check(false, "fm2_foobar,foobar", "fm2_foobar")
	check(false, "foobar,fm2_foobar", "fm2_foobar")
	check(false, "foobar", "foobar")
	check(false, "fm2_foobar", "fm2_foobar")
}
