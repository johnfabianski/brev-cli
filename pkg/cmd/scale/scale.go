package scale

import (
	"github.com/spf13/cobra"

	"github.com/brevdev/brev-cli/pkg/cmd/util"
	breverrors "github.com/brevdev/brev-cli/pkg/errors"
	"github.com/brevdev/brev-cli/pkg/terminal"
)

var (
	long    = "Scale your Brev environment to get a more powerful machine or save costs"
	example = `
  brev start <existing_ws_name>
  brev start <git url>
  brev start <git url> --org myFancyOrg
	`
	instanceTypes = []string{"p4d.24xlarge", "p3.2xlarge", "p3.8xlarge", "p3.16xlarge", "p3dn.24xlarge", "p2.xlarge", "p2.8xlarge", "p2.16xlarge", "g5.xlarge", "g5.2xlarge", "g5.4xlarge", "g5.8xlarge", "g5.16xlarge", "g5.12xlarge", "g5.24xlarge", "g5.48xlarge", "g5g.xlarge", "g5g.2xlarge", "g5g.4xlarge", "g5g.8xlarge", "g5g.16xlarge", "g5g.metal", "g4dn.xlarge", "g4dn.2xlarge", "g4dn.4xlarge", "g4dn.8xlarge", "g4dn.16xlarge", "g4dn.12xlarge", "g4dn.metal", "g4ad.xlarge", "g4ad.2xlarge", "g4ad.4xlarge", "g4ad.8xlarge", "g4ad.16xlarge", "g3s.xlarge", "g3.4xlarge", "g3.8xlarge", "g3.16xlarge"}
)

type ScaleStore interface {
	util.GetWorkspaceByNameOrIDErrStore
}

func NewCmdScale(t *terminal.Terminal, store ScaleStore) *cobra.Command {
	var instanceType string

	cmd := &cobra.Command{
		Use:                   "scale",
		DisableFlagsInUseLine: true,
		Short:                 "Scale your Brev environment",
		Long:                  long,
		Example:               "brev scale MyEnvironment --instance-type p3.2xlarge",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return breverrors.NewValidationError("You must provide an instance with flage -i")
			}

			err := Runscale(t, args, instanceType, store)
			if err != nil {
				return breverrors.WrapAndTrace(err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&instanceType, "instance", "i", "", "GPU or CPU instance type.  See docs.brev.dev/gpu for details")
	return cmd
}

func Runscale(t *terminal.Terminal, args []string, instanceType string, store ScaleStore) error {
	workspace, err := util.GetUserWorkspaceByNameOrIDErr(store, args[0])
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	t.Vprintf("\nScaling %s to %s\n", workspace.Name, instanceType)

	return nil
}