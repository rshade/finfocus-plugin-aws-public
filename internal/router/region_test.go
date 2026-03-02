package router

import "testing"

func TestValidateRegion(t *testing.T) {
	tests := []struct {
		name   string
		region string
		ok     bool
	}{
		{"standard", "us-east-1", true},
		{"govcloud", "us-gov-west-1", true},
		{"empty", "", false},
		{"bad format", "use1", false},
		{"path traversal", "../us-east-1", false},
		{"path separator", "us-east-1/../../x", false},
		{"windows separator", "us-east-1\\x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegion(tt.region)
			if tt.ok && err != nil {
				t.Fatalf("expected ok, got err=%v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}
