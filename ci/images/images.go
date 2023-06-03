package images

import (
	"context"
	"dagger.io/dagger"
	"fmt"
	"path/filepath"
)

type ContainerImage struct {
	*dagger.Container
	usr       string
	home      string
	mountPath string
	binPath   string
}

func (c ContainerImage) Home() string {
	return c.home
}

func (c ContainerImage) User() string {
	return c.usr
}

func (c ContainerImage) MountPath() string {
	return c.mountPath
}

// WithExternalBin downloads file from URL and saves in PATH under name
func WithExternalBin(dc *dagger.Client, url string, name string) ContainerOption {
	bin := dc.HTTP(url)
	return func(ci *ContainerImage) error {
		ci.Container = ci.Container.WithFile(filepath.Join(ci.binPath, name), bin, dagger.ContainerWithFileOpts{Permissions: 0o755})
		return nil
	}
}

func WithUnprivilegedUser(ctx context.Context, usrAdder func(*dagger.Container) *dagger.Container) ContainerOption {
	return func(ci *ContainerImage) error {
		origEntrypoint, err := ci.Entrypoint(ctx)
		if err != nil {
			return err
		}
		origPath, err := ci.EnvVariable(ctx, "PATH")
		if err != nil {
			return err
		}
		ci.Container = ci.Container.WithEntrypoint([]string{"/bin/sh"}).
			With(usrAdder).
			WithUser(ci.User()).
			WithExec([]string{"-c", fmt.Sprintf("mkdir -p %s %s", ci.binPath, ci.mountPath)}).
			WithEnvVariable("PATH", fmt.Sprintf("%s:%s", origPath, ci.binPath)).
			WithEntrypoint(origEntrypoint)
		return nil
	}
}

func WithGCPAuthGen(ctx context.Context, h *dagger.Host) ContainerOption {
	return func(c *ContainerImage) error {
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
		c.Container = c.Container.
			WithEnvVariable("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(c.Home(), adcPath)).
			WithEnvVariable("GOOGLE_CREDENTIALS", filepath.Join(c.Home(), adcPath))
		c.Container = c.Container.WithMountedFile(filepath.Join(c.Home(), adcPath), h.Directory(filepath.Dir(hostCredPath)).File(credFile), dagger.ContainerWithMountedFileOpts{Owner: c.User()})
		return nil
	}
}

type ContainerOption func(ci *ContainerImage) error

func NewContainerImage(c *dagger.Container, opts ...ContainerOption) (*ContainerImage, error) {
	img := &ContainerImage{Container: c, usr: "app", home: "/home/app"}
	img.binPath = filepath.Join(img.Home(), "bin")
	img.mountPath = filepath.Join(img.Home(), "mnt")

	for _, opt := range opts {
		if err := opt(img); err != nil {
			return nil, fmt.Errorf("error applying the option: %w", err)
		}
	}
	return img, nil
}
