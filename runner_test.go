package conflint

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRunner(t *testing.T) {
	testcases := []struct {
		dir string
		out string
		err string
	}{
		{
			dir: "simple",
			out: "app1/nginx.deploy.yaml:15:11: `privileged: true` is forbidden\n",
			err: "found 1 linter error",
		},
	}

	for i := range testcases {
		tc := testcases[i]
		t.Run(fmt.Sprintf(tc.dir), func(t *testing.T) {
			buf := &bytes.Buffer{}

			runner := &Runner{
				Output:     buf,
				WorkDir:    filepath.Join("testdata", tc.dir),
				ConfigFile: "conflint.yaml",
				Errformat:  "%f:%l:%c: %m",
				Delim:      ": ",
			}

			err := runner.Run()

			if err != nil {
				if tc.err == "" {
					t.Fatalf("unexpected error: %v", err)
				} else if diff := cmp.Diff(tc.err, err.Error()); diff != "" {
					t.Fatalf("unexpected error: %s", diff)
				}
			} else if tc.err != "" {
				t.Fatalf("expected error: want %v, got none", tc.err)
			}

			out := buf.String()

			if out != tc.out {
				t.Errorf("unexpected output: want\n%s\n\ngot\n%s", tc.out, out)
			}
		})
	}
}
