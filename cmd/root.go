package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var appVersion = "v1.0.0"

var rootCmd = &cobra.Command{
	Use:   "kbot",
	Short: "Telegram bot written in Go",
	Long:  "kbot - Telegram bot built with cobra and telebot.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
