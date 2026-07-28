package testutil

import (
	"os"
	"testing"
)

// RequireOPNsense skips an integration test unless all connection settings are available.
func RequireOPNsense(t *testing.T) {
	t.Helper()

	for _, name := range []string{"OPNSENSE_URI", "OPNSENSE_API_KEY", "OPNSENSE_API_SECRET"} {
		if os.Getenv(name) == "" {
			t.Skipf("%s is not set; skipping OPNsense integration test", name)
		}
	}
}
