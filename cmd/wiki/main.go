// Command wiki is a fast, friendly command line for Wikipedia.
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"

	"github.com/charmbracelet/fang"
	"github.com/tamnd/wikipedia-cli/cli"
)

// Build metadata, injected via -ldflags at release time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.Version, cli.Commit, cli.Date = version, commit, date

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	root := cli.Root()
	err := fang.Execute(ctx, root, fang.WithVersion(version))
	os.Exit(exitCode(err))
}

// exitCode maps an error to the documented exit codes: commands attach a code
// via the exitCoder interface; everything else is a generic runtime failure.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var coder interface{ ExitCode() int }
	if errors.As(err, &coder) {
		return coder.ExitCode()
	}
	return 1
}
