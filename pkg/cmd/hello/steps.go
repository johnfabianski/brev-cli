package hello

import (
	"fmt"
	"time"

	"github.com/brevdev/brev-cli/pkg/entity"
	"github.com/brevdev/brev-cli/pkg/terminal"
	"github.com/fatih/color"
)

func Stall(t *terminal.Terminal, workspace entity.Workspace) {
}

const DEFAULT_WORKSPACE = "first-workspace-react"

func GetWorkspaceOrStall(t *terminal.Terminal, workspaces []entity.Workspace) entity.Workspace {
	var firstWorkspace entity.Workspace
	var runningWorkspaces []entity.Workspace
	for _, v := range workspaces {
		if v.Name == DEFAULT_WORKSPACE {
			firstWorkspace = v
		}
		if v.Status == "RUNNING" {
			runningWorkspaces = append(runningWorkspaces, v)
		}
	}

	if firstWorkspace.Status == "RUNNING" {
		// all is good, proceed.
		// always prefer to do the demo with the first workspace react cus it's setup properly
		return firstWorkspace
	} else if firstWorkspace.Status == "DEPLOYING" {
		// TODO: STALL
	} else {
		s := t.Yellow("Please create a running dev environment for this walk through. ")
		s += "\nYou can do that here: " + t.Yellow("https://console.brev.dev/environments/new")
		s += "\n\nRun " + t.Yellow("brev hello") + " to start this walk through again"
		TypeItToMe(s)
		// // BANANA: This whole section feels like feature creep
		// // Do they have a running workspace? -> use it
		// if len(runningWorkspaces) > 0 {
		// 	firstWorkspace = runningWorkspaces[0]
		// } else {
		// 	// No running workspaces, do they have a workspace that is deploying? -> use it
		// 	for _, v := range workspaces {
		// 		if v.Status == "DEPLOYING" {
		// 			firstWorkspace = v
		// 			// STALL
		// 		}
		// 	}

		// 	// No workspace? -> tell them to create one...
		// 	t.Vprintf("\n You don't have a dev environment yet. Go to the console to create a new one: https://console.brev.dev")
		// 	res := terminal.PromptSelectInput(terminal.PromptSelectContent{
		// 		Label:    "Want me to create a demo environment for you?",
		// 		ErrorMsg: "Please pick yes or no",
		// 		Items:    []string{"Yes", "No thanks, I'll do it'"},
		// 	})
		// 	if res == "Yes" {
		// 		// TODO: create workspace react....
		// 	}
		// }
	}

	return firstWorkspace
}

/*
	Step 1:
		The user just ran brev ls
*/
func Step1(t *terminal.Terminal, workspaces []entity.Workspace) {
	firstWorkspace := GetWorkspaceOrStall(t, workspaces)

	s := "\n\nThe command " + t.Yellow("brev ls") + " shows your dev environments"
	s += "\nIf the dev environment is " + t.Green("RUNNING") + ", you can open it."
	s += "\n\nIn a new terminal, try running " + t.Green("brev shell %s", firstWorkspace.Name) + " to get a terminal in your dev environment\n"
	TypeItToMe(s)

	// Reset the onboarding object to walk through the onboarding fresh
	res, err := GetOnboardingObject()
	if err != nil {
		return
	}
	SetOnboardingObject(OnboardingObject{res.Step, false, false})

	// a while loop in golang
	sum := 0
	spinner := t.NewSpinner()
	spinner.Suffix = "👆 try that, I'll wait"
	spinner.Start()
	for sum > -1 {
		sum += 1
		res, err := GetOnboardingObject()
		if err != nil {
			return
		}
		if res.HasRunBrevShell {
			spinner.Suffix = "🎉 you did it!"
			time.Sleep(250 * time.Millisecond)
			spinner.Stop()
			break
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	s = "\nHit enter to continue:"
	TypeItToMe(s)

	fmt.Print("\n")
	bold := color.New(color.Bold).SprintFunc()
	_ = terminal.PromptGetInput(terminal.PromptContent{
		// Label:      "   " + bold("▸") + "    Press " + bold("Enter") + " to continue",
		Label:      "   " + bold("▸"),
		ErrorMsg:   "error",
		AllowEmpty: true,
	})

	// TODO: "ready to try opening vs code to that environment?"
	s = "\n\nAwesome! Now try opening VS Code in that environment"
	s += "\nIn a new terminal, try running " + t.Green("brev open %s", firstWorkspace.Name) + " to open VS Code in the dev environment\n"
	TypeItToMe(s)

	// a while loop in golang
	sum = 0
	spinner.Suffix = "☝️ try that, I'll wait"
	spinner.Start()
	for sum < 1 {
		sum += sum
		res, err := GetOnboardingObject()
		if err != nil {
			return
		}
		if res.HasRunBrevOpen {
			spinner.Suffix = "🎉 you did it!"
			time.Sleep(250 * time.Millisecond)
			spinner.Stop()
			sum += 1
			break
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	s = "\nHit enter to continue:"
	TypeItToMe(s)

	fmt.Print("\n")
	_ = terminal.PromptGetInput(terminal.PromptContent{
		// Label:      "   " + bold("▸") + "    Press " + bold("Enter") + " to continue",
		Label:      "   " + bold("▸"),
		ErrorMsg:   "error",
		AllowEmpty: true,
	})

	handleLocalhostURLIfDefaultProject(firstWorkspace, t)

	s = "\n\nI think I'm done here. Now you know how to open a dev environment and start coding."
	s += "\n\nUse the console " + t.Green("(https://console.brev.dev)") + " to create a new dev environment or share it with people"
	s += "\nand use this CLI to code the way you would normally 🤙"
	s += "\n\nCheck out the docs at " + t.Yellow("https://brev.dev/docs") + " and let us know if we can help!\n"
	s += "\n\nIn case you missed it, my cell is " + t.Yellow("(415) 237-2247") + "\n\t-Nader\n"
	TypeItToMe(s)
}

func handleLocalhostURLIfDefaultProject(ws entity.Workspace, t *terminal.Terminal) {
	if ws.Name == DEFAULT_WORKSPACE {
		s := "\n\nOne last thing, since you're coding in the cloud, you can get a public URL to your localhost."
		s += "\nFrom within that Brev dev environment,\n\tRun " + t.Yellow("npm run start") + " to spin up the service"
		s += "\nThen instead of going to localhost:3000, \n\tGo to " + t.Yellow("https://3000-%s", ws.DNS)

		TypeItToMe(s)

		// TODO: Give that a shot then press enter
		bold := color.New(color.Bold).SprintFunc()

		s = "\n\nGive that a shot then press enter👆:"
		TypeItToMe(s)

		fmt.Print("\n")
		_ = terminal.PromptGetInput(terminal.PromptContent{
			// Label:      "   " + bold("▸") + "    Press " + bold("Enter") + " to continue",
			Label:      "   " + bold("▸"),
			ErrorMsg:   "error",
			AllowEmpty: true,
		})

		fmt.Print("\n")
	}
}
