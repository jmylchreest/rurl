package cli

import (
	"fmt"

	"github.com/jmylchreest/rurl/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Display version, build date, and git commit information for rurl.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("rurl %s\n", config.GetVersionInfo())
	},
}
