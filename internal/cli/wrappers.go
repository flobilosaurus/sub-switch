package cli

import (
	"os"

	"github.com/florian-balling/sub-switch/internal/wrappers"
	"github.com/spf13/cobra"
)

func newInstallWrappersCommand() *cobra.Command {
	var dir string
	var force bool
	cmd := &cobra.Command{Use: "install-wrappers --dir <path>", Short: "install PATH wrappers", RunE: func(cmd *cobra.Command, args []string) error {
		if dir == "" {
			return cmd.Help()
		}
		exe, _ := os.Executable()
		res, err := wrappers.Install(dir, exe, force)
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
