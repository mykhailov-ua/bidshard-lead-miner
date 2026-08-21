package main

import (
	"context"
	"os"
)

var Version = "dev"

func main() {
	rootCmd.SetContext(context.Background())
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
