package api

import "testing"

// TestCheckLANToken exercises the pure token-check at LAN mode's auth gate
// table-style, per the LAN-reachable mode PRD's Testing Decisions: no token
// configured always allows (default/localhost-only mode is unaffected);
// once a token is configured, only a matching presented token is allowed.
func TestCheckLANToken(t *testing.T) {
	cases := []struct {
		name            string
		configuredToken string
		presentedToken  string
		want            bool
	}{
		{
			name:            "no token configured allows even with no header",
			configuredToken: "",
			presentedToken:  "",
			want:            true,
		},
		{
			name:            "no token configured allows any presented value",
			configuredToken: "",
			presentedToken:  "whatever",
			want:            true,
		},
		{
			name:            "token configured and header matches allows",
			configuredToken: "s3cret",
			presentedToken:  "s3cret",
			want:            true,
		},
		{
			name:            "token configured and header missing rejects",
			configuredToken: "s3cret",
			presentedToken:  "",
			want:            false,
		},
		{
			name:            "token configured and header mismatched rejects",
			configuredToken: "s3cret",
			presentedToken:  "wrong",
			want:            false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkLANToken(tc.configuredToken, tc.presentedToken)
			if got != tc.want {
				t.Errorf("checkLANToken(%q, %q) = %v, want %v", tc.configuredToken, tc.presentedToken, got, tc.want)
			}
		})
	}
}
