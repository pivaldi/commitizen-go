package main

import (
	"log"

	"github.com/lintingzhen/commitizen-go/cmd"
)

func main() {
	rootCmd, err := cmd.GetRootCmd()
	if err != nil {
		log.Fatal(err)
	}

	rootCmd.Execute()
}
