package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Run the standalone UniFi controller mock",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return errors.New("mock: not implemented yet")
	},
}
