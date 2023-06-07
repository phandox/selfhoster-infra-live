package images

import (
	"context"
	"dagger.io/dagger"
	"fmt"
	"path/filepath"
	"time"
)

type Helm struct {
	*ContainerImage
}

type HelmRepo struct {
	Name string
	Url  string
}

func WithRepository(repositories ...HelmRepo) HelmOption {
	return func(h *Helm) error {
		for _, r := range repositories {
			h.Container = h.Container.WithExec([]string{"helm", "repo", "add", r.Name, r.Url})
		}
		h.Container = h.Container.WithExec([]string{"helm", "repo", "update"})
		return nil
	}
}

func WithK8SCluster(dc *dagger.Client, name string, doToken *dagger.Secret) HelmOption {
	return func(h *Helm) error {
		doctl := dc.Container().From("digitalocean/doctl:1-latest").File("/app/doctl")
		kubectl := dc.Container().From("bitnami/kubectl:1.27").File("/opt/bitnami/kubectl/bin/kubectl")
		h.Container = h.Container.
			WithFile(filepath.Join(h.binPath, "kubectl"), kubectl, dagger.ContainerWithFileOpts{Owner: h.User(), Permissions: 0o755}).
			WithFile(filepath.Join(h.binPath, "doctl"), doctl, dagger.ContainerWithFileOpts{Owner: h.User(), Permissions: 0o755}).
			WithFile(filepath.Join(h.Home(), ".kube", "doctl"), doctl, dagger.ContainerWithFileOpts{Owner: h.User(), Permissions: 0o755})
		cred := dc.Container().
			From("digitalocean/doctl:1-latest").
			WithSecretVariable("DIGITALOCEAN_ACCESS_TOKEN", doToken).
			WithEnvVariable("BUST", time.Now().String()).
			WithExec([]string{"kubernetes", "cluster", "kubeconfig", "save", name})
		h.Container = h.Container.WithSecretVariable("DIGITALOCEAN_ACCESS_TOKEN", doToken).
			WithMountedFile(filepath.Join(h.Home(), ".kube", "config"), cred.File("/root/.kube/config"), dagger.ContainerWithMountedFileOpts{Owner: h.User()})
		return nil
	}
}

type HelmOption func(*Helm) error

func withHelm(dc *dagger.Client, helmVer string, secretsVer string, sopsVer string) HelmOption {
	sopsRelease := fmt.Sprintf("https://github.com/mozilla/sops/releases/download/%s/sops-%s.linux", sopsVer, sopsVer)
	helmRelease := fmt.Sprintf("https://get.helm.sh/helm-%s-linux-amd64.tar.gz", helmVer)
	helmSecrets := "https://github.com/jkroepke/helm-secrets"
	return func(h *Helm) error {
		var err error
		if err = WithExternalBin(dc, helmRelease, "helm.tgz")(h.ContainerImage); err != nil {
			return nil
		}
		h.Container = h.Container.WithExec([]string{"tar", "-xzf", filepath.Join(h.binPath, "helm.tgz"), "-C", h.Home()}).
			WithExec([]string{"mv", filepath.Join(h.Home(), "linux-amd64", "helm"), filepath.Join(h.binPath, "helm")}).
			WithExec([]string{"rm", "-f", filepath.Join(h.binPath, "helm.tgz")})
		h.Container = h.Container.WithExec([]string{filepath.Join(h.binPath, "helm"), "plugin", "install", helmSecrets, "--version", secretsVer})
		if err = WithExternalBin(dc, sopsRelease, "sops")(h.ContainerImage); err != nil {
			return nil
		}
		h.Container = h.Container.WithEnvVariable("HELM_SECRETS_SOPS_PATH", filepath.Join(h.binPath, "sops"))
		return nil
	}
}

func NewHelm(ctx context.Context, c *dagger.Client, opts ...HelmOption) (*Helm, error) {
	base, err := NewContainerImage(c.Container().From("ubuntu:22.04").WithExec([]string{"apt-get", "update"}).WithExec([]string{"apt-get", "install", "-y", "git"}))
	if err != nil {
		return nil, err
	}
	err = WithUnprivilegedUser(ctx, func(dc *dagger.Container) *dagger.Container {
		return dc.WithExec([]string{"-c", fmt.Sprintf("/usr/sbin/useradd -d %s -m %s", base.Home(), base.User())})
	})(base)
	if err != nil {
		return nil, err
	}

	h := &Helm{base}
	err = withHelm(c, "v3.12.0", "v4.4.2", "v3.7.3")(h)
	for _, opt := range opts {
		if err := opt(h); err != nil {
			return nil, err
		}
	}
	h.Container = h.Container.WithEntrypoint([]string{filepath.Join(h.binPath, "helm")})
	return h, nil
}
