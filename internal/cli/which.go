package cli

import (
	"os"

	"github.com/florian-balling/sub-switch/internal/config"
	"github.com/florian-balling/sub-switch/internal/resolver"
	"github.com/spf13/cobra"
)

func newWhichCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "which <agent>", Args: cobra.ExactArgs(1), Short: "show selected project/profile", RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := config.Load(opts.configPath)
		if err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		analysis, err := resolver.AnalyzeRun(*c, cwd, args[0])
		if err != nil {
			return err
		}
		if !analysis.Ready() {
			cmd.Printf("[sub-switch] denied: %s\n", analysis.Reason)
			return nil
		}
		cmd.Printf("[sub-switch] %s -> profile %s (%s)\n", analysis.Agent, analysis.Profile, analysis.ProjectPath)
		return nil
	}}
}
