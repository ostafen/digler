package main

import (
	"fmt"

	"github.com/ostafen/digler/cmd/cmd"
	"github.com/ostafen/digler/internal/env"
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
	fmt.Printf("Version:   %s\n", env.Version)
	fmt.Printf("Commit:    %s\n", env.CommitHash)
	fmt.Printf("Build Time: %s\n", env.BuildTime)
	fmt.Println(" ")
}
