package cli

import (
	"fmt"

	"github.com/florian-balling/sub-switch/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCommand(opts *options) *cobra.Command {
	var wrapperDir string
	cmd := &cobra.Command{Use:"doctor", Short:"diagnose config, wrappers, and PATH", RunE: func(cmd *cobra.Command,args []string) error { rs := doctor.Run(opts.configPath, wrapperDir); for _, r := range rs { cmd.Printf("%s\t%s\n", r.Severity, r.Message) }; if doctor.HasError(rs) { return fmt.Errorf("doctor found errors") }; return nil }}
	cmd.Flags().StringVar(&wrapperDir,"wrapper-dir","","directory containing managed wrappers")
	return cmd
}
