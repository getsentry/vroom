package frame

import (
	"hash/fnv"
	"testing"
)

func TestFindFrameByFingerprintWithFallback(t *testing.T) {
	tests := []struct {
		name              string
		frames            []Frame
		targetFingerprint uint32
		expectMatch       bool
		expectFallback    bool
		expectedFunction  string
	}{
		{
			name: "exact match",
			frames: []Frame{
				{Function: "testFunc", Module: "testModule"},
			},
			targetFingerprint: Frame{Function: "testFunc", Module: "testModule"}.Fingerprint(),
			expectMatch:       true,
			expectFallback:    false,
			expectedFunction:  "testFunc",
		},
		{
			name: "fallback match with raw package",
			frames: []Frame{
				{Function: "testFunc", Package: "/path/to/libtest.so"},
			},
			targetFingerprint: Frame{Function: "testFunc", Package: "/path/to/libtest.so"}.Fingerprint(),
			expectMatch:       true,
			expectFallback:    false,
			expectedFunction:  "testFunc",
		},
		{
			name: "no match",
			frames: []Frame{
				{Function: "testFunc", Module: "testModule"},
			},
			targetFingerprint: Frame{Function: "differentFunc", Module: "differentModule"}.Fingerprint(),
			expectMatch:       false,
			expectFallback:    false,
		},
		{
			name: "fallback match with file instead of module",
			frames: []Frame{
				{Function: "testFunc", File: "testFile.py", Module: "testModule"},
			},
			targetFingerprint: computeFingerprintFromFileAndFunction("testFile.py", "testFunc"),
			expectMatch:       true,
			expectFallback:    true,
			expectedFunction:  "testFunc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matchedFrame, usedFallback, err := FindFrameByFingerprintWithFallback(tt.frames, tt.targetFingerprint)

			if tt.expectMatch {
				if err != nil {
					t.Errorf("Expected match but got error: %v", err)
				}
				if matchedFrame.Function != tt.expectedFunction {
					t.Errorf("Expected function %s, got %s", tt.expectedFunction, matchedFrame.Function)
				}
				if usedFallback != tt.expectFallback {
					t.Errorf("Expected fallback=%v, got %v", tt.expectFallback, usedFallback)
				}
			} else {
				if err == nil {
					t.Errorf("Expected no match but found frame: %+v", matchedFrame)
				}
			}
		})
	}
}

func TestComputeFingerprintVariations(t *testing.T) {
	tests := []struct {
		name          string
		frame         Frame
		minVariations int
	}{
		{
			name:          "frame with module and function",
			frame:         Frame{Function: "testFunc", Module: "testModule"},
			minVariations: 2,
		},
		{
			name:          "frame with package and function",
			frame:         Frame{Function: "testFunc", Package: "/path/to/lib.so"},
			minVariations: 3,
		},
		{
			name:          "frame with file, module and function",
			frame:         Frame{Function: "testFunc", Module: "testModule", File: "test.py"},
			minVariations: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variations := computeFingerprintVariations(tt.frame)
			if len(variations) < tt.minVariations {
				t.Errorf("Expected at least %d variations, got %d", tt.minVariations, len(variations))
			}

			// Verify first variation matches the standard fingerprint
			if variations[0] != tt.frame.Fingerprint() {
				t.Errorf("First variation should match standard fingerprint")
			}
		})
	}
}

// Helper function to compute fingerprint from file and function.
func computeFingerprintFromFileAndFunction(file, function string) uint32 {
	h := fnv.New64()
	h.Write([]byte(file))
	h.Write([]byte{':'})
	h.Write([]byte(function))
	return uint32(h.Sum64())
}
