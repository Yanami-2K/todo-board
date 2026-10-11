package store

import "testing"

func TestGenLookMatchesPetsJS(t *testing.T) {
	// Vectors computed with the same algorithm as web/js/pets.js.
	cases := []struct {
		name, salt       string
		saltN            int
		ear, color, tool string
		glasses          bool
		h                uint32
	}{
		{"", "", 0, "sprout", "pink", "wrench", false, 2872998923},
		{"remote-dev", "", 0, "sprout", "lav", "mug", false, 542642609},
		{"log-center", "", 0, "bunny", "blue", "none", false, 1548206575},
		{"我", "", 0, "bunny", "yellow", "cube", false, 133277269},
		{"review-bot", "1", 1, "mouse", "blue", "pencil", false, 1503997605},
	}
	for _, tc := range cases {
		if got := HashName(tc.name + func() string {
			if tc.saltN > 0 {
				return "#" + tc.salt
			}
			return ""
		}()); got != tc.h {
			t.Fatalf("hash %q = %d, want %d", tc.name, got, tc.h)
		}
		got := GenLook(tc.name, tc.saltN)
		if got.Ear != tc.ear || got.Color != tc.color || got.Tool != tc.tool || got.Glasses != tc.glasses {
			t.Fatalf("look %q = %+v", tc.name, got)
		}
	}
}
