package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/lintingzhen/commitizen-go/git"
	"github.com/spf13/cobra"
)

// InstallCmd copies the current binary into Git's exec path as "git-cz".
var InstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install this tool to git-core as git-cz",
	RunE: func(_ *cobra.Command, _ []string) error {
		appFilePath, _ := exec.LookPath(os.Args[0])
		path, err := git.InstallSubCmd(appFilePath, "cz")
		if err != nil {
			return fmt.Errorf("failed to install %s: %w", Name, err)
		}

		fmt.Printf("Install commitizen to %s\n", path)

		return nil
	},
}
