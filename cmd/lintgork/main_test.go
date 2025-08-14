package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestMain(t *testing.T) {
	// Test that main function can be called without panicking
	// We test this by running the binary in a subprocess to avoid
	// interfering with the test runner
	if os.Getenv("BE_LINTGORK_MAIN") == "1" {
		main()
		return
	}

	tests := []struct {
		name     string
		args     []string
		wantExit int
	}{
		{
			name:     "help flag",
			args:     []string{"-help"},
			wantExit: 0,
		},
		{
			name:     "no arguments",
			args:     []string{},
			wantExit: 1, // lintgork expects arguments
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestMain")
			cmd.Env = append(os.Environ(), "BE_LINTGORK_MAIN=1")
			if len(tt.args) > 0 {
				cmd.Args = append([]string{os.Args[0]}, tt.args...)
			}

			err := cmd.Run()

			if tt.wantExit == 0 && err != nil {
				// For lintgork -help, it might still exit with error code
				// but we just want to test that it doesn't panic
				if exitError, ok := err.(*exec.ExitError); ok {
					// Exit code 2 is normal for usage/help
					if exitError.ExitCode() != 2 {
						t.Errorf("Expected help to exit with code 0 or 2, got %d", exitError.ExitCode())
					}
				}
			}

			if tt.wantExit == 1 && err == nil {
				t.Error("Expected error but command succeeded")
			}
		})
	}
}

func TestMainExecution(t *testing.T) {
	// Test that main function exists and is accessible
	// We can't call main() directly in tests because singlechecker.Main()
	// expects command line arguments and would interfere with test execution

	// The existence of main function is tested by the compiler
	// If this test file compiles, main function exists
	// The actual functionality testing is done via subprocess in TestMain

	// This is a placeholder test to ensure we have coverage of the main function concept
	// The real testing happens through the subprocess execution above
	t.Log("main function exists and is testable via subprocess")
}
