package images

import (
	"dagger.io/dagger"
	"fmt"
	"path/filepath"
)

type Ansible struct {
	*PythonEnv
	sshDir string
}

type AnsibleOption func(*Ansible) error

func WithSSH(privateKey *dagger.Secret) func(*Ansible) error {
	return func(a *Ansible) error {
		mountedKey := filepath.Join(a.sshDir, "id_ed25519")
		a.Container = a.Container.WithEnvVariable("ANSIBLE_HOST_KEY_CHECKING", "False")
		a.Container = a.Container.WithMountedSecret(mountedKey, privateKey, dagger.ContainerWithMountedSecretOpts{Owner: a.User()}).
			WithEnvVariable("ANSIBLE_PRIVATE_KEY_FILE", mountedKey)
		return nil
	}
}

func WithGalaxyInstall(requirementsYML *dagger.File) func(*Ansible) error {
	return func(a *Ansible) error {
		galaxyMount := filepath.Join(a.MountPath(), "requirements.yml")
		a.Container = a.Container.WithMountedFile(galaxyMount, requirementsYML, dagger.ContainerWithMountedFileOpts{Owner: a.User()}).
			WithExec([]string{filepath.Join(a.BinDir(), "ansible-galaxy"), "install", "-r", galaxyMount})
		return nil
	}
}

func NewAnsible(p *PythonEnv, opts ...AnsibleOption) (*Ansible, error) {
	a := &Ansible{p, ""}
	a.sshDir = filepath.Join(a.home, "ssh")
	a.Container = a.Container.WithExec([]string{"mkdir", filepath.Join(a.home, ".ssh")})

	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, fmt.Errorf("error creating ansible image: %w", err)
		}
	}
	return a, nil
}
