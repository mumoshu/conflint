package conflint

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type Runner struct {
	Output     io.Writer
	ConfigFile string
	Errformat  string
	WorkDir    string
	Delim      string
	LogLevel   string
}

type Config struct {
	Conftest []ConftestConfig `yaml:"conftest"`
	Kubeval  []KubevalConfig  `yaml:"kubeval"`
}

type ConftestConfig struct {
	Files         []string `yaml:"files"`
	Policy        string   `yaml:"policy"`
	Input         string   `yaml:"input"`
	Combine       bool     `yaml:"combine"`
	FailOnWarn    bool     `yaml:"failOnWarn"`
	Data          []string `yaml:"data"`
	AllNamespaces bool     `yaml:"allNamespaces`
	Namespaces    []string `yaml:"namespaces"`
}

type KubevalConfig struct {
	Files                   []string `yaml:"files"`
	Strict                  bool     `yaml:"strict"`
	SchemaLocations         []string `yaml:"schemaLocations"`
	IgnoreMissingSchemas    bool     `yaml:"ignoreMissingSchemas"`
	IgnoredFilenamePatterns []string `yaml:"ignoredFilenamePatterns`
	SkipKinds               []string `yaml:"skipKinds"`
}

type KubevalOutput = []KubevalFileResult

type KubevalFileResult struct {
	Filename string   `yaml:"filename"`
	Kind     string   `yaml:"kind"`
	Status   string   `yaml:"status"`
	Errors   []string `yaml:"errors"`
}

type ConftestOutput = []ConftestFileResult

type ConftestFileResult struct {
	Filename string           `yaml:"filename"`
	Warnings []ConftestResult `yaml:"warnings"`
	Failures []ConftestResult `yaml:"failures"`
}

type ConftestResult struct {
	Msg string `yaml:"msg"`
}

func (r *Runner) Run() error {
	var config Config

	file := filepath.Join(r.WorkDir, r.ConfigFile)
	bs, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(bs, &config); err != nil {
		return err
	}

	var output int

	for _, ct := range config.Conftest {
		for _, fp := range ct.Files {
			files, err := filepath.Glob(filepath.Join(r.WorkDir, fp))
			if err != nil {
				return fmt.Errorf("searching files matching %s: %w", fp, err)
			}

			var fs []string

			for _, f := range files {
				f = strings.TrimPrefix(f, r.WorkDir)
				f = strings.TrimPrefix(f, "/")

				fs = append(fs, f)
			}

			args := []string{"test"}
			args = append(args, fs...)
			args = append(args, "-p", ct.Policy, "-o", "json")

			if ct.Input != "" {
				args = append(args, "-i", ct.Input)
			}
			if ct.Combine {
				args = append(args, "--combine")
			}
			if ct.AllNamespaces {
				args = append(args, "--all-namespaces")
			}
			if len(ct.Data) > 0 {
				args = append(args, "--data", strings.Join(ct.Data, ","))
			}
			if len(ct.Namespaces) > 0 {
				args = append(args, "--namespace", strings.Join(ct.Namespaces, ","))
			}

			cmd := exec.Command("conftest", args...)
			cmd.Dir = r.WorkDir
			out, err := cmd.CombinedOutput()
			if err != nil && r.LogLevel == "DEBUG" {
				fmt.Fprintf(os.Stderr, "DEBUG: running conftest %s: %v\n", strings.Join(args, " "), err)
			}

			var conftestOut ConftestOutput

			if err := yaml.Unmarshal(out, &conftestOut); err != nil {
				return err
			}

			for _, res := range conftestOut {
				handle := func(msg string) error {
					sub := strings.SplitN(msg, r.Delim, 2)
					if len(sub) > 1 {
						line, col, err := getLineColFromJsonpathExpr(filepath.Join(r.WorkDir, res.Filename), "$."+sub[0])
						if err != nil {
							return fmt.Errorf("processing %s: %w", sub[0], err)
						}
						if err := r.Print(res.Filename, line, col, sub[1]); err != nil {
							return fmt.Errorf("printing %s: %w", sub[1], err)
						}
						output++
					} else {
						log.Printf("ignoring unsupported output: %s", msg)
					}

					return nil
				}

				for _, f := range res.Failures {
					if err := handle(f.Msg); err != nil {
						return err
					}
				}
			}
		}
	}

	for _, ke := range config.Kubeval {
		for _, fp := range ke.Files {
			files, err := filepath.Glob(filepath.Join(r.WorkDir, fp))
			if err != nil {
				return fmt.Errorf("searching files matching %s: %w", fp, err)
			}

			for _, f := range files {
				f = strings.TrimPrefix(f, r.WorkDir)
				f = strings.TrimPrefix(f, "/")

				args := []string{f, "-o", "json"}
				if ke.Strict {
					args = append(args, "--strict")
				}
				if ke.IgnoreMissingSchemas {
					args = append(args, "--ignore-missing-schemas")
				}
				if len(ke.IgnoredFilenamePatterns) > 0 {
					args = append(args, "--ignored-filename-patterns", strings.Join(ke.IgnoredFilenamePatterns, ","))
				}
				if len(ke.SkipKinds) > 0 {
					args = append(args, "--skip-kinds", strings.Join(ke.SkipKinds, ","))
				}
				if len(ke.SchemaLocations) > 0 {
					args = append(args, "--schema-location", ke.SchemaLocations[0])

					if len(ke.SchemaLocations) > 1 {
						args = append(args, "--additional-schema-locations", strings.Join(ke.SchemaLocations[1:], ","))
					}
				}
				cmd := exec.Command("kubeval", args...)
				cmd.Dir = r.WorkDir
				out, err := cmd.CombinedOutput()
				if err != nil && r.LogLevel == "DEBUG" {
					fmt.Fprintf(os.Stderr, "DEBUG: running kubeval %s: %v\n", strings.Join(args, " "), err)
				}

				var conftestOut KubevalOutput

				if err := yaml.Unmarshal(out, &conftestOut); err != nil {
					return err
				}

				for _, res := range conftestOut {
					handle := func(msg string) error {
						sub := strings.SplitN(msg, ": ", 2)
						if len(sub) > 1 {
							line, col, err := getLineColFromJsonpathExpr(filepath.Join(r.WorkDir, f), "$."+sub[0])
							if err != nil {
								return fmt.Errorf("processing %s: %w", sub[0], err)
							}
							if err := r.Print(f, line, col, sub[1]); err != nil {
								return fmt.Errorf("printing %s: %w", sub[1], err)
							}
							output++
						} else {
							log.Printf("ignoring unsupported output: %s", msg)
						}

						return nil
					}

					for _, f := range res.Errors {
						if err := handle(f); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	if output > 0 {
		var word string
		if output > 1 {
			word = "errors"
		} else {
			word = "error"
		}
		return fmt.Errorf("found %d linter %s", output, word)
	}

	return nil
}

func getLineColFromJsonpathExpr(file string, jsonpathExpr string) (int, int, error) {
	if jsonpathExpr[0] != '$' {
		return 0, 0, fmt.Errorf("Expression must start with $, but got: %s", jsonpathExpr)
	}

	path, err := parseJsonpath(jsonpathExpr)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing jsonpath %s: %w", jsonpathExpr, err)
	}

	f, err := os.Open(file)
	if err != nil {
		return 0, 0, fmt.Errorf("opening file %s: %w", file, err)
	}

	defer f.Close()

	dec := yaml.NewDecoder(f)

	next := func() (*yaml.Node, error) {
		doc := &yaml.Node{}

		if err := dec.Decode(doc); err != nil {
			if err == io.EOF {
				return nil, nil
			}
			return nil, fmt.Errorf("decoding yaml from %s: %w", file, err)
		}

		if doc.Kind != yaml.DocumentNode {
			panic(fmt.Errorf("the top-level yaml node must be a document node. got %v", doc.Kind))
		}

		node := doc.Content[0]

		if node.Kind != yaml.MappingNode {
			panic(fmt.Errorf("the only yaml node in a document must be a mapping node. got %v", node.Kind))
		}

		got, err := path.Get(node)
		if err != nil {
			return nil, fmt.Errorf("getting node at %s: %w", jsonpathExpr, err)
		}

		return got, nil
	}

	var lastErr error

	for {
		node, err := next()
		if node != nil {
			return node.Line, node.Column, nil
		}

		if err == nil {
			break
		}

		lastErr = err
	}

	if lastErr != nil {
		return 0, 0, fmt.Errorf("getting line and column numbers from %s: %w", file, lastErr)
	}

	return 0, 0, fmt.Errorf("gettling line and colum numbers from %s: no value found at %s", file, jsonpathExpr)
}

func (r *Runner) Print(file string, line, col int, msg string) error {
	// TODO maybe use https://github.com/phayes/checkstyle for additional checkstyle xml output?

	replacer := strings.NewReplacer("%m", msg, "%f", file, "%l", fmt.Sprintf("%d", line), "%c", fmt.Sprintf("%d", col))

	printed := replacer.Replace(r.Errformat)

	if _, err := r.Output.Write([]byte(printed)); err != nil {
		return err
	}

	if _, err := r.Output.Write([]byte("\n")); err != nil {
		return err
	}

	return nil
}
