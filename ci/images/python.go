package images

import (
	"dagger.io/dagger"
	"fmt"
	"path/filepath"
)

type PythonEnv struct {
	*ContainerImage
	venv string
}

type PythonOption func(*PythonEnv) error

// BinDir returns directory with installed Python binaries
func (p PythonEnv) BinDir() string {
	return filepath.Join(p.venv, "bin")
}

// WithPipInstall install dependencies in requirements.txt file to env
func WithPipInstall(requirementsTXT *dagger.File) PythonOption {
	return func(env *PythonEnv) error {
		requirementsMount := filepath.Join(env.home, "mnt", "requirements.txt")
		env.Container = env.Container.WithMountedFile(requirementsMount, requirementsTXT, dagger.ContainerWithMountedFileOpts{Owner: env.usr}).
			WithExec([]string{filepath.Join(env.BinDir(), "pip"), "install", "-r", requirementsMount})
		return nil
	}
}

func NewPythonEnv(c *dagger.Client, opts ...PythonOption) (*PythonEnv, error) {
	base, err := NewContainerImage(c.Container().From("python:3.10"))
	if err != nil {
		return nil, err
	}
	p := PythonEnv{base, filepath.Join(base.Home(), "venv")}
	p.Container = p.Container.WithExec([]string{"python3", "-m", "venv", p.venv, "--upgrade-deps"})

	for _, opt := range opts {
		if err := opt(&p); err != nil {
			return nil, fmt.Errorf("error applying Python option: %w", err)
		}
	}
	return &p, nil
}
