package fly

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestIsReleaseCommandMachine(t *testing.T) {
	type testcase struct {
		name     string
		machine  Machine
		expected bool
	}

	cases := []testcase{
		{
			name:     "release machine using 'process_group'",
			expected: true,
			machine: Machine{
				Config: &MachineConfig{
					Metadata: map[string]string{
						"process_group": "release_command",
					},
				},
			},
		},
		{
			name:     "release machine using 'fly_process_group'",
			expected: true,
			machine: Machine{
				Config: &MachineConfig{
					Metadata: map[string]string{
						"fly_process_group": "fly_app_release_command",
					},
				},
			},
		},
		{
			name:     "non-release machine using 'fly_process_group'",
			expected: false,
			machine: Machine{
				Config: &MachineConfig{
					Metadata: map[string]string{
						"fly_process_group": "web",
					},
				},
			},
		},
		{
			name:     "non-release machine using 'process_group'",
			expected: false,
			machine: Machine{
				Config: &MachineConfig{
					Metadata: map[string]string{
						"process_group": "web",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		result := tc.machine.IsReleaseCommandMachine()
		if result != tc.expected {
			t.Errorf("%s, got '%v', want '%v'", tc.name, result, tc.expected)
		}
	}
}

func TestGetProcessGroup(t *testing.T) {
	type testcase struct {
		name     string
		machine  *Machine
		expected string
	}

	cases := []testcase{
		{
			name:     "machine with only 'process_group'",
			expected: "web",
			machine: &Machine{
				Config: &MachineConfig{
					Metadata: map[string]string{
						"process_group": "web",
					},
				},
			},
		},
		{
			name:     "machine with both 'process_group' & 'fly_process_group'",
			expected: "app",
			machine: &Machine{
				Config: &MachineConfig{
					Metadata: map[string]string{
						"process_group":     "web",
						"fly_process_group": "app",
					},
				},
			},
		},
		{
			name:     "machine with only 'fly_process_group'",
			expected: "web",
			machine: &Machine{
				Config: &MachineConfig{
					Metadata: map[string]string{
						"fly_process_group": "web",
					},
				},
			},
		},
		{
			name:     "machine with incomplete config and 'fly_process_group'",
			expected: "web",
			machine: &Machine{
				IncompleteConfig: &MachineConfig{
					Metadata: map[string]string{
						"fly_process_group": "web",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		result := tc.machine.ProcessGroup()
		if result != tc.expected {
			t.Errorf("%s, got '%v', want '%v'", tc.name, result, tc.expected)
		}
	}
}

func TestMachineGuest_SetSize(t *testing.T) {
	var guest MachineGuest

	if err := guest.SetSize("unknown"); err == nil {
		t.Error("want error for invalid kind")
	}

	if err := guest.SetSize("shared-cpu-3x"); err == nil {
		t.Error("want error for invalid preset name")
	}

	// Set GPU related fields that must be unset for non-gpu-size-alias
	if err := guest.SetSize("a100-40gb"); err != nil {
		t.Errorf("got error for valid preset name: %v", err)
	} else {
		if guest.GPUs != 1 {
			t.Errorf("Expected 1 gpu, got: %v", guest.GPUs)
		}
		if guest.GPUKind != "a100-pcie-40gb" {
			t.Errorf("Expected a100-pcie-40gb gpu kind, got: %v", guest.GPUKind)
		}
	}

	if err := guest.SetSize("performance-4x"); err != nil {
		t.Errorf("got error for valid preset name: %v", err)
	} else {
		if guest.CPUs != 4 {
			t.Errorf("Expected 4 cpus, got: %v", guest.CPUs)
		}
		if guest.CPUKind != "performance" {
			t.Errorf("Expected performance cpu kind, got: %v", guest.CPUKind)
		}
		if guest.MemoryMB != 8192 {
			t.Errorf("Expected 8192 MB of memory , got: %v", guest.MemoryMB)
		}
		if guest.GPUs != 0 {
			t.Errorf("Expected 0 gpus, got: %v", guest.GPUs)
		}
		if guest.GPUKind != "" {
			t.Errorf("Expected non gpu kind, got: %v", guest.GPUKind)
		}
	}
}

func TestMachineGuest_ToSize(t *testing.T) {
	for want, guest := range MachinePresets {
		got := guest.ToSize()
		if want != got {
			t.Errorf("want '%s', got '%s'", want, got)
		}
	}

	got := (&MachineGuest{}).ToSize()
	if got != "unknown" {
		t.Errorf("want 'unknown', got '%s'", got)
	}
}

func TestMachineMostRecentStartTimeAfterLaunch(t *testing.T) {
	type testcase struct {
		name        string
		machine     *Machine
		expected    time.Time
		expectedErr bool
	}
	var (
		time01 = time.Now()
		time05 = time01.Add(5 * time.Second)
		time17 = time01.Add(17 * time.Second)
		time99 = time01.Add(99 * time.Second)
	)
	cases := []testcase{
		{name: "nil machine", machine: nil, expectedErr: true},
		{name: "no events", machine: &Machine{}, expectedErr: true},
		{name: "launch only event", expectedErr: true,
			machine: &Machine{Events: []*MachineEvent{
				{Type: "launch", Timestamp: time01.UnixMilli()},
			}},
		},
		{name: "start only event", expectedErr: true,
			machine: &Machine{Events: []*MachineEvent{
				{Type: "start", Timestamp: time01.UnixMilli()},
			}},
		},
		{name: "launch after start", expectedErr: true,
			machine: &Machine{Events: []*MachineEvent{
				{Type: "launch", Timestamp: time05.UnixMilli()},
				{Type: "start", Timestamp: time01.UnixMilli()},
			}},
		},
		{name: "exit after start", expectedErr: true,
			machine: &Machine{Events: []*MachineEvent{
				{Type: "exit", Timestamp: time05.UnixMilli()},
				{Type: "start", Timestamp: time01.UnixMilli()},
			}},
		},
		{name: "launch, start", expected: time17,
			machine: &Machine{Events: []*MachineEvent{
				{Type: "start", Timestamp: time17.UnixMilli()},
				{Type: "launch", Timestamp: time05.UnixMilli()},
			}},
		},
		{name: "exit, launch, start", expected: time17,
			machine: &Machine{Events: []*MachineEvent{
				{Type: "start", Timestamp: time17.UnixMilli()},
				{Type: "launch", Timestamp: time05.UnixMilli()},
				{Type: "exit", Timestamp: time01.UnixMilli()},
			}},
		},
		{name: "exit, launch, start, exit", expectedErr: true,
			machine: &Machine{Events: []*MachineEvent{
				{Type: "exit", Timestamp: time99.UnixMilli()},
				{Type: "start", Timestamp: time17.UnixMilli()},
				{Type: "launch", Timestamp: time05.UnixMilli()},
				{Type: "exit", Timestamp: time01.UnixMilli()},
			}},
		},
	}
	for _, testCase := range cases {
		actual, err := testCase.machine.MostRecentStartTimeAfterLaunch()
		if testCase.expectedErr {
			if err == nil {
				t.Error(testCase.name, "expected error, got nil")
			}
		} else {
			if err != nil {
				t.Error(testCase.name, "unexpected error:", err)
			} else {
				delta := testCase.expected.Sub(actual)
				if delta < -1*time.Millisecond || delta > 1*time.Millisecond {
					t.Error(testCase.name, "expected", testCase.expected, "got", actual)
				}
			}
		}
	}
}

func TestMachineAutostopUnmarshalJSON(t *testing.T) {
	type testcase struct {
		input  string
		output MachineAutostop
	}
	cases := []testcase{
		{`false`, MachineAutostopOff},
		{`true`, MachineAutostopStop},
		{`"off"`, MachineAutostopOff},
		{`"stop"`, MachineAutostopStop},
		{`"suspend"`, MachineAutostopSuspend},
	}
	for _, testCase := range cases {
		var s MachineAutostop
		if err := json.Unmarshal([]byte(testCase.input), &s); err != nil {
			t.Errorf("input %s: unexpected error: %v", testCase.input, err)
		} else if s != testCase.output {
			t.Errorf("input %s: expected %v, got %v", testCase.input, testCase.output, s)
		}
	}
}

func TestMachineAutostopMarshalJSON(t *testing.T) {
	type testcase struct {
		input  MachineAutostop
		output string
	}
	cases := []testcase{
		{MachineAutostopOff, `false`}, // it's important for backward-compatibility
		{MachineAutostopStop, `true`}, // that these are serialized as booleans!
		{MachineAutostopSuspend, `"suspend"`},
	}
	for _, testCase := range cases {
		b, err := json.Marshal(testCase.input)
		if err != nil {
			t.Errorf("input %v: unexpected error: %v", testCase.input, err)
		} else if !bytes.Equal(b, []byte(testCase.output)) {
			t.Errorf("input %v: expected %v, got %s", testCase.input, testCase.output, string(b))
		}
	}
}

func TestIsAppV2(t *testing.T) {
	type testcase struct {
		name     string
		machine  *Machine
		expected bool
	}

	cases := []testcase{
		{
			name:     "machine with 'fly_platform_version=v2'",
			expected: true,
			machine: &Machine{
				Config: &MachineConfig{
					Metadata: map[string]string{"fly_platform_version": "v2"},
				},
			},
		},
		{
			name:     "machine with non v2 'fly_platform_version'",
			expected: false,
			machine: &Machine{
				Config: &MachineConfig{
					Metadata: map[string]string{"fly_platform_version": "v1"},
				},
			},
		},
		{
			name:     "machine without config",
			expected: false,
			machine:  &Machine{},
		},
		{
			name:     "machine with 'fly_platform_version=v2' in incomplete config",
			expected: true,
			machine: &Machine{
				IncompleteConfig: &MachineConfig{
					Metadata: map[string]string{"fly_platform_version": "v2"},
				},
			},
		},
	}

	for _, tc := range cases {
		result := tc.machine.IsAppsV2()
		if result != tc.expected {
			t.Errorf("%s, got '%v', want '%v'", tc.name, result, tc.expected)
		}
	}
}
