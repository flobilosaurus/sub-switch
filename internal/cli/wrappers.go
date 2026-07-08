package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/florian-balling/sub-switch/internal/config"
	"github.com/florian-balling/sub-switch/internal/launcher"
	"github.com/florian-balling/sub-switch/internal/wrappers"
	"github.com/spf13/cobra"
)

func newInstallWrappersCommand(opts *options) *cobra.Command {
	var dir string
	var force bool
	cmd := &cobra.Command{Use: "install-wrappers <agent> --dir <path>", Short: "add an agent and install PATH wrappers", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if dir == "" {
			return cmd.Help()
		}
		agent := args[0]
		cfg, cfgPath, err := config.Load(opts.configPath)
		if err != nil {
			return err
		}
		command, err := exec.LookPath(agent)
		if err != nil {
			return fmt.Errorf("could not find real command for %s on PATH: %w", agent, err)
		}
		if abs, err := filepath.Abs(command); err == nil {
			command = abs
		}
		if err := launcher.ValidateCommand(command); err != nil {
			return fmt.Errorf("cannot add agent %s with command %s: %w", agent, command, err)
		}
		if cfg.Agents == nil {
			cfg.Agents = map[string]config.AgentConfig{}
		}
		cfg.Agents[agent] = config.AgentConfig{Command: command}
		if err := cfg.Validate(); err != nil {
			return err
		}
		if err := config.Save(cfgPath, *cfg, true); err != nil {
			return err
		}
		cmd.Printf("configured\t%s\t%s\n", agent, command)
		exe, _ := os.Executable()
		res, err := wrappers.Install(dir, exe, config.AgentNames(cfg.Agents), force)
		if err != nil {
			return err
		}
		refused := 0
		for _, r := range res {
			if r.Err != nil {
				refused++
				cmd.Printf("%s\t%s\t%s: %v\n", r.Action, r.Agent, r.Path, r.Err)
			} else {
				cmd.Printf("%s\t%s\t%s\n", r.Action, r.Agent, r.Path)
			}
		}
		if refused > 0 {
			return nil
		}
		return nil
	}}
	cmd.Flags().StringVar(&dir, "dir", "", "directory for wrapper executables")
	_ = cmd.MarkFlagRequired("dir")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite unrelated files")
	return cmd
}
