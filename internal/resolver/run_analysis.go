package resolver

import (
	"fmt"
	"path/filepath"

	"github.com/florian-balling/sub-switch/internal/config"
)

// RunStatus describes what is missing (if anything) for a complete run launch.
type RunStatus int

const (
	// RunReady means all requirements are met and the agent can launch.
	RunReady RunStatus = iota
	// RunNoProject means no project rule matches the cwd.
	RunNoProject
	// RunProjectMissingAgent means a project matches but has no mapping for the requested agent.
	RunProjectMissingAgent
	// RunMissingProfile means the project maps the agent to a profile that doesn't exist in top-level profiles.
	RunMissingProfile
	// RunMissingProfileAgent means the top-level profile exists but lacks the agent entry.
	RunMissingProfileAgent
	// RunMissingAgentCommand means agents[agent].command is not configured.
	RunMissingAgentCommand
)

// RunAnalysis contains the full result of analyzing whether a run can proceed.
type RunAnalysis struct {
	Agent       string
	CWD         string
	ProjectPath string
	Profile     string
	Status      RunStatus
	Reason      string
	// ProjectIndex is the index of the matched project rule in the config, or -1 if none.
	ProjectIndex int
}

// Ready returns true if the analysis indicates the run can launch.
func (a RunAnalysis) Ready() bool {
	return a.Status == RunReady
}

// AnalyzeRun checks all requirements for launching an agent run.
// It returns the first missing requirement found, checking in this order:
//  1. Project match (longest exact-or-child prefix)
//  2. Agent mapping in matched project
//  3. Top-level profile existence
//  4. Profile-agent entry existence
//  5. Agent command configuration
func AnalyzeRun(c config.Config, cwd, agent string) (RunAnalysis, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return RunAnalysis{}, err
	}
	abs = physicalPath(abs)

	base := RunAnalysis{Agent: agent, CWD: abs, ProjectIndex: -1}

	// 1. Find matching project (longest prefix).
	var bestIdx int = -1
	bestLen := -1
	for i := range c.Projects {
		p := physicalPath(c.Projects[i].Path)
		if matches(abs, p) && len(p) > bestLen {
			bestIdx = i
			bestLen = len(p)
		}
	}

	if bestIdx == -1 {
		base.Status = RunNoProject
		base.Reason = fmt.Sprintf("no project rule matches %s", abs)
		return base, nil
	}

	project := c.Projects[bestIdx]
	base.ProjectPath = project.Path
	base.ProjectIndex = bestIdx

	// 2. Agent mapping in matched project.
	profileName, ok := project.Profiles[agent]
	if !ok || profileName == "" {
		base.Status = RunProjectMissingAgent
		base.Reason = fmt.Sprintf("project %s has no profile for %s", project.Path, agent)
		return base, nil
	}
	base.Profile = profileName

	// 3. Top-level profile existence.
	pc, profileExists := c.Profiles[profileName]
	if !profileExists {
		base.Status = RunMissingProfile
		base.Reason = fmt.Sprintf("profile %q referenced by project %s is not defined in top-level profiles", profileName, project.Path)
		return base, nil
	}

	// 4. Profile-agent entry existence.
	if _, agentEntryExists := pc[agent]; !agentEntryExists {
		base.Status = RunMissingProfileAgent
		base.Reason = fmt.Sprintf("profile %q does not have an entry for agent %s", profileName, agent)
		return base, nil
	}

	// 5. Agent command configuration.
	ac, agentConfigured := c.Agents[agent]
	if !agentConfigured || ac.Command == "" {
		base.Status = RunMissingAgentCommand
		base.Reason = fmt.Sprintf("no configured command for agent %s", agent)
		return base, nil
	}

	base.Status = RunReady
	return base, nil
}
