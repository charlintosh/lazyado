package main

import (
    "errors"
    "flag"
    "fmt"
    "os"

    "github.com/charlintosh/lazyado/cmd"
    "github.com/charlintosh/lazyado/internal/debug"
)

func main() {
    // Parse flags at program entry to improve testability of cmd.Execute.
    debugFlag := flag.Bool("debug", false, "Enable debug logging to /tmp/lazyado-debug.log")
    flag.Parse()

    if *debugFlag {
        debug.Enable()
    }

    if err := cmd.Execute(); err != nil {
        if errors.Is(err, cmd.ErrConfigCreated) {
            // cmd.Execute already informed the user; exit successfully so the
            // shell doesn't treat this as an error.
            os.Exit(0)
        }
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
