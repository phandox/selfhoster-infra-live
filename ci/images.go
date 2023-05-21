package main

import (
	"ci/ci/images"
	"context"
	"dagger.io/dagger"
	"fmt"
	"path/filepath"
)

type AnsibleEnv struct {
	p *images.PythonEnv
}

func AnsibleImage(ctx context.Context, c *dagger.Client) (*dagger.Container, error) {
	sopsRelease := "https://github.com/mozilla/sops/releases/download/v3.7.3/sops-v3.7.3.linux"
	requirementsFile := c.Host().Directory(".").File("ansible/requirements.txt")
	pythonImg := images.NewPythonEnv(c,
		images.WithExternalBin(c, sopsRelease, "sops"),
		images.WithPipInstall(requirementsFile))

	container := pythonImg.C
	// TODO method for Ansible Image
	galaxyReq := c.Host().Directory(".").File("ansible/requirements.yml")
	galaxyMount := filepath.Join(pythonImg.MountPath(), "requirements.yml")
	container = container.WithMountedFile(galaxyMount, galaxyReq, dagger.ContainerWithMountedFileOpts{Owner: pythonImg.User()}).
		WithExec([]string{filepath.Join(pythonImg.BinDir(), "ansible-galaxy"), "install", "-r", galaxyMount})
	mountPath, googleCredFile, err := googleEnv(ctx, container, c.Host())
	if err != nil {
		return nil, err
	}
	container = container.WithMountedFile(mountPath, googleCredFile, dagger.ContainerWithMountedFileOpts{Owner: pythonImg.User()})
	return container, nil
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
