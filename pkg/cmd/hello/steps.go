package hello

import (
	"fmt"
	"time"

	"github.com/brevdev/brev-cli/pkg/entity"
	breverrors "github.com/brevdev/brev-cli/pkg/errors"
	"github.com/brevdev/brev-cli/pkg/terminal"
	"github.com/brevdev/brev-cli/pkg/util"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

const DefaultDevEnvName = "first-workspace-react"

func GetTextBasedONStatus(status string, t *terminal.Terminal) string {
	s := ""
	switch status {
	case "RUNNING":
	case "DEPLOYING":
		s += t.Yellow("Your dev environment is deploying.")
		s += "\nPlease wait for it to finish deploying then run " + t.Yellow("brev hello") + " to resume this walk through when your dev env is ready\n"
	case "UNHEALTHY":
		s += t.Red("Your dev environment seems stuck. Can you reach out to support?")
		s += "\nMessage us "
		s += "\n\t in discord 👉 " + t.Yellow("https://discord.gg/RpszWaJFRA")
		s += "\n\t via text or call 👉 " + t.Yellow("(415) 237-2247\n")
		s += "\n\nRun " + t.Yellow("brev hello") + " to resume this walk through when your dev env is ready\n"
	case "STOPPED":
		s += t.Yellow("Your dev environment is stopped.")
		s += "\nRun this in your terminal to start it 👉 " + t.Yellow("brev start %s", DefaultDevEnvName)
		s += "\n\nRun " + t.Yellow("brev hello") + " to resume this walk through when your dev env is ready\n"

	case "STOPPING":
		s += t.Yellow("Your dev environment is stopped.")
		s += "\nRun this in your terminal to start it 👉 " + t.Yellow("brev start %s", DefaultDevEnvName)
		s += "\n\nRun " + t.Yellow("brev hello") + " to resume this walk through when your dev env is ready\n"
	default:
		s += t.Red("Please create a running dev environment for this walk through. ")
		s += "\n\tYou can do that here: " + t.Yellow("https://console.brev.dev/environments/new")
		s += "\n\t\t-- Or --\n\tRun this in your terminal 👉 " + t.Yellow("brev start https://github.com/brevdev/hello-react --name %s", DefaultDevEnvName)
		s += "\n\nRun " + t.Yellow("brev hello") + " to resume this walk through when your dev env is ready\n"
	}
	return s
}

/*
Return nil to exit the onboarding
*/
func GetDevEnvOrStall(t *terminal.Terminal, workspaces []entity.Workspace) *entity.Workspace {
	var runningDevEnvs []entity.Workspace
	noneFound := true
	for _, v := range workspaces {
		if v.Status == "RUNNING" {
			noneFound = false
			runningDevEnvs = append(runningDevEnvs, v)
		}
	}

	if noneFound {
		s := t.Red("Please create a running dev environment for this walk through. ")
		s += "\n\tYou can do that here: " + t.Yellow("https://console.brev.dev/environments/new")
		s += "\n\t\t-- Or --\n\tRun this in your terminal 👉 " + t.Yellow("brev start https://github.com/brevdev/hello-react --name %s", DefaultDevEnvName)
		s += "\n\nRun: " + t.Yellow("brev hello") + " to resume this walk through when your dev env is ready\n"
		TypeItToMe(s)
		return nil
	}
	msg := GetTextBasedONStatus(runningDevEnvs[0].Status, t)
	if msg != "" {
		TypeItToMe(msg)
	}
	return &runningDevEnvs[0]
}

func printLsIntroText(t *terminal.Terminal, firstWorkspace entity.Workspace) {
	s := "\n\nThe command " + t.Yellow("brev ls") + " shows your dev environments"
	s += "\nIf the dev environment is " + t.Green("RUNNING") + ", you can open it."
	s += "\n\nIn a new terminal, try running " + t.Green("brev shell %s", firstWorkspace.Name) + " to get a terminal in your dev environment\n"
	TypeItToMe(s)
}

func printAskInstallVsCode(t *terminal.Terminal) {
	// The error here is most likely because code isn't in path and we depend on that
	// TODO: remove the dependency on code being in path
	s := t.Yellow("\n\nCould you please install the following VSCode extension? %s", t.Green("ms-vscode-remote.remote-ssh"))
	s += "\nDo that then run " + t.Yellow("brev hello") + " to resume this walk-through\n"
	// s += "Here's a video of me installing the VS Code extension 👉 " + ""
	TypeItToMe(s)
}

func printBrevOpen(t *terminal.Terminal, firstWorkspace entity.Workspace) {
	s := "\n\nAwesome! Now try opening VS Code in that environment"
	s += "\nIn a new terminal, try running " + t.Green("brev open %s", firstWorkspace.Name) + " to open VS Code in the dev environment\n"
	TypeItToMe(s)
}

func printCompletedOnboarding(t *terminal.Terminal) {
	s := "\n\nI think I'm done here. Now you know how to open a dev environment and start coding."
	s += "\n\nUse the console " + t.Green("(https://console.brev.dev)") + " to create a new dev environment or share it with people"
	s += "\nand use this CLI to code the way you would normally 🤙"
	s += "\n\nCheck out the docs at " + t.Yellow("https://brev.dev/docs") + " and let us know if we can help!\n"
	s += "\n\nIn case you missed it, my cell is " + t.Yellow("(415) 237-2247") + "\n\t-Nader\n"
	TypeItToMe(s)
}

func waitSpinner(spinner *spinner.Spinner) error {
	// a while loop in golang
	sum := 0
	spinner.Suffix = "👆 try that, I'll wait"
	spinner.Start()
	for sum > -1 {
		sum++

		res, err2 := GetOnboardingObject()
		if err2 != nil {
			return breverrors.WrapAndTrace(err2)
		}
		if res.HasRunBrevShell {
			spinner.Suffix = "🎉 you did it!"
			time.Sleep(250 * time.Millisecond)
			spinner.Stop()
			break
		}
		time.Sleep(1 * time.Second)

	}
	return nil
}

/*
Step 1:

	The user just ran brev ls
*/
func Step1(t *terminal.Terminal, workspaces []entity.Workspace, user *entity.User, store HelloStore) error {
	err := CompletedOnboardingLs(user, store)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	firstWorkspace := GetDevEnvOrStall(t, workspaces)
	if firstWorkspace == nil {
		return nil
	}
	printLsIntroText(t, *firstWorkspace)
	spinner := t.NewSpinner()
	err = waitSpinner(spinner)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	err = CompletedOnboardingShell(user, store)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	TypeItToMe("\nHit " + t.Yellow("enter") + " to continue")
	fmt.Println()
	bold := color.New(color.Bold).SprintFunc()
	_ = terminal.PromptGetInput(terminal.PromptContent{
		// Label:      "   " + bold("▸") + "    Press " + bold("Enter") + " to continue",
		Label:      "   " + bold("▸"),
		ErrorMsg:   "error",
		AllowEmpty: true,
	})

	// TODO: check if VS Code is preferred editor
	currentOnboardingStatus, err := user.GetOnboardingData()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	if currentOnboardingStatus.Editor == "VSCode" {
		err = doVsCodeOnboarding(t, firstWorkspace, user, store, spinner, bold)
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
	}

	// Commenting out the below since public urls is gone
	// handleLocalhostURLIfDefaultProject(*firstWorkspace, t)
	printCompletedOnboarding(t)
	err = CompletedOnboarding(user, store)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	return nil
}

// func handleLocalhostURLIfDefaultProject(ws entity.Workspace, t *terminal.Terminal) {
// 	if ws.Name == DefaultDevEnvName {
// 		s := "\n\nOne last thing, since you're coding in the cloud, you can get a public URL to your localhost."
// 		s += "\nFrom within that Brev dev environment,\n\tRun " + t.Yellow("npm run start") + " to spin up the service"
// 		s += "\nThen instead of going to localhost:3000, \n\tGo to " + t.Yellow("https://3000-%s", ws.DNS)

// 		// TODO: Give that a shot then press enter
// 		bold := color.New(color.Bold).SprintFunc()

// 		s += "\n\nGive that a shot then press enter👆:"
// 		TypeItToMe(s)

// 		fmt.Print("\n")
// 		_ = terminal.PromptGetInput(terminal.PromptContent{
// 			// Label:      "   " + bold("▸") + "    Press " + bold("Enter") + " to continue",
// 			Label:      "   " + bold("▸"),
// 			ErrorMsg:   "error",
// 			AllowEmpty: true,
// 		})

// 		fmt.Print("\n")
// 	}
// }

func doVsCodeOnboarding(
	t *terminal.Terminal,
	firstWorkspace *entity.Workspace,
	user *entity.User,
	store HelloStore,
	spinner *spinner.Spinner,
	bold func(a ...interface{}) string,
) error {
	// TODO: check if ext is installed
	isInstalled, err2 := util.IsVSCodeExtensionInstalled("ms-vscode-remote.remote-ssh")
	if !isInstalled || err2 != nil {
		printAskInstallVsCode(t)
		return nil
	}

	printBrevOpen(t, *firstWorkspace)

	// a while loop in golang
	sum := 0
	spinner.Suffix = "☝️ try that, I'll wait"
	spinner.Start()
	for sum < 1 {
		sum += sum
		res, err2 := GetOnboardingObject()
		if err2 != nil {
			return breverrors.WrapAndTrace(err2)
		}
		if res.HasRunBrevOpen {
			spinner.Suffix = "🎉 you did it!"
			time.Sleep(250 * time.Millisecond)
			spinner.Stop()
			break
		}
		time.Sleep(1 * time.Second)

	}

	err := CompletedOnboardingOpen(user, store)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	TypeItToMe("\nHit " + t.Yellow("enter") + " to continue")
	fmt.Println()

	_ = terminal.PromptGetInput(terminal.PromptContent{
		// Label:      "   " + bold("▸") + "    Press " + bold("Enter") + " to continue",
		Label:      "   " + bold("▸"),
		ErrorMsg:   "error",
		AllowEmpty: true,
	})
	return nil
}
