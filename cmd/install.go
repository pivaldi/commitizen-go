package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// InstallCmd copies the current binary into Git's exec path as "git-cz".
var InstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install this tool to git-core as git-cz",
	RunE: func(_ *cobra.Command, _ []string) error {
		appFilePath, _ := exec.LookPath(os.Args[0])
		path, err := installSubCmd(appFilePath, "cz")
		if err != nil {
			return fmt.Errorf("failed to install %s: %w", Name, err)
		}

		fmt.Printf("Install commitizen to %s\n", path)

		return nil
	},
}

// installSubCmd copies srcFilePath into Git's exec path as "git-<subCmdName>".
func installSubCmd(srcFilePath, subCmdName string) (string, error) {
	dst, err := gitExecPath()
	if err != nil {
		return "", err
	}

	dstFilePath := filepath.Join(dst, "git-"+subCmdName)
	if _, err := copyFile(dstFilePath, srcFilePath); err != nil {
		return dstFilePath, err
	}

	return dstFilePath, nil
}

func copyFile(dstName, srcName string) (int64, error) {
	src, err := os.Open(srcName)
	if err != nil {
		return 0, fmt.Errorf("open source: %w", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return 0, fmt.Errorf("open destination: %w", err)
	}
	defer func() { _ = dst.Close() }()

	return io.Copy(dst, src)
}

func gitExecPath() (string, error) {
	cmd := exec.Command("git", "--exec-path")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("exec-path pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("exec-path start: %w", err)
	}
	result, err := io.ReadAll(stdout)
	if err != nil {
		return "", fmt.Errorf("exec-path read: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("exec-path wait: %w", err)
	}

	return strings.TrimSpace(string(result)), nil
}
