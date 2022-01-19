package ssh

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/brevdev/brev-cli/pkg/entity"
	breverrors "github.com/brevdev/brev-cli/pkg/errors"
	"github.com/brevdev/brev-cli/pkg/tasks"
	"github.com/hashicorp/go-multierror"
)

type ConfigUpdaterStore interface {
	GetContextWorkspaces() ([]entity.Workspace, error)
}

type Config interface {
	Update(workspaces []entity.Workspace) error
}

type ConfigUpdater struct {
	Store   ConfigUpdaterStore
	Configs []Config
}

var _ tasks.Task = ConfigUpdater{}

func (c ConfigUpdater) Run() error {
	workspaces, err := c.Store.GetContextWorkspaces()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	var runningWorkspaces []entity.Workspace
	for _, workspace := range workspaces {
		if workspace.Status == "RUNNING" {
			runningWorkspaces = append(runningWorkspaces, workspace)
		}
	}

	var res error
	for _, c := range c.Configs {
		err := c.Update(runningWorkspaces)
		if err != nil {
			res = multierror.Append(res, err)
		}
	}

	if res != nil {
		return breverrors.WrapAndTrace(res)
	}
	return nil
}

func (c ConfigUpdater) GetTaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{RunCronImmediately: true, Cron: "@every 3s"}
}

// SSHConfigurerV2 speciallizes in configuring ssh config with ProxyCommand
type SSHConfigurerV2 struct {
	store SSHConfigurerV2Store
}

type SSHConfigurerV2Store interface {
	WriteBrevSSHConfig(config string) error
	GetUserSSHConfig() (string, error)
	WriteUserSSHConfig(config string) error
	GetPrivateKeyPath() string
	GetUserSSHConfigPath() (string, error)
	GetBrevSSHConfigPath() (string, error)
}

var _ Config = SSHConfigurerV2{}

func NewSSHConfigurerV2(store SSHConfigurerV2Store) *SSHConfigurerV2 {
	return &SSHConfigurerV2{
		store: store,
	}
}

func (s SSHConfigurerV2) Update(workspaces []entity.Workspace) error {
	newConfig, err := s.CreateNewSSHConfig(workspaces)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	err = s.store.WriteBrevSSHConfig(newConfig)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	err = s.EnsureConfigHasInclude()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}

	return nil
}

func (s SSHConfigurerV2) CreateNewSSHConfig(workspaces []entity.Workspace) (string, error) {
	log.Print("creating new ssh config")

	configPath, err := s.store.GetUserSSHConfigPath()
	if err != nil {
		return "", breverrors.WrapAndTrace(err)
	}

	sshConfig := fmt.Sprintf("# included in %s\n", configPath)
	for _, w := range workspaces {
		// need to make getlocalidentifier conformal
		entry, err := makeSSHConfigEntry(string(w.GetLocalIdentifier(nil)), w.ID, s.store.GetPrivateKeyPath())
		if err != nil {
			return "", breverrors.WrapAndTrace(err)
		}

		sshConfig += entry
	}

	return sshConfig, nil
}

const SSHConfigEntryTemplateV2 = `Host {{ .Alias }}
  IdentityFile {{ .IdentityFile }}
  User {{ .User }}
  ProxyCommand {{ .ProxyCommand }}
  ServerAliveInterval 30

`

type SSHConfigEntryV2 struct {
	Alias        string
	IdentityFile string
	User         string
	ProxyCommand string
}

func makeSSHConfigEntry(alias string, workspaceID string, privateKeyPath string) (string, error) {
	proxyCommand := makeProxyCommand(workspaceID)
	entry := SSHConfigEntryV2{
		Alias:        alias,
		IdentityFile: privateKeyPath,
		User:         "brev",
		ProxyCommand: proxyCommand,
	}

	tmpl, err := template.New(alias).Parse(SSHConfigEntryTemplateV2)
	if err != nil {
		return "", breverrors.WrapAndTrace(err)
	}
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, entry)
	if err != nil {
		return "", breverrors.WrapAndTrace(err)
	}

	return buf.String(), nil
}

func makeProxyCommand(workspaceID string) string {
	huproxyExec := "brev proxy"
	return fmt.Sprintf("%s %s", huproxyExec, workspaceID)
}

func (s SSHConfigurerV2) EnsureConfigHasInclude() error {
	// openssh-7.3
	log.Print("ensuring has include")

	brevConfigPath, err := s.store.GetBrevSSHConfigPath()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	conf, err := s.store.GetUserSSHConfig()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	if !s.doesUserSSHConfigIncludeBrevConfig(conf, brevConfigPath) {
		newConf, err := s.AddIncludeToUserConfig(conf, brevConfigPath)
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
		err = s.store.WriteUserSSHConfig(newConf)
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
	}

	return nil
}

func (s SSHConfigurerV2) AddIncludeToUserConfig(conf string, brevConfigPath string) (string, error) {
	newConf := makeIncludeBrevStr(brevConfigPath) + conf
	return newConf, nil
}

func makeIncludeBrevStr(brevSSHConfigPath string) string {
	return fmt.Sprintf("Include %s\n", brevSSHConfigPath)
}

func (s SSHConfigurerV2) doesUserSSHConfigIncludeBrevConfig(conf string, brevConfigPath string) bool {
	return strings.Contains(conf, makeIncludeBrevStr(brevConfigPath))
}
