package open

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/brevdev/brev-cli/pkg/cmd/cmderrors"
	"github.com/brevdev/brev-cli/pkg/cmd/completions"
	"github.com/brevdev/brev-cli/pkg/cmd/hello"
	"github.com/brevdev/brev-cli/pkg/cmd/refresh"
	"github.com/brevdev/brev-cli/pkg/cmd/util"
	"github.com/brevdev/brev-cli/pkg/entity"
	breverrors "github.com/brevdev/brev-cli/pkg/errors"
	"github.com/brevdev/brev-cli/pkg/store"
	"github.com/brevdev/brev-cli/pkg/terminal"
	"github.com/briandowns/spinner"

	"github.com/spf13/cobra"
)

var (
	openLong    = "[command in beta] This will open VS Code SSH-ed in to your workspace. You must have 'code' installed in your path."
	openExample = "brev open workspace_id_or_name\nbrev open my-app\nbrev open h9fp5vxwe"
)

type OpenStore interface {
	util.GetWorkspaceByNameOrIDErrStore
	refresh.RefreshStore
	UpdateUser(string, *entity.UpdateUser) (*entity.User, error)
	GetOrganizations(options *store.GetOrganizationsOptions) ([]entity.Organization, error)
	GetWorkspaces(organizationID string, options *store.GetWorkspacesOptions) ([]entity.Workspace, error)
	StartWorkspace(workspaceID string) (*entity.Workspace, error)
	GetActiveOrganizationOrDefault() (*entity.Organization, error)
	GetWorkspace(workspaceID string) (*entity.Workspace, error)
}

