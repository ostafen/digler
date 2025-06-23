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
package cmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ostafen/digler/internal/format"
	"github.com/spf13/cobra"
)

func DefineFormatsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "formats",
		Short: "List all supported file formats",
		Long: `The 'formats' command displays a table of all file formats currently supported by the recovery scanner.
Each format includes its name, associated file extensions, category (e.g., image, document), and magic byte signatures used for detection.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE:         RunFormats,
	}
}

func RunFormats(cmd *cobra.Command, args []string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESC\tSIGNATURES")

	headers, err := format.FileHeaders()
	if err != nil {
		return err
	}

	for _, hdr := range headers {
		signatures := make([]string, len(hdr.Signatures))
		for i, sig := range hdr.Signatures {
			signatures[i] = hex.EncodeToString(sig)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			hdr.Ext,
			hdr.Description,
			strings.Join(signatures, ","),
		)
	}
	return w.Flush()
}
