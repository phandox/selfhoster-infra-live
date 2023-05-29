package images

import (
	"dagger.io/dagger"
	"fmt"
)

type Terragrunt struct {
	*ContainerImage
}

type TerragruntOption func(t *Terragrunt) error

func NewTerragrunt(c *dagger.Client, opts ...TerragruntOption) (*Terragrunt, error) {
	base, err := NewContainerImage(c.Container().From("hashicorp/terraform:1.3.9"))
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
		WithExternalBin(dc, tgruntRelease, "terragrunt")
		return nil
	}
}
