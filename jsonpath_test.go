package conflint

import (
	"fmt"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

func TestJsonpath(t *testing.T) {
	testcases := []struct {
		expr             string
		data             string
		jsonpathParseErr string
		jsonpathGetErr   string
		line, col        int
		val              string
	}{
		{
			expr: `$.foo.bar`,
			data: `foo:
  bar: 1
`,
			line: 2,
			col:  8,
			val:  "1",
		},
		{
			expr: `$.spec.containers[*]?(@.privileged == true).privileged`,
			data: `spec:
  containers:
  - name: nginx
    privileged: true
`,
			line: 4,
			col:  17,
			val:  "true",
		},
		{
			expr: `$.spec.containers[?(@.privileged == true)].privileged`,
			data: `spec:
  containers:
  - name: nginx
    privileged: true
`,
			line: 4,
			col:  17,
			val:  "true",
		},
		{
			expr: `$.spec.containers[?(@.name == 'nginx' && @.privileged == true)].privileged`,
			data: `spec:
  containers:
  - name: fluentd
    privileged: true
  - name: nginx
    privileged: true
`,
			line: 6,
			col:  17,
			val:  "true",
		},
		{
			expr: `$.spec.containers[1].privileged`,
			data: `spec:
  containers:
  - name: fluentd
    privileged: true
  - name: nginx
    privileged: true
`,
			line: 6,
			col:  17,
			val:  "true",
		},
		{
			expr: `$.spec.containers.1.privileged`,
			data: `spec:
  containers:
  - name: fluentd
    privileged: true
  - name: nginx
    privileged: true
`,
			line: 6,
			col:  17,
			val:  "true",
		},
		{
			expr: `$.spec.containers`,
			data: `spec:
  containers:
  - name: fluentd
    privileged: true
  - name: nginx
    privileged: true
`,
			line: 3,
			col:  3,
		},
	}

	for i := range testcases {
		tc := testcases[i]

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()

			path, err := parseJsonpath(tc.expr)
			if err != nil {
				if tc.jsonpathParseErr == "" {
					t.Fatalf("unexpected error: %v", err)
				} else if err.Error() != tc.jsonpathParseErr {
					t.Fatalf("unexpected error: want %q, got %q", tc.jsonpathParseErr, err.Error())
				}
			} else if tc.jsonpathParseErr != "" {
				t.Fatalf("expected error: want %q, got none", tc.jsonpathParseErr)
			}

			root := yaml.Node{}

			if err := yaml.Unmarshal([]byte(tc.data), &root); err != nil {
				t.Fatal("bug: failed parsing yaml")
			}

			mappingNode := root.Content[0]

			got, err := path.Get(mappingNode)
			if err != nil {
				if tc.jsonpathGetErr == "" {
					t.Fatalf("unexpected error: %w", err)
				} else if err.Error() != tc.jsonpathGetErr {
					t.Fatalf("unexpected error: want %q, got %q", tc.jsonpathGetErr, err.Error())
				}
			} else if tc.jsonpathGetErr != "" {
				t.Fatalf("expected error: want %w, got none", tc.jsonpathGetErr)
			}

			if got.Value != tc.val {
				t.Errorf("unexpected result: want %v, got %v", tc.val, got.Value)
			}

			if got.Column != tc.col {
				t.Errorf("unexpected column: want %v, got %v", tc.col, got.Column)
			}

			if got.Line != tc.line {
				t.Errorf("unexpected line: want %v, got %v", tc.line, got.Line)
			}
		})
	}
}
