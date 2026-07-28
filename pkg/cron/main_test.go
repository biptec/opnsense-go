package cron

import (
	"fmt"
	"os"
	"testing"
)

// Integration tests in this package require a live OPNsense instance. Skip
// cleanly when the required environment variables are unset.
func TestMain(m *testing.M) {
	if os.Getenv("OPNSENSE_URI") == "" || os.Getenv("OPNSENSE_API_KEY") == "" || os.Getenv("OPNSENSE_API_SECRET") == "" {
		fmt.Fprintln(os.Stderr, "OPNSENSE_URI/OPNSENSE_API_KEY/OPNSENSE_API_SECRET not set; skipping integration tests in pkg/cron")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
