package clientsignals

import "testing"

func TestDetect_Composition(t *testing.T) {
	t.Setenv("FLY_INVOKED_BY", "test-harness")
	t.Setenv("CI", "true")

	s := Detect()

	if s.Agent != "test-harness" || s.AgentSource != "env:FLY_INVOKED_BY" {
		t.Fatalf("expected agent from FLY_INVOKED_BY, got agent=%q source=%q", s.Agent, s.AgentSource)
	}
	if !s.CI {
		t.Fatal("expected CI to be true")
	}
	switch s.Parent {
	case "node", "python", "shell", "other":
	default:
		t.Fatalf("Parent must be a finite value, got %q", s.Parent)
	}
}

func TestDetect_AgentAndSourceAreBothEmptyOrBothSet(t *testing.T) {
	s := Detect()
	if (s.Agent == "") != (s.AgentSource == "") {
		t.Fatalf("Agent and AgentSource must be empty together or set together, got agent=%q source=%q", s.Agent, s.AgentSource)
	}
}

func TestDetectOnce_ComputesOnce(t *testing.T) {
	resetCachedForTest()
	t.Cleanup(resetCachedForTest)

	t.Setenv("FLY_INVOKED_BY", "cached-tool")
	first := DetectOnce()

	// Changing the environment after the first call must not affect the
	// already-cached result.
	t.Setenv("FLY_INVOKED_BY", "different-tool")
	second := DetectOnce()

	if first != second {
		t.Fatalf("expected DetectOnce to return the same cached value on repeated calls, got %+v then %+v", first, second)
	}
	if second.Agent != "cached-tool" {
		t.Fatalf("expected cached value to reflect the environment at the first call, got agent=%q", second.Agent)
	}
}
