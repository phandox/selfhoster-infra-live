package main

import (
	"context"
	"dagger.io/dagger"
	"fmt"
	"github.com/pkg/errors"
	"go.mozilla.org/sops/v3/decrypt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const tgruntVersion = "v0.45.2"

func googleEnv(ctx context.Context, c *dagger.Container, h *dagger.Host) (*dagger.Container, error) {
	hostCredPath, err := h.EnvVariable("GOOGLE_APPLICATION_CREDENTIALS").Value(ctx)
	if err != nil {
		return nil, err
	}
	credFile := filepath.Base(hostCredPath)
	return c.WithMountedFile("/"+credFile, h.Directory(".").File(credFile)).
		WithEnvVariable("GOOGLE_APPLICATION_CREDENTIALS", "/"+credFile).
		WithEnvVariable("GOOGLE_GHA_CREDS_PATH", "/"+credFile), nil
}

type secrets struct {
	DoToken string `yaml:"do_token"`
}

func sopsDecrypt(cryptText string) (secrets, error) {
	clearText, err := decrypt.Data([]byte(cryptText), "yaml")
	if err != nil {
		return secrets{}, errors.Wrap(err, "problem decrypting SOPS data")
	}
	s := secrets{}
	if err = yaml.Unmarshal(clearText, &s); err != nil {
		return secrets{}, errors.Wrap(err, "problem unmarshalling data")
	}
	return s, nil
}

func main() {
	action := os.Args[1]

	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	tgruntRelease := fmt.Sprintf("https://github.com/gruntwork-io/terragrunt/releases/download/%s/terragrunt_linux_amd64", tgruntVersion)

	tgruntBinary := client.HTTP(tgruntRelease)

	code := client.Host().Directory(".")
	cryptFile, err := client.Host().Directory(".").File("prod/secrets.yaml").Contents(ctx)
	if err != nil {
		panic(err)
	}

	s, err := sopsDecrypt(cryptFile)
	if err != nil {
		panic(err)
	}

	terragrunt := client.Container().
		From("hashicorp/terraform:1.3.9").
		WithFile("/bin/terragrunt", tgruntBinary, dagger.ContainerWithFileOpts{Permissions: 0755}).
		WithEntrypoint([]string{"/bin/terragrunt"})
	terragrunt, err = googleEnv(ctx, terragrunt, client.Host())
	if err != nil {
		panic(err)
	}

	tgruntExec := terragrunt.WithMountedDirectory("/infra", code).
		WithWorkdir("/infra/prod").
		WithEnvVariable("TF_VAR_do_token", s.DoToken)

	switch action {
	case "apply":
		apply, err := tgruntExec.WithExec([]string{"run-all", "apply", "--terragrunt-non-interactive"}).Stdout(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Println(apply)
	case "destroy":
		destroy, err := tgruntExec.WithExec([]string{"run-all", "destroy", "--terragrunt-non-interactive", "--terragrunt-exclude-dir", "volumes/"}).Stdout(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Println(destroy)
	default:
		panic("Unknown action. 'apply' and 'destroy' are supported")
	}
}
