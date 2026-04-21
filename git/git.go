package git

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InstallSubCmd copies srcFilePath into Git's exec path as "git-<subCmdName>".
func InstallSubCmd(srcFilePath, subCmdName string) (string, error) {
	dstDir, err := execPath()
	if err != nil {
		return "", err
	}

	subCmdFileName := "git-" + subCmdName

	dstFilePath := filepath.Join(dstDir, subCmdFileName)
	if _, err := copyFile(dstFilePath, srcFilePath); err != nil {
		return dstFilePath, err
	}

	return dstFilePath, nil
}

// IsCurrentDirectoryGitRepo reports whether the current working directory is inside a Git repository.
func IsCurrentDirectoryGitRepo() (bool, error) {
	// run git remote command
	cmd := exec.Command("git", "remote")

	var err error
	var stderr io.ReadCloser
	if stderr, err = cmd.StderrPipe(); err != nil {
		return false, err
	}

	if err := cmd.Start(); err != nil {
		return false, err
	}

	var result []byte
	if result, err = io.ReadAll(stderr); err != nil {
		return false, err
	}

	if err := cmd.Wait(); err != nil {
		return false, fmt.Errorf("%s", string(result))
	}

	return true, nil
}

// WorkingTreeRoot returns the absolute path of the top-level directory of the working tree.
func WorkingTreeRoot() (path string, err error) {
	output, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// Commit writes message to a temp file and runs "git commit -F <file>", passing -a if all is true.
func Commit(message []byte, all bool) ([]byte, error) {
	// save the commit message to temp file
	var err error
	var file *os.File
	if file, err = os.CreateTemp("", "COMMIT_MESSAGE_"); err != nil {
		return nil, err
	}
	defer func() {
		_ = os.Remove(file.Name())
	}()

	if _, err := file.Write(message); err != nil {
		return nil, err
	}

	// run git commit command
	cmd := exec.Command("git", "commit", "-F")
	cmd.Args = append(cmd.Args, file.Name())
	if all {
		cmd.Args = append(cmd.Args, "-a")
	}

	return cmd.CombinedOutput()
}

// copyFile copies the file at srcName to dstName with executable permissions (0755).
func copyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer func() {
		_ = src.Close()
	}()

	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return
	}
	defer func() {
		_ = dst.Close()
	}()

	return io.Copy(dst, src)
}

// execPath returns the path reported by "git --exec-path" (Git's core executables directory).
func execPath() (string, error) {
	cmd := exec.Command("git", "--exec-path")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	result, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		return "", err
	}

	return strings.TrimSpace(string(result)), nil
}
