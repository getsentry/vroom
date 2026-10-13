package frame

import "testing"

func TestRegressionFingerprint(t *testing.T) {
	tests := []struct {
		name    string
		frame   Frame
		legacy  uint32
		vroomrs uint32
	}{
		{
			name:    "module",
			frame:   Frame{Module: "demo", Function: "work"},
			legacy:  25692501,
			vroomrs: 152761553,
		},
		{
			name:    "module takes precedence",
			frame:   Frame{Module: "demo", Package: "/tmp/other.so", Function: "work"},
			legacy:  25692501,
			vroomrs: 152761553,
		},
		{
			name:    "unix package",
			frame:   Frame{Package: "/tmp/demo.so", Function: "work"},
			legacy:  25692501,
			vroomrs: 152761553,
		},
		{
			name:    "windows package",
			frame:   Frame{Package: `C:\app\demo.dll`, Function: "work"},
			legacy:  25692501,
			vroomrs: 152761553,
		},
		{
			name:    "no module or package",
			frame:   Frame{File: "demo.py", Function: "work", InstructionAddr: "0x1234"},
			legacy:  2431592238,
			vroomrs: 1176630088,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.frame.Fingerprint(); got != tt.legacy {
				t.Fatalf("Fingerprint() = %d, want %d", got, tt.legacy)
			}
			if got := tt.frame.VroomrsFingerprint(); got != tt.vroomrs {
				t.Fatalf("VroomrsFingerprint() = %d, want %d", got, tt.vroomrs)
			}
		})
	}
}
