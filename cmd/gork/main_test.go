package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestMain(t *testing.T) {
	// Test that main function can be called without panicking
	// We'll test this by running the binary in a subprocess
	if os.Getenv("BE_MAIN") == "1" {
		main()
		return
	}

	tests := []struct {
		name     string
		args     []string
		wantExit int
	}{
		{
			name:     "help command",
			args:     []string{"--help"},
			wantExit: 0,
		},
		{
			name:     "invalid flag",
			args:     []string{"--invalid-flag"},
			wantExit: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestMain")
			cmd.Env = append(os.Environ(), "BE_MAIN=1")
			cmd.Args = append([]string{os.Args[0]}, tt.args...)
			
			err := cmd.Run()
			
			if tt.wantExit == 0 && err != nil {
				t.Errorf("Expected success but got error: %v", err)
			}
			
			if tt.wantExit == 1 && err == nil {
				t.Errorf("Expected error but command succeeded")
			}
		})
	}
}

func TestMainErrorHandling(t *testing.T) {
	// Test main function with intentionally failing CLI
	if os.Getenv("TEST_MAIN_ERROR") == "1" {
		// This will cause an error since we're passing invalid args
		os.Args = []string{"gork", "--nonexistent-flag"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainErrorHandling")
	cmd.Env = append(os.Environ(), "TEST_MAIN_ERROR=1")
	
	err := cmd.Run()
	if err == nil {
		t.Error("Expected main to exit with error, but it didn't")
	}

	// Check exit code
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() != 1 {
			t.Errorf("Expected exit code 1, got %d", exitError.ExitCode())
		}
	} else {
		t.Errorf("Expected ExitError, got %T", err)
	}
}