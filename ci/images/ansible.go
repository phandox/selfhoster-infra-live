package images

import (
	"context"
	"dagger.io/dagger"
	"fmt"
	"path/filepath"
)

type Ansible struct {
	*PythonEnv
	sshDir string
}

type AnsibleOption func(*Ansible) error

func WithGCPAuth(ctx context.Context, h *dagger.Host) func(*Ansible) error {
	return func(a *Ansible) error {
		const adcPath = ".config/gcloud/application_default_credentials.json"
		hostCredPath, err := h.EnvVariable("GOOGLE_APPLICATION_CREDENTIALS").Value(ctx)
		if err != nil {
			return err
		}
		credSource := func(detectedAppCred string) (string, error) {
			if len(detectedAppCred) != 0 {
				return detectedAppCred, nil
			}
			var credPath string
			hostHome, err := h.EnvVariable("HOME").Value(ctx)
			if err != nil {
				return "", err
			}
			credPath = filepath.Join(hostHome, adcPath)
			return credPath, nil
		}

		hostCredPath, err = credSource(hostCredPath)
		if err != nil {
			return err
		}
		credFile := filepath.Base(hostCredPath)
		a.C = a.C.
			WithEnvVariable("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(a.Home(), adcPath)).
			WithEnvVariable("GOOGLE_CREDENTIALS", filepath.Join(a.Home(), adcPath))
		a.C = a.C.WithMountedFile(filepath.Join(a.Home(), adcPath), h.Directory(filepath.Dir(hostCredPath)).File(credFile), dagger.ContainerWithMountedFileOpts{Owner: a.User()})
		return nil
	}
}

func WithSSH(privateKey *dagger.Secret) func(*Ansible) error {
	return func(a *Ansible) error {
		mountedKey := filepath.Join(a.sshDir, "id_ed25519")
		a.C = a.C.WithEnvVariable("ANSIBLE_HOST_KEY_CHECKING", "False")
		a.C = a.C.WithMountedSecret(mountedKey, privateKey, dagger.ContainerWithMountedSecretOpts{Owner: a.User()}).
			WithEnvVariable("ANSIBLE_PRIVATE_KEY_FILE", mountedKey)
		return nil
	}
}

func WithGalaxyInstall(requirementsYML *dagger.File) func(*Ansible) error {
	return func(a *Ansible) error {
		galaxyMount := filepath.Join(a.MountPath(), "requirements.yml")
		a.C = a.C.WithMountedFile(galaxyMount, requirementsYML, dagger.ContainerWithMountedFileOpts{Owner: a.User()}).
			WithExec([]string{filepath.Join(a.BinDir(), "ansible-galaxy"), "install", "-r", galaxyMount})
		return nil
	}
}

func NewAnsible(p *PythonEnv, opts ...AnsibleOption) (*Ansible, error) {
	a := &Ansible{p, ""}
	a.sshDir = filepath.Join(a.home, "ssh")
	a.C = a.C.WithExec([]string{"mkdir", filepath.Join(a.home, ".ssh")})

	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, fmt.Errorf("error creating ansible image: %w", err)
		}
	}
	return a, nil
}
