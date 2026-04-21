package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/lintingzhen/commitizen-go/commit"
	"github.com/lintingzhen/commitizen-go/git"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	all     bool
	isDebug bool
)

func GetRootCmd() (*cobra.Command, error) {
	var rootCmd = &cobra.Command{
		Use:  "commitizen-go",
		Long: `Command line utility to standardize git commit messages, golang version.`,
		RunE: rootFnE,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
		},
	}

	err := initConfig()
	if err != nil {
		return nil, err
	}

	doc := "tell the command to automatically stage files" +
		" that have been modified and deleted, but new files you have" +
		" not told Git about are not affected"
	rootCmd.Flags().BoolVarP(&all, "all", "a", false, doc)
	rootCmd.PersistentFlags().BoolVarP(&isDebug, "debug", "d", false, "debug mode, output debug info to debug.log")

	rootCmd.AddCommand(VersionCmd)
	rootCmd.AddCommand(InstallCmd)

	return rootCmd, nil
}

// initConfig sets up logging and loads the .git-czrc config file via Viper.
func initConfig() error {
	if !isDebug {
		log.SetOutput(io.Discard)
	} else {
		f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open debug.log: %w", err)
		}
		// defer f.Close()
		log.SetFlags(log.Lshortfile | log.LstdFlags)
		log.SetOutput(f)
	}

	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("get home dir failed: %w", err)
	}
	workingTreeRoot, err := git.WorkingTreeRoot()
	if err != nil || workingTreeRoot == "" {
		return errors.New("current directory is not a git working tree")
	}

	// Search config in repository working tree root directory and home directory with name ".git-czrc" or ".git-czrc.json".
	viper.AddConfigPath(workingTreeRoot)
	viper.AddConfigPath(home)
	viper.SetConfigName(".git-czrc")
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Println("can not find config file")
		} else {
			// Config file was found but another error was produced
			log.Printf("read config failed, err=%v\n", err)
		}
	} else {
		log.Println("read config success")
	}

	return nil
}

// rootFnE is the Cobra handler for the root command: presents the commit form and runs git commit.
func rootFnE(_ *cobra.Command, _ []string) error {
	if _, err := git.IsCurrentDirectoryGitRepo(); err != nil {
		return fmt.Errorf("does not seem to be a git repo: %w", err)
	}

	if message, err := commit.FillOutForm(); err == nil {
		// do git commit
		result, err := git.Commit(message, all)
		if err != nil {
			log.Printf("commit message is: \n\n%s\n\n", string(message))
			return fmt.Errorf("run git commit failed: %w", err)
		}

		_, _ = fmt.Print(string(result))
	}

	return nil
}
