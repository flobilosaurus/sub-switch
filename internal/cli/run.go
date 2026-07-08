package cli

import (
	"fmt"
	"os"

	"github.com/florian-balling/sub-switch/internal/config"
	"github.com/florian-balling/sub-switch/internal/launcher"
	"github.com/florian-balling/sub-switch/internal/profile"
	"github.com/florian-balling/sub-switch/internal/resolver"
	"github.com/spf13/cobra"
)

func newRunCommand(opts *options) *cobra.Command {
	var quiet bool
	cmd := &cobra.Command{Use: "run <agent> -- [args...]", Short: "run real agent with selected profile", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		agent := args[0]
		pass := args[1:]
		c, _, err := config.Load(opts.configPath)
		if err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		sel, err := resolver.Resolve(*c, cwd, agent)
		if err != nil {
			return err
		}
		if sel.Denied {
			return fmt.Errorf("[sub-switch] denied: %s", sel.Reason)
		}
		ac, ok := c.Agents[agent]
		if !ok || ac.Command == "" {
			return fmt.Errorf("configured command for %s is missing", agent)
		}
		env, err := profile.BuildForCurrentUser(sel.Profile, agent)
		if err != nil {
			return err
		}
		if err := env.Ensure(); err != nil {
			return err
		}
		if c.UI.StartupBanner && !quiet {
			cmd.Printf("[sub-switch] %s -> profile %s (%s)\n", agent, sel.Profile, sel.ProjectPath)
		}
		return launcher.Run(launcher.CommandSpec{Command: ac.Command, Args: pass, Env: env.Merge(os.Environ()), CWD: cwd})
	}}
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress startup banner")
	cmd.DisableFlagParsing = false
	return cmd
}
