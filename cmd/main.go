package main

import (
	"fmt"

	"github.com/ostafen/digler/cmd/cmd"
)

var (
	Version    = "dev" // Default to "dev" if not set by build flags
	CommitHash = "none"
	BuildTime  = "unknown"
)

func main() {
	PrintLogo()

	_ = cmd.Execute()
}

func PrintLogo() {
	fmt.Println("    _ _        _          ")
	fmt.Println("  __| (_) __ _| | ___ _ __")
	fmt.Println(" / _` | |/ _` | |/ _ \\ '__|")
	fmt.Println("| (_| | | (_| | |  __/ |   ")
	fmt.Println(" \\__,_|_|\\__, |_|\\___|_|   ")
	fmt.Println("          |___/           ")
	fmt.Println()
	fmt.Println("Disk analysis and recovery tool")
	fmt.Println()
	fmt.Printf("Version:   %s\n", Version)
	fmt.Printf("Commit:    %s\n", CommitHash)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Println(" ")
}
