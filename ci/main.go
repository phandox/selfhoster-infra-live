package main

import (
	"context"
	"dagger.io/dagger"
	"fmt"
	"go.mozilla.org/sops/v3/decrypt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const tgruntVersion = "v0.45.2"

// TODO decouple mounting based on local / CI env
// TODO let caller mount with correct permissions
//func mountGoogleEnv(credFile string) (string, *dagger.File) {
//	return "/" + credFile, h.Direc
//
//}

func googleEnv(ctx context.Context, c *dagger.Container, h *dagger.Host) (*dagger.File, error) {
	// TODO use mounted Secret for credentials
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
		return h.Directory(filepath.Dir(hostCredPath)).File(credFile), nil
		//return c.WithEnvVariable("HOME", "/app").WithMountedFile("/app/.config/gcloud/application_default_credentials.json",
		//	h.Directory(filepath.Dir(hostCredPath)).File(credFile)), nil
	}
	credFile := filepath.Base(hostCredPath)
	c = c.WithEnvVariable("GOOGLE_APPLICATION_CREDENTIALS", "/root/.config/gcloud/"+credFile).WithEnvVariable("GOOGLE_CREDENTIALS", "/root/.config/gcloud/"+credFile)
	return h.Directory(filepath.Dir(hostCredPath)).File(credFile), nil
}

type daggerSecrets struct {
	DoToken       *dagger.Secret
	SSHPrivateKey *dagger.Secret
}

func sopsDecrypt(cryptText string, c *dagger.Client) (daggerSecrets, error) {
	type secrets struct {
		DoToken       string `yaml:"do_token"`
		SSHPrivateKey string `yaml:"ssh_private_key"`
	}
	clearText, err := decrypt.Data([]byte(cryptText), "yaml")
	if err != nil {
		return daggerSecrets{}, fmt.Errorf("problem decrypting SOPS data: %w", err)
	}
	s := secrets{}
	if err = yaml.Unmarshal(clearText, &s); err != nil {
		return daggerSecrets{}, fmt.Errorf("problem unmarshalling data: %w", err)
	}
	var ds daggerSecrets
	ds.DoToken = c.SetSecret("do_token", s.DoToken)
	ds.SSHPrivateKey = c.SetSecret("private_key", s.SSHPrivateKey)
	return ds, nil
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
	googleCredFile, err := googleEnv(ctx, terragrunt, c.Host())
	if err != nil {
		return nil, err
	}
	terragrunt = terragrunt.WithMountedFile("/root/.config/gcloud/application_default_credentials.json", googleCredFile)
	return terragrunt.WithMountedDirectory("/infra", code).
		WithWorkdir("/infra/dev").
		WithSecretVariable("TF_VAR_do_token", s.DoToken), nil

}

func ansibleImage(ctx context.Context, c *dagger.Client) (*dagger.Container, error) {
	sopsRelease := "https://github.com/mozilla/sops/releases/download/v3.7.3/sops-v3.7.3.linux"
	sopsBin := c.HTTP(sopsRelease)

	code := c.Host().Directory(".")
	ansible := c.Container().From("python:3.10").WithExec([]string{"/usr/sbin/useradd", "-d", "/app", "-m", "app"}).WithUser("app").WithExec([]string{"python3", "-m", "venv", "/app/.venv", "--upgrade-deps"})
	ansible = ansible.WithFile("/bin/sops", sopsBin, dagger.ContainerWithFileOpts{Permissions: 0o755})
	ansible = ansible.WithMountedDirectory("/mnt/app/code", code).WithWorkdir("/mnt/app/code/ansible").
		WithExec([]string{"/app/.venv/bin/pip3", "install", "-r", "requirements.txt"}).
		WithExec([]string{"/app/.venv/bin/ansible-galaxy", "install", "-r", "requirements.yml"}).
		WithExec([]string{"mkdir", "/app/.ssh"})
	googleCredFile, err := googleEnv(ctx, ansible, c.Host())
	if err != nil {
		return nil, err
	}
	ansible = ansible.WithMountedFile("/app/.config/gcloud/application_default_credentials.json", googleCredFile, dagger.ContainerWithMountedFileOpts{Owner: "app:app"})
	return ansible, nil
}

func main() {
	action := os.Args[1]

	ctx := context.Background()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		panic(err)
	}
	defer client.Close()
	code := client.Host().Directory(".")
	cryptFile, err := code.File("secrets.yaml").Contents(ctx)
	if err != nil {
		panic(err)
	}
	s, err := sopsDecrypt(cryptFile, client)
	if err != nil {
		panic(err)
	}

	tgruntExec, err := terragruntImage(ctx, client)
	if err != nil {
		panic(err)
	}

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
		ansibleExec, _ := ansibleImage(ctx, client)
		ansibleExec = ansibleExec.WithEnvVariable("ANSIBLE_HOST_KEY_CHECKING", "False").
			WithSecretVariable("TF_VAR_do_token", s.DoToken).
			WithMountedSecret("/app/.ssh/id_ed25519", s.SSHPrivateKey, dagger.ContainerWithMountedSecretOpts{Owner: "app:app"}).
			WithEnvVariable("ANSIBLE_PRIVATE_KEY_FILE", "/app/.ssh/id_ed25519")
		_, err = ansibleExec.Export(ctx, "/tmp/ansible-debug.tgz")
		if err != nil {
			panic(err)
		}
		ansibleExec.WithExec([]string{"/app/.venv/bin/ansible-playbook", "-i", "../dev/postgres-vm/do_hosts.yml", "--extra-vars", "exec_env=dev", "db.yml"}).Stdout(ctx)
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
