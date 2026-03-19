package fly

import "testing"

func TestTransportSetDefaults_DoesNotOverrideFlyForceRegionFromTransport(t *testing.T) {
	t.Setenv("FLY_FORCE_REGION", "ord")

	transport := &Transport{FlyForceRegion: "iad"}
	opts := ClientOptions{Transport: transport}

	transport.setDefaults(&opts)

	if transport.FlyForceRegion != "iad" {
		t.Fatalf("expected FlyForceRegion to remain %q, got %q", "iad", transport.FlyForceRegion)
	}
}
