package main

import (
	"ci/images"
	"context"
	"dagger.io/dagger"
	"fmt"
	"path/filepath"
)

func AnsibleImage(ctx context.Context, c *dagger.Client, s daggerSecrets) (*images.Ansible, error) {
	sopsRelease := "https://github.com/mozilla/sops/releases/download/v3.7.3/sops-v3.7.3.linux"
	requirementsFile := c.Host().Directory(".").File("ansible/requirements.txt")

	pythonImg, err := images.NewPythonEnv(ctx, c, images.WithPipInstall(requirementsFile))
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

func TerragruntImage(ctx context.Context, c *dagger.Client, s daggerSecrets, env string) (*images.Terragrunt, error) {
	tg, err := images.NewTerragrunt(ctx, c, images.WithTerragrunt(c, tgruntVersion))
	if err != nil {
		return nil, err
	}
	if err = images.WithGCPAuthGen(ctx, c.Host())(tg.ContainerImage); err != nil {
		return nil, err
	}
	tgCode := c.Host().Directory(".") // Needed because root terragrunt.hcl is in top-level
	tg.Container = tg.Container.
		WithMountedDirectory(filepath.Join(tg.MountPath(), "infra"), tgCode, dagger.ContainerWithMountedDirectoryOpts{Owner: tg.User()}).
		WithWorkdir(filepath.Join(tg.MountPath(), "infra", env)).WithSecretVariable("TF_VAR_do_token", s.DoToken)
	return tg, nil
}

func HelmImage(ctx context.Context, c *dagger.Client, s daggerSecrets) (*images.Helm, error) {
	h, err := images.NewHelm(ctx, c,
		images.WithK8SCluster(c, "doks-fra1-001", s.DoToken),
		images.WithRepository(
			images.HelmRepo{Name: "ingress-nginx", Url: "https://kubernetes.github.io/ingress-nginx"},
			images.HelmRepo{Name: "external-dns", Url: "https://kubernetes-sigs.github.io/external-dns/"},
		))
	if err = images.WithGCPAuthGen(ctx, c.Host())(h.ContainerImage); err != nil {
		return nil, fmt.Errorf("helm: error with GCP auth: %w", err)
	}
	charts := c.Host().Directory("charts")
	h.Container = h.Container.
		WithMountedDirectory(filepath.Join(h.MountPath(), "charts"), charts, dagger.ContainerWithMountedDirectoryOpts{Owner: h.User()}).
		WithWorkdir(filepath.Join(h.MountPath(), "charts"))
	return h, nil
}
