package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the poll -> MQTT presence service",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return errors.New("serve: not implemented yet")
	},
}
