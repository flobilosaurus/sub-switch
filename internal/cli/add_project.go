package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/florian-balling/sub-switch/internal/config"
	"github.com/florian-balling/sub-switch/internal/launcher"
	"github.com/spf13/cobra"
)

func newAddProjectCommand(opts *options) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "add-project <agent>=<profile>...",
		Short: "add the current folder to the config",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cfgPath, err := config.Load(opts.configPath)
			if err != nil {
				return err
			}

			profiles, err := parseProfileMappings(args)
			if err != nil {
				return err
			}
			configuredAgents, err := ensureProjectAgentsConfigured(cfg, profiles)
			if err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			cwd, err = filepath.Abs(cwd)
			if err != nil {
				return err
			}
			cwd = filepath.Clean(cwd)

			created := false
			project := findProjectRule(cfg.Projects, cwd)
			if project == nil {
				cfg.Projects = append(cfg.Projects, config.ProjectRule{Path: cwd, Profiles: map[string]string{}})
				project = &cfg.Projects[len(cfg.Projects)-1]
				created = true
			}
			if project.Profiles == nil {
				project.Profiles = map[string]string{}
			}

			for agent, profile := range profiles {
				if existing, ok := project.Profiles[agent]; ok && existing != profile && !force {
					return fmt.Errorf("project %s already maps %s to profile %q (use --force to replace with %q)", cwd, agent, existing, profile)
				}
				project.Profiles[agent] = profile
			}

			if err := cfg.Validate(); err != nil {
				return err
			}
			if err := config.Save(cfgPath, *cfg, true); err != nil {
				return err
			}

			if created {
				cmd.Printf("added project\t%s\n", cwd)
			} else {
				cmd.Printf("updated project\t%s\n", cwd)
			}
			for _, agent := range configuredAgents {
				cmd.Printf("configured\t%s\t%s\n", agent, cfg.Agents[agent].Command)
			}
			for agent, profile := range profiles {
				cmd.Printf("profile\t%s\t%s\n", agent, profile)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "replace existing profile mappings for this folder")
	return cmd
}

func parseProfileMappings(args []string) (map[string]string, error) {
	profiles := make(map[string]string, len(args))
	for _, arg := range args {
		agent, profile, ok := strings.Cut(arg, "=")
		agent = strings.TrimSpace(agent)
		profile = strings.TrimSpace(profile)
		if !ok || agent == "" || profile == "" {
			return nil, fmt.Errorf("profile mapping must use <agent>=<profile>, got %q", arg)
		}
		profiles[agent] = profile
	}
	return profiles, nil
}

func ensureProjectAgentsConfigured(cfg *config.Config, profiles map[string]string) ([]string, error) {
	if cfg.Agents == nil {
		cfg.Agents = map[string]config.AgentConfig{}
	}
	configured := []string{}
	for agent := range profiles {
		if _, ok := cfg.Agents[agent]; ok {
			continue
		}
		command, err := exec.LookPath(agent)
		if err != nil {
			return nil, fmt.Errorf("agent %q is not configured and could not be found on PATH: %w", agent, err)
		}
		if abs, err := filepath.Abs(command); err == nil {
			command = abs
		}
		if err := launcher.ValidateCommand(command); err != nil {
			return nil, fmt.Errorf("cannot add agent %s with command %s: %w", agent, command, err)
		}
		cfg.Agents[agent] = config.AgentConfig{Command: command}
		configured = append(configured, agent)
	}
	return configured, nil
}

func findProjectRule(projects []config.ProjectRule, path string) *config.ProjectRule {
	for i := range projects {
		if filepath.Clean(projects[i].Path) == path {
			return &projects[i]
		}
	}
	return nil
}
