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
	//credDir := filepath.Dir(hostCredPath)
	credFile := filepath.Base(hostCredPath)
	debug, _ := h.Directory(".").File(credFile).Contents(ctx)
	fmt.Fprintf(os.Stderr, debug)
	return c.WithMountedFile("/"+credFile, h.Directory(".").File(credFile)).
		WithEnvVariable("GOOGLE_APPLICATION_CREDENTIALS", "/"+credFile).
		WithEnvVariable("GOOGLE_GHA_CREDS_PATH", "/"+credFile), nil
}

func main() {
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
	out, err := terragrunt.WithMountedDirectory("/infra", code).
		WithWorkdir("/infra/prod").
		WithExec([]string{"run-all", "init"}).Stdout(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
}
