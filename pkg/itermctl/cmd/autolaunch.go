package cmd

import (
	"bytes"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"io"
	"mrz.io/itermctl/pkg/itermctl/internal/shell"
	"os"
	"path"
	"strings"
)

const autolaunchDir = "~/Library/ApplicationSupport/iTerm2/Scripts/AutoLaunch"

var AutolaunchCommand = &cobra.Command{
	Use:   "autolaunch",
	Short: "Generate a Python script for iTerm2's autolaunch",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		saveAs, err := cmd.Flags().GetString("save-as")
		if err != nil {
			return err
		}

		executablePath, err := shell.Which(args[0])
		if err != nil {
			return err
		}

		quotedPath := quote(executablePath)[0]
		quotedArgs := quote(args...)

		var br io.Reader
		var f *os.File

		bw := &bytes.Buffer{}
		br = bw

		if saveAs != "" {
			autolaunchPath, err := homedir.Expand(autolaunchDir)
			if err != nil {
				return err
			}

			autolaunchScriptPath := path.Join(autolaunchPath, fmt.Sprintf("%s.py", saveAs))

			f, err = os.Create(autolaunchScriptPath)
			if err != nil {
				return err
			}

			br = io.TeeReader(bw, f)
		}

		_, _ = fmt.Fprintf(bw, "from os import execv\nexecv(%s, [%s])\n", quotedPath, strings.Join(quotedArgs, ", "))

		_, err = io.Copy(os.Stdout, br)
		if err != nil {
			return err
		}

		if f != nil {
			if err := f.Close(); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(os.Stderr, "saved to %s\n", f.Name())
		}

		return nil
	},
}

func init() {
	AutolaunchCommand.Flags().String("save-as", "",
		"in addition to print the script also save it to iTerm2's AutoLaunch directory with the given name")
}

func quote(strings ...string) (quoted []string) {
	for _, s := range strings {
		quoted = append(quoted, fmt.Sprintf("%q", s))
	}

	return
}
