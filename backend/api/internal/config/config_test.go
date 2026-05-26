package config

import "testing"

func TestResolveSecretKey(t *testing.T) {
	const good = "this-is-a-sufficiently-long-secret-key!!"
	const short = "too-short"

	tests := []struct {
		name      string
		appEnv    string
		secret    string
		wantErr   bool
		wantExact string // when set, the returned key must equal this
		wantRand  bool   // when true, expect a generated key (no error, len >= 32)
	}{
		{name: "prod empty rejected", appEnv: "production", secret: "", wantErr: true},
		{name: "prod alias empty rejected", appEnv: "prod", secret: "", wantErr: true},
		{name: "prod short rejected", appEnv: "production", secret: short, wantErr: true},
		{name: "prod good accepted", appEnv: "production", secret: good, wantExact: good},
		{name: "prod case-insensitive", appEnv: "Production", secret: "", wantErr: true},
		{name: "dev empty generates random", appEnv: "development", secret: "", wantRand: true},
		{name: "dev short rejected", appEnv: "development", secret: short, wantErr: true},
		{name: "dev good accepted", appEnv: "development", secret: good, wantExact: good},
		{name: "unknown env treated as non-prod", appEnv: "staging", secret: "", wantRand: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveSecretKey(tt.appEnv, tt.secret)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got key %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantExact != "" && got != tt.wantExact {
				t.Errorf("key = %q, want %q", got, tt.wantExact)
			}
			if tt.wantRand {
				if len(got) < minSecretKeyLen {
					t.Errorf("generated key too short: %d < %d", len(got), minSecretKeyLen)
				}
				if got == tt.secret {
					t.Errorf("expected a generated key, got the input back")
				}
			}
		})
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		appEnv string
		want   bool
	}{
		{"production", true},
		{"Production", true},
		{"PROD", true},
		{"prod", true},
		{"development", false},
		{"staging", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isProduction(tt.appEnv); got != tt.want {
			t.Errorf("isProduction(%q) = %v, want %v", tt.appEnv, got, tt.want)
		}
	}
}
