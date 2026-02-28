package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"timon/internal/buildinfo"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		fmt.Printf("Timon %s\n", buildinfo.Version)
		return nil
	},
}
