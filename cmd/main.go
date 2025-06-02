// Copyright (c) 2025 Stefano Scafiti
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
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
