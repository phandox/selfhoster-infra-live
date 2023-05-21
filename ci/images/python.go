package images

import (
	"dagger.io/dagger"
	"fmt"
	"path/filepath"
)

type PythonEnv struct {
	C         *dagger.Container
	usr       string
	home      string
	mountPath string
	binPath   string
}

func (p PythonEnv) User() string {
	return p.usr
}

func (p PythonEnv) MountPath() string {
	return p.mountPath
}

// BinDir returns directory with installed Python binaries
func (p PythonEnv) BinDir() string {
	return p.binPath
}

// WithExternalBin downloads file from URL and saves in PATH under name
func WithExternalBin(dc *dagger.Client, url string, name string) func(*PythonEnv) {
	bin := dc.HTTP(url)
	return func(env *PythonEnv) {
		env.C = env.C.WithFile(fmt.Sprintf("/bin/%s", name), bin, dagger.ContainerWithFileOpts{Permissions: 0o755})
	}
}

// WithPipInstall install dependencies in requirements.txt file to env
func WithPipInstall(requirementsTXT *dagger.File) func(*PythonEnv) {
	return func(env *PythonEnv) {
		requirementsMount := filepath.Join(env.home, "mnt", "requirements.txt")
		env.C = env.C.WithMountedFile(requirementsMount, requirementsTXT, dagger.ContainerWithMountedFileOpts{Owner: env.usr}).
			WithExec([]string{filepath.Join(env.binPath, "pip"), "install", "-r", requirementsMount})
	}
}

func NewPythonEnv(c *dagger.Client, opts ...func(*PythonEnv)) *PythonEnv {
	env := PythonEnv{C: c.Container().From("python:3.10")}
	// defaults
	env.usr = "app"
	env.home = "/home/app"
	venv := filepath.Join(env.home, "venv")
	env.binPath = filepath.Join(venv, "bin")
	env.mountPath = filepath.Join(env.home, "mnt")
	env.C = env.C.WithExec([]string{"/usr/sbin/useradd", "-d", env.home, "-m", env.usr}).
		WithUser(env.usr).
		WithExec([]string{"python3", "-m", "venv", venv, "--upgrade-deps"}).
		WithExec([]string{"mkdir", filepath.Join(env.home, ".ssh")}) // TODO decouple this to Ansible image

	for _, opt := range opts {
		opt(&env)
	}
	return &env
}
