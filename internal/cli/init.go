package cli

import (
	"github.com/florian-balling/sub-switch/internal/config"
	"github.com/spf13/cobra"
)

func newInitCommand(opts *options) *cobra.Command {
	var force bool
	cmd := &cobra.Command{Use: "init", Short: "create starter config", RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Save(opts.configPath, config.StarterConfig(), force); err != nil {
			return err
		}
		p, _ := config.ResolvePath(opts.configPath)
		cmd.Printf("created config: %s\n", p)
		return nil
	}}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config")
	return cmd
}
