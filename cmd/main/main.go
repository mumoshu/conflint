package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mumoshu/conflint"
)

func flagUsage() {
	text := `Run various configuration linters and print aggregated results for CI

Usage:
  conflint [command]
Available Commands:
  run		Runs linters against certain files and print results as configured

Use "conflint [command] --help" for more information about a command
`

	fmt.Fprintf(os.Stderr, "%s\n", text)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	flag.Usage = flagUsage

	CmdRun := "run"

	if len(os.Args) == 1 {
		flag.Usage()
		return
	}

	switch os.Args[1] {
	case CmdRun:
		runCmd := flag.NewFlagSet(CmdRun, flag.ExitOnError)
		configFile := runCmd.String("c", "conflint.yaml", "Configuration file to be loaded")
		errformat := runCmd.String("efm", "%f:%l:%c: %m", "errorformat-style output format. Specify the same format to reviewdog for integration")
		delim := runCmd.String("d", ": ", "Delimiter between the jsonpath part and the message part. For a linter error `$.apiVersion| apiVersion must be apps/v1` and `-d '|'`, `$.apiVersion` is considered as the jsonpath part, and the `apiVersion must be apps/v1` as the message part")

		if err := runCmd.Parse(os.Args[2:]); err != nil {
			fatal("%v", err)
		}

		wd, err := os.Getwd()
		if err != nil {
			fatal("%v", err)
		}

		runner := &conflint.Runner{
			ConfigFile: *configFile,
			Errformat:  *errformat,
			Output:     os.Stdout,
			WorkDir:    wd,
			Delim:      *delim,
			LogLevel:   os.Getenv("CONFLINT_LOG"),
		}

		if err := runner.Run(); err != nil {
			fatal("%v", err)
		}
	default:
		flag.Usage()
	}
}
