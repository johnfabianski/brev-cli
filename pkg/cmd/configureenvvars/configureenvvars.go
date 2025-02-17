package configureenvvars

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/brevdev/brev-cli/pkg/collections" //nolint:typecheck
	"github.com/brevdev/brev-cli/pkg/terminal"
	"github.com/spf13/cobra"
)

const (
	BREV_WORKSPACE_ENV_PATH  = "/home/brev/workspace/.env"
	BREV_DEV_PLANE_ENV_PATH  = "/home/ubuntu/.brev/.env"
	BREV_MANGED_ENV_VARS_KEY = "BREV_MANAGED_ENV_VARS"
)

type envVars map[string]string

type ConfigureEnvVarsStore interface {
	GetFileAsString(path string) (string, error)
}

func NewCmdConfigureEnvVars(_ *terminal.Terminal, cevStore ConfigureEnvVarsStore) *cobra.Command {
	cmd := &cobra.Command{
		Annotations:           map[string]string{"housekeeping": ""},
		Use:                   "configure-env-vars",
		DisableFlagsInUseLine: true,
		Short:                 "configure env vars in supported shells",
		Long:                  "configure env vars in supported shells",
		Example:               "",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := RunConfigureEnvVars(cevStore)
			if err != nil {
				// todo bubble up error, but in the meantime make sure there
				// is no output
				return nil
			}
			fmt.Print(output)
			return nil
		},
	}

	return cmd
}

func RunConfigureEnvVars(cevStore ConfigureEnvVarsStore) (string, error) {
	brevEnvsString := os.Getenv(BREV_MANGED_ENV_VARS_KEY)
	// intentionally ignoring err
	envFileContents, _ := cevStore.GetFileAsString(BREV_WORKSPACE_ENV_PATH)
	devplaneContents, _ := cevStore.GetFileAsString(BREV_DEV_PLANE_ENV_PATH)
	envFileContents = envFileContents + "\n" + devplaneContents
	return generateExportString(brevEnvsString, envFileContents), nil
}

func generateExportString(brevEnvsString, envFileContents string) string {
	if brevEnvsString == "" && envFileContents == "" {
		return ""
	}
	brevEnvKeys := strings.Split(brevEnvsString, ",")

	envfileEntries := parse(envFileContents)
	for key, val := range envfileEntries {
		if !strings.HasPrefix(val, "'") { // already quoted
			envfileEntries[key] = shellescape.Quote(val)
		}
	}
	envFileKeys := keys(envfileEntries)
	// sort to make tests consistent
	sort.Slice(envFileKeys, func(i, j int) bool {
		return envFileKeys[i] < envFileKeys[j]
	})

	// todo parameterize by shell
	envCmdOutput := makeEnvCmdOutputLines(brevEnvKeys, envFileKeys, envfileEntries)

	return strings.Join(envCmdOutput, "\n")
}

func makeEnvCmdOutputLines(brevEnvKeys, envFileKeys []string, envfileEntries envVars) []string {
	envCmdOutput := []string{}
	envCmdOutput = addUnsetEntriesToOutput(brevEnvKeys, envFileKeys, envCmdOutput)
	envCmdOutput = append(envCmdOutput, addExportPrefix(envfileEntries)...)
	newBrevEnvKeys := strings.Join(envFileKeys, ",")
	newBrevEnvKeysEntry := ""
	if newBrevEnvKeys != "" {
		newBrevEnvKeysEntry = BREV_MANGED_ENV_VARS_KEY + "=" + newBrevEnvKeys
	}
	if newBrevEnvKeysEntry != "" {
		envCmdOutput = append(envCmdOutput, "export "+newBrevEnvKeysEntry)
	}
	return collections.FilterEmpty(envCmdOutput)
}

func addExportPrefix(envFile envVars) []string {
	if len(envFile) == 0 {
		return []string{}
	}
	out := []string{}

	// sorted order to make tests consistent
	envFileKeys := keys(envFile)
	for _, k := range envFileKeys {
		out = append(out, fmt.Sprintf("%s %s=%s", "export", k, envFile[k]))
	}
	return out
}

// return map's keys in sorted order
func keys(m map[string]string) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

// this may be a good place to parameterize bby shell
func addUnsetEntriesToOutput(currentEnvs, newEnvs, output []string) []string {
	for _, envKey := range currentEnvs {
		if !collections.Contains(newEnvs, envKey) && envKey != "" {
			output = append(output, "unset "+envKey)
		}
	}
	return output
}

// https://stackoverflow.com/a/38579502
func zip(elements []string, elementMap map[string]string) map[string]string {
	for i := 0; i < len(elements); i += 2 {
		elementMap[elements[i]] = elements[i+1]
	}
	return elementMap
}

func parse(content string) envVars {
	keyValPairs := []string{}
	lexer := lex("keys from env", content)
	scanning := true
	for scanning {
		token := lexer.nextItem()
		switch token.typ {
		case itemKey, itemValue:
			keyValPairs = append(keyValPairs, token.val)
		case itemError:
			return nil
		case itemEOF:
			scanning = false

		}

	}
	if len(keyValPairs)%2 != 0 {
		return nil
	}

	return zip(keyValPairs, make(envVars))
}
