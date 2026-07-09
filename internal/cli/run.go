package cli

import (
	"errors"
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
		c, cfgPath, err := config.Load(opts.configPath)
		if err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		analysis, err := resolver.AnalyzeRun(*c, cwd, agent)
		if err != nil {
			return err
		}

		if !analysis.Ready() {
			if !isTTY() {
				return fmt.Errorf("[sub-switch] denied: %s (run from a terminal to set this up interactively)", analysis.Reason)
			}

			printer := func(format string, a ...interface{}) {
				cmd.Printf(format, a...)
			}

			updated, setupErr := runSetup(huhRunPrompter{}, c, cfgPath, analysis, printer)
			if setupErr != nil {
				if errors.Is(setupErr, ErrSetupAborted) {
					return fmt.Errorf("[sub-switch] %w", setupErr)
				}
				return setupErr
			}
			c = updated

			// Re-analyze with updated config.
			analysis, err = resolver.AnalyzeRun(*c, cwd, agent)
			if err != nil {
				return err
			}
			if !analysis.Ready() {
				return fmt.Errorf("[sub-switch] denied: %s (setup did not fully resolve configuration)", analysis.Reason)
			}
		}

		ac, ok := c.Agents[agent]
		if !ok || ac.Command == "" {
			return fmt.Errorf("configured command for %s is missing", agent)
		}
		profileEnv := map[string]string{}
		if pc, ok := c.Profiles[analysis.Profile]; ok {
			if apc, ok := pc[agent]; ok {
				profileEnv = apc.Env
			}
		}
		env, err := profile.BuildForCurrentUserWithEnv(analysis.Profile, agent, profileEnv)
		if err != nil {
			return err
		}
		if err := env.Ensure(); err != nil {
			return err
		}
		if c.UI.StartupBanner && !quiet {
			cmd.Printf("[sub-switch] %s -> profile %s (%s)\n", agent, analysis.Profile, analysis.ProjectPath)
		}
		return launcher.Run(launcher.CommandSpec{Command: ac.Command, Args: pass, Env: env.Merge(os.Environ()), CWD: cwd})
	}}
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress startup banner")
	cmd.DisableFlagParsing = false
	return cmd
}
