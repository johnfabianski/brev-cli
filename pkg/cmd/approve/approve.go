package approve

import (
	"github.com/brevdev/brev-cli/pkg/entity"
	breverrors "github.com/brevdev/brev-cli/pkg/errors"
	"github.com/brevdev/brev-cli/pkg/terminal"
	"github.com/spf13/cobra"
)

var (
	approveDescriptionShort = "Brev admin approve user"
	approveDescriptionLong  = "Brev admin approve user"
	approveExample          = "brev approve <user id>"
)

type ApproveStore interface {
	ApproveUserByID(userID string) (*entity.User, error)
}

func NewCmdApprove(t *terminal.Terminal, approveStore ApproveStore) *cobra.Command {
	cmd := &cobra.Command{
		Annotations:           map[string]string{"workspace": ""},
		Use:                   "reset",
		DisableFlagsInUseLine: true,
		Short:                 approveDescriptionShort,
		Long:                  approveDescriptionLong,
		Example:               approveExample,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := approveUser(args[0], t, approveStore)
			if err != nil {
				t.Vprint(t.Red(err.Error()))
			}
		},
	}

	return cmd
}

func approveUser(userID string, _ *terminal.Terminal, approveStore ApproveStore) error {
	_, err := approveStore.ApproveUserByID(userID)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	return nil
}
