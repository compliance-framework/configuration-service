package main

import (
	"fmt"
	"os"
	"slices"
	"testing"
)

func isIntegrationTestRun() bool {
	return slices.Contains(os.Args, "integration")
}

// TestIntegration runs the full suite of integration tests, requiring a mongo instance.
// Integration tests take significantly longer than units, and should not be run as regularly.
// This function prevents the integration tests from being run when doing `go test ./...`
// In order to run these tests, you must run `go test ./... --args integration
func TestIntegration(t *testing.T) {
	fmt.Println(os.Args)
	fmt.Println(isIntegrationTestRun())
	if !isIntegrationTestRun() {
		t.Skipf("Integration tests disabled. Pass `--args integration` with the test command to enable.")
	}
	t.Logf("Integration tests are active.")
}
