// Command specmetry is the Specmetry CLI and local dashboard server.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/XKHoshizora/specmetry/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
