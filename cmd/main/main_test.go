package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRun(t *testing.T) {
	if os.Getenv("RUN_MAIN_FOR_TESTING") == "1" {
		os.Args = []string{"conflint", "run"}

		// We DO call helm's main() here. So this looks like a normal `helm` process.
		main()

		// As main calls os.Exit, we never reach this line.
		// But the test called this block of code catches and verifies the exit code.
		return
	}

	testcases := []struct {
		dir     string
		wantOut string
		wantErr string
	}{
		{
			dir:     "simple",
			wantOut: "app1/nginx.deploy.yaml:15:11: `privileged: true` is forbidden\n",
			wantErr: "Error: found 1 linter error\n",
		},
		{
			dir:     "kubeval-fail",
			wantOut: "app1/nginx.deploy.yaml:18:25: Invalid type. Expected: [boolean,null], given: string\n",
			wantErr: "Error: found 1 linter error\n",
		},
	}

	for i := range testcases {
		tc := testcases[i]

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			// Do a second run of this specific test with RUN_MAIN_FOR_TESTING=1 set,
			// So that the second run is able to run main() and this first run can verify the exit status returned by that.
			//
			// This technique originates from https://talks.golang.org/2014/testing.slide#23.
			cmd := exec.Command(os.Args[0], "-test.run=TestRun")
			cmd.Env = append(
				os.Environ(),
				"RUN_MAIN_FOR_TESTING=1",
			)
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			cmd.Dir = filepath.Join("..", "..", "testdata", tc.dir)
			err := cmd.Run()
			exiterr, ok := err.(*exec.ExitError)

			if !ok {
				t.Fatalf("Unexpected error returned by os.Exit: %T", err)
			}

			if diff := cmp.Diff(stdout.String(), tc.wantOut); diff != "" {
				t.Errorf("Unexpected write to stdout: %s", diff)
			}

			if diff := cmp.Diff(stderr.String(), tc.wantErr); diff != "" {
				t.Errorf("Unexpected write to stderr: %s", diff)
			}

			if exiterr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1: Got %d", exiterr.ExitCode())
			}
		})
	}
}
