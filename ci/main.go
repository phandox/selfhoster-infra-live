package main

import (
	"context"
	"dagger.io/dagger"
	"fmt"
	"os"
	"path/filepath"
)

const tgruntVersion = "v0.45.2"

func googleEnv(ctx context.Context, c *dagger.Container, h *dagger.Host) (*dagger.Container, error) {
	hostCredPath, err := h.EnvVariable("GOOGLE_APPLICATION_CREDENTIALS").Value(ctx)
	if err != nil {
		return nil, err
	}
	if hostCredPath == "" {
		hostHome, err := h.EnvVariable("HOME").Value(ctx)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch Google Cloud credentials: %w", err)
		}
		hostCredPath = filepath.Join(hostHome, ".config/gcloud/application_default_credentials.json")
		credFile := filepath.Base(hostCredPath)
		return c.WithEnvVariable("HOME", "/app").WithMountedFile("/app/.config/gcloud/application_default_credentials.json",
			h.Directory(filepath.Dir(hostCredPath)).File(credFile)), nil
	}
	credFile := filepath.Base(hostCredPath)
	return c.WithMountedFile("/"+credFile, h.Directory(filepath.Dir(hostCredPath)).File(credFile)).
		WithEnvVariable("GOOGLE_APPLICATION_CREDENTIALS", "/"+credFile).
		WithEnvVariable("GOOGLE_CREDENTIALS", "/"+credFile), nil
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

	terragrunt := client.Container().
		From("hashicorp/terraform:1.3.9").
		WithFile("/bin/terragrunt", tgruntBinary, dagger.ContainerWithFileOpts{Permissions: 0755}).
		WithEntrypoint([]string{"/bin/terragrunt"})
	terragrunt, err = googleEnv(ctx, terragrunt, client.Host())
	if err != nil {
		panic(err)
	}

	tgruntExec := terragrunt.WithMountedDirectory("/infra", code).
		WithWorkdir("/infra/prod")

	switch action {
	case "plan":
		plan, err := tgruntExec.WithExec([]string{"run-all", "plan", "--terragrunt-non-interactive"}).Stdout(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Println(plan)
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
		panic("Unknown action. 'plan', 'apply' and 'destroy' are supported")
	}
}
