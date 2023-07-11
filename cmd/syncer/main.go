package main

import (
	"context"
	"fmt"
	"github.com/cresta/syncer/internal/cli"
	"os"
)

func main() {
	cmd := cli.WireRootCommand()
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
