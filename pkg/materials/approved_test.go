package materials

import "testing"

func TestIsAlwaysApproved(t *testing.T) {
	tests := []struct {
		material string
		want     bool
	}{
		{"image:alpine", true},
		{"image:nginx", true},
		{"apk:nginx", true},
		{"apk:curl", true},
		{"git:some/repo", false},
		{"golang", false},
		{"postgres:14", false},
		{"alpine", false},
	}

	for _, tt := range tests {
		t.Run(tt.material, func(t *testing.T) {
			if got := IsAlwaysApproved(tt.material); got != tt.want {
				t.Errorf("IsAlwaysApproved(%q) = %v, want %v", tt.material, got, tt.want)
			}
		})
	}
}
