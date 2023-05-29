package main

import (
	"ci/ci/images"
	"context"
	"dagger.io/dagger"
	"fmt"
)

func AnsibleImage(ctx context.Context, c *dagger.Client, s daggerSecrets) (*images.Ansible, error) {
	sopsRelease := "https://github.com/mozilla/sops/releases/download/v3.7.3/sops-v3.7.3.linux"
	requirementsFile := c.Host().Directory(".").File("ansible/requirements.txt")

	pythonImg, err := images.NewPythonEnv(c, images.WithPipInstall(requirementsFile))
	if err != nil {
		return nil, err
	}
	if err := images.WithExternalBin(c, sopsRelease, "sops")(pythonImg.ContainerImage); err != nil {
		return nil, err
	}
	galaxyReq := c.Host().Directory(".").File("ansible/requirements.yml")
	a, err := images.NewAnsible(pythonImg,
		images.WithGalaxyInstall(galaxyReq),
		images.WithSSH(s.SSHPrivateKey))
	if err != nil {
		return nil, err
	}
	if err := images.WithGCPAuthGen(ctx, c.Host())(a.ContainerImage); err != nil {
		return nil, err
	}
	a.Container = a.Container.WithSecretVariable("TF_VAR_do_token", s.DoToken)
	return a, nil
}

func genTerragrunt(ctx context.Context, c *dagger.Client) (*images.Terragrunt, error) {
	tg, err := images.NewTerragrunt(c, images.WithTerragrunt(c, tgruntVersion))
	if err != nil {
		return nil, err
	}
	if err = images.WithGCPAuthGen(ctx, c.Host())(tg.ContainerImage); err != nil {
		return nil, err
	}
	return nil, nil
}

func terragruntImage(ctx context.Context, c *dagger.Client) (*dagger.Container, error) {
	tgruntRelease := fmt.Sprintf("https://github.com/gruntwork-io/terragrunt/releases/download/%s/terragrunt_linux_amd64", tgruntVersion)

	tgruntBinary := c.HTTP(tgruntRelease)

	code := c.Host().Directory(".")
	cryptFile, err := code.File("secrets.yaml").Contents(ctx)
	if err != nil {
		return nil, err
	}

	s, err := sopsDecrypt(cryptFile, c)
	if err != nil {
		return nil, err
	}

	terragrunt := c.Container().
		From("hashicorp/terraform:1.3.9").
		WithFile("/bin/terragrunt", tgruntBinary, dagger.ContainerWithFileOpts{Permissions: 0755}).
		WithEntrypoint([]string{"/bin/terragrunt"})
	mountPath, googleCredFile, err := googleEnv(ctx, terragrunt, c.Host())
	if err != nil {
		return nil, err
	}
	terragrunt = terragrunt.WithMountedFile(mountPath, googleCredFile)
	return terragrunt.WithMountedDirectory("/infra", code).
		WithWorkdir("/infra/dev").
		WithSecretVariable("TF_VAR_do_token", s.DoToken), nil
}
