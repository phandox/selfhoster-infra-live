package images

import (
	"context"
	"dagger.io/dagger"
	"fmt"
	"path/filepath"
)

type Terragrunt struct {
	*ContainerImage
}

type TerragruntOption func(t *Terragrunt) error

func NewTerragrunt(ctx context.Context, c *dagger.Client, opts ...TerragruntOption) (*Terragrunt, error) {
	base, err := NewContainerImage(c.Container().From("hashicorp/terraform:1.3.9"))
	if err != nil {
		return nil, err
	}
	err = WithUnprivilegedUser(ctx, func(d *dagger.Container) *dagger.Container {
		return d.WithExec([]string{"-c", fmt.Sprintf("/usr/sbin/adduser -h %s -D %s", base.Home(), base.User())})
	})(base)
	if err != nil {
		return nil, err
	}
	tg := &Terragrunt{base}

	for _, opt := range opts {
		if err := opt(tg); err != nil {
			return nil, fmt.Errorf("error applying the Terragrunt option: %w", err)
		}
	}
	return tg, nil
}

func WithTerragrunt(dc *dagger.Client, version string) TerragruntOption {
	tgruntRelease := fmt.Sprintf("https://github.com/gruntwork-io/terragrunt/releases/download/%s/terragrunt_linux_amd64", version)
	return func(t *Terragrunt) error {
		if err := WithExternalBin(dc, tgruntRelease, "terragrunt")(t.ContainerImage); err != nil {
			return err
		}
		t.Container = t.Container.WithEntrypoint([]string{filepath.Join(t.binPath, "terragrunt")})
		return nil
	}
}
