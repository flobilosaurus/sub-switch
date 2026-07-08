package cli

import "github.com/spf13/cobra"

const version = "0.1.0-dev"

type options struct {
	configPath string
}

func NewRootCommand() *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "sub-switch",
		Short: "Select safe agent subscriptions by project folder",
		Long:  "sub-switch selects an allowed profile for supported agent CLIs from the current folder and launches the real command with isolated XDG state.",
	}
	cmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "config file path")
	cmd.Flags().BoolP("version", "v", false, "print version")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		show, _ := cmd.Flags().GetBool("version")
		if show {
			cmd.Println(version)
			return nil
		}
		return cmd.Help()
	}
	cmd.AddCommand(newInitCommand(opts), newWhichCommand(opts), newRunCommand(opts), newInstallWrappersCommand(), newDoctorCommand(opts))
	return cmd
}