func NewCmdOpen(t *terminal.Terminal, store OpenStore, noLoginStartStore OpenStore) *cobra.Command {
	var runRemotCMD bool

	cmd := &cobra.Command{
		Annotations:           map[string]string{"ssh": ""},
		Use:                   "open",
		DisableFlagsInUseLine: true,
		Short:                 "[beta] open VSCode to ",
		Long:                  openLong,
		Example:               openExample,
		Args:                  cmderrors.TransformToValidationError(cobra.ExactArgs(1)),
		ValidArgsFunction:     completions.GetAllWorkspaceNameCompletionHandler(noLoginStartStore, t),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runOpenCommand(t, store, args[0], runRemotCMD)
			if err != nil {
				return breverrors.WrapAndTrace(err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&runRemotCMD, "remote", "r", true, "run remote command")

	return cmd
}

// Fetch workspace info, then open code editor
func runOpenCommand(t *terminal.Terminal, tstore OpenStore, wsIDOrName string, runRemoteCMD bool) error {
	// todo check if workspace is stopped and start if it if it is stopped
	fmt.Println("finding your dev environment...")

	res := refresh.RunRefreshAsync(tstore, runRemoteCMD)

	workspace, err := util.GetUserWorkspaceByNameOrIDErr(tstore, wsIDOrName)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	if workspace.Status == "STOPPED" { // we start the env for the user
		err = startWorkspaceIfStopped(t, tstore, wsIDOrName, workspace)
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
	}
	workspace, err = util.GetUserWorkspaceByNameOrIDErr(tstore, wsIDOrName)

	if workspace.Status != "RUNNING" {
		return breverrors.WorkspaceNotRunning{Status: workspace.Status}
	}

	projPath := workspace.GetProjectFolderPath()

	localIdentifier := workspace.GetLocalIdentifier()

	err = res.Await()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	err = hello.SetHasRunOpen(true)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	err = openVsCodeWithSSH(t, string(localIdentifier), projPath, tstore, runRemoteCMD)
	if err != nil {
		if strings.Contains(err.Error(), `"code": executable file not found in $PATH`) {
			errMsg := "code\": executable file not found in $PATH\n\nadd 'code' to your $PATH to open VS Code from the terminal\n\texport PATH=\"/Applications/Visual Studio Code.app/Contents/Resources/app/bin:$PATH\""
			_, errStore := tstore.UpdateUser(
				workspace.CreatedByUserID,
				&entity.UpdateUser{
					OnboardingData: map[string]interface{}{
						"pathErrorTS": time.Now().UTC().Unix(),
					},
				})
			if errStore != nil {
				return errors.New(errMsg + "\n" + errStore.Error())
			}
			return errors.New(errMsg)
		}
		return breverrors.WrapAndTrace(err)
	}
	return nil
}

func pollUntil(t *terminal.Terminal, wsid string, state string, openStore OpenStore) error {
	s := t.NewSpinner()
	isReady := false
	s.Suffix = " hang tight 🤙"
	s.Start()
	for !isReady {
		time.Sleep(5 * time.Second)
		ws, err := openStore.GetWorkspace(wsid)
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
		s.Suffix = "  workspace is currently " + strings.ToLower(ws.Status)
		if ws.Status == state {
			s.Suffix = "Workspace is ready!"
			s.Stop()
			isReady = true
		}
	}
	return nil
}

func startWorkspaceIfStopped(t *terminal.Terminal, tstore OpenStore, wsIDOrName string, workspace *entity.Workspace) error {
	activeOrg, err := tstore.GetActiveOrganizationOrDefault()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	workspaces, err := tstore.GetWorkspaceByNameOrID(activeOrg.ID, wsIDOrName)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	startedWorkspace, err := tstore.StartWorkspace(workspaces[0].ID)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	t.Vprintf(t.Yellow("Dev environment %s is starting. \n\n", startedWorkspace.Name))
	err = pollUntil(t, workspace.ID, entity.Running, tstore)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	workspace, err = util.GetUserWorkspaceByNameOrIDErr(tstore, wsIDOrName)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	return nil
}

// Opens code editor. Attempts to install code in path if not installed already
func openVsCodeWithSSH(t *terminal.Terminal, sshAlias string, path string, tstore OpenStore, runRemoteCMD bool) error {
	// infinite for loop:
	res := refresh.RunRefreshAsync(tstore, runRemoteCMD)
	err := res.Await()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	s := t.NewSpinner()
	s.Start()
	waitForSSHToBeAvailable(t, s, sshAlias)

	waitForLoggerFileToBeAvailable(t, s, sshAlias)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	setupFinished, err := checkSetupFinished(sshAlias)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	if !setupFinished {
		err = streamOutput(t, s, sshAlias, path)
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
	} else {
		err = openVsCode(sshAlias, path)
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
	}
	return nil
}

func waitForSSHToBeAvailable(t *terminal.Terminal, s *spinner.Spinner, sshAlias string) {
	counter := 0
	for {
		cmd := exec.Command("ssh", "-o", "RemoteCommand=none", "-o", "ConnectTimeout=1", sshAlias, "echo", "hello")
		_, err := cmd.CombinedOutput()
		if err == nil {
			return
		}
		if counter == 10 {
			s.Suffix = t.Green(" booting up your machine 🤙")
		}

		counter++
		time.Sleep(1 * time.Second)
	}
}

func waitForLoggerFileToBeAvailable(t *terminal.Terminal, s *spinner.Spinner, sshAlias string) {
	counter := 0
	for {
		cmd := exec.Command("ssh", "-o", "RemoteCommand=none", "-o", "ConnectTimeout=1", sshAlias, "tail", "-n", "1", "/var/log/brev-workspace.log")
		_, err := cmd.CombinedOutput()
		if err == nil {
			return
		}
		if counter == 5 {
			s.Suffix = t.Green(" setting up Ubuntu...")
		}
		if counter == 20 {
			s.Suffix = t.Green(" installing a few more bits and bobs... ")
		}
		if counter == 27 {
			s.Suffix = t.Green(" this is only slow the first time... ")
		}
		counter++
		time.Sleep(1 * time.Second)

	}
}

func checkSetupFinished(sshAlias string) (bool, error) {
	out, err := exec.Command("ssh", "-o", "RemoteCommand=none", sshAlias, "tail", "-n", "50", "/var/log/brev-workspace.log").CombinedOutput() // RemoteCommand=none
	if err != nil {
		return false, breverrors.WrapAndTrace(err)
	}
	return strings.Contains(string(out), "------ Setup End ------"), nil
}

func streamOutput(t *terminal.Terminal, s *spinner.Spinner, sshAlias string, path string) error {
	s.Suffix = t.Green(" should be no more than a minute now...hit ENTER to see logs")
	cmd := exec.Command("ssh", "-o", "RemoteCommand=none", sshAlias, "tail", "-f", "/var/log/brev-workspace.log")
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	vscodeAlreadyOpened := false
	showLogsToUser := false
	scanner := bufio.NewScanner(cmdReader)
	errChannel := make(chan error)

	go scanLoggerFile(t, scanner, sshAlias, path, s, &vscodeAlreadyOpened, &showLogsToUser, errChannel)

	err = cmd.Start()
	if err != nil {
		os.Exit(0)
		return breverrors.WrapAndTrace(err)
	}
	go showLogsToUserIfTheyPressEnter(t, sshAlias, &showLogsToUser, s)
	out := <-errChannel
	if out != nil {
		return out
	}
	err = cmd.Wait()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	return nil
}

func scanLoggerFile(t *terminal.Terminal, scanner *bufio.Scanner, sshAlias string, path string, s *spinner.Spinner, vscodeAlreadyOpened *bool, showLogsToUser *bool, err chan error) {
	for scanner.Scan() {
		if *showLogsToUser {
			fmt.Println(t.Yellow("\n", scanner.Text()))
		}
		if strings.Contains(scanner.Text(), "------ Setup End ------") || strings.Contains(scanner.Text(), "------ Git repo cloned ------") {
			if !*vscodeAlreadyOpened {
				s.Stop()
				err <- openVsCode(sshAlias, path)
				*vscodeAlreadyOpened = true

			}
		}
		if strings.Contains(scanner.Text(), "------ Setup End ------") {
			s.Stop()
			os.Exit(0)
		}
	}
}

func openVsCode(sshAlias string, path string) error {
	vscodeString := fmt.Sprintf("vscode-remote://ssh-remote+%s%s", sshAlias, path)
	vscodeString = shellescape.QuoteCommand([]string{vscodeString})
	cmd := exec.Command("code", "--folder-uri", vscodeString) // #nosec G204
	err := cmd.Run()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	return nil
}

func showLogsToUserIfTheyPressEnter(t *terminal.Terminal, sshAlias string, showLogsToUser *bool, s *spinner.Spinner) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		*showLogsToUser = true
		out, err := exec.Command("ssh", "-o", "RemoteCommand=none", sshAlias, "cat", "/var/log/brev-workspace.log").CombinedOutput()
		fmt.Print(t.Yellow(string(out)))
		if err != nil {
			fmt.Println(err)
		}
		s.Suffix = ""
	}
}
