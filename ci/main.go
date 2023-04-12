package main

import (
	"context"
	"dagger.io/dagger"
	"fmt"
	"os"
)

const tgruntVersion = "v0.45.2"

func googleEnv(c *dagger.Container) *dagger.Container {
	return c.WithEnvVariable("GOOGLE_APPLICATION_CREDENTIALS", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")).
		WithEnvVariable("GOOGLE_GHA_CREDS_PATH", os.Getenv("GOOGLE_GHA_CREDS_PATH"))
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
	terragrunt = googleEnv(terragrunt)
	out, err := terragrunt.WithMountedDirectory("/infra", code).
		WithWorkdir("/infra/prod").
		WithExec([]string{"run-all", "plan"}).Stdout(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
}
