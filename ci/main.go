package main

import (
	"context"
	"dagger.io/dagger"
	"flag"
	"fmt"
	"go.mozilla.org/sops/v3/decrypt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const tgruntVersion = "v0.45.2"

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

type ciArgs struct {
	action string
	env    string
}

func (cia ciArgs) validate() error {
	if len(cia.env) == 0 {
		return fmt.Errorf("-env flag must not be empty")
	}
	if len(cia.action) == 0 {
		return fmt.Errorf("-action flag must not be empty")
	}
	if cia.env != "dev" && cia.env != "prod" {
		return fmt.Errorf("invalid env: %q not in %q or %q", cia.env, "dev", "prod")
	}
	if cia.action != "plan" && cia.action != "apply" && cia.action != "destroy" && cia.action != "helm-platform" {
		return fmt.Errorf("invalid action: %q not in %q or %q or %q", cia.env, "plan", "apply", "destroy")
	}
	return nil
}

func cliFlags(args []string) (*ciArgs, error) {
	fs := flag.NewFlagSet("dagger-ci", flag.ExitOnError)
	cfg := ciArgs{}
	fs.StringVar(&cfg.env, "env", "dev", "environment for executing pipeline")
	fs.StringVar(&cfg.action, "action", "", "action for pipeline - plan / apply / destroy")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}
	err = cfg.validate()
	if err != nil {
		return nil, fmt.Errorf("error validating flags: %w", err)
	}
	return &cfg, nil
}

func main() {
	cfg, err := cliFlags(os.Args[1:])
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	// Preparing Dagger client (with Secrets, with Host code as functional options?)
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

	// Building images
	// Build of Tgrunt image
	tgruntExec, err := TerragruntImage(ctx, client, s, cfg.env)
	if err != nil {
		panic(err)
	}
	// build of Ansible image
	ansibleExec, err := AnsibleImage(ctx, client, s)
	if err != nil {
		panic(err)
	}
	helmExec, err := HelmImage(ctx, client, s)
	if err != nil {
		panic(err)
	}

	// Action dispatcher / deploy
	switch cfg.action {
	case "plan":
		plan, err := tgruntExec.WithExec([]string{"run-all", "plan", "--terragrunt-non-interactive"}).Stdout(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Println(plan)
	case "apply":
		// Run Terragrunt phase
		tgruntExec.WithExec([]string{"run-all", "apply", "--terragrunt-non-interactive"}).Stdout(ctx)
		ansibleExec.Container = ansibleExec.Container.WithMountedDirectory(filepath.Join(ansibleExec.MountPath(), "code"), code, dagger.ContainerWithMountedDirectoryOpts{Owner: ansibleExec.User()}).
			WithWorkdir(filepath.Join(ansibleExec.MountPath(), "code", "ansible"))
		// Run Ansible phase
		ansibleExec.Container.WithExec([]string{filepath.Join(ansibleExec.BinDir(), "ansible-playbook"), "-i", fmt.Sprintf("../%s/postgres-vm/do_hosts.yml", cfg.env), "--extra-vars", "exec_env=" + cfg.env, "db.yml"}).Stdout(ctx)
	case "destroy":
		helmRemove, err := helmExec.
			WithExec([]string{"uninstall", "-n", "ingress-nginx", "ingress-nginx", "--wait"}).
			WithExec([]string{"uninstall", "-n", "external-dns", "external-dns", "--wait"}).Stdout(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Println(helmRemove)
		destroy, err := tgruntExec.WithExec([]string{"run-all", "destroy", "--terragrunt-non-interactive", "--terragrunt-exclude-dir", "volumes/"}).Stdout(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Println(destroy)
	case "helm-platform":
		helm, err := helmExec.
			WithExec([]string{"upgrade", "--install", "ingress-nginx", "ingress-nginx/ingress-nginx", "--create-namespace", "-n", "ingress-nginx", "--version", "4.5.2", "-f", "ingress-nginx/values.yaml"}).
			WithExec([]string{"secrets", "upgrade", "--install", "external-dns", "external-dns/external-dns", "--create-namespace", "-n", "external-dns", "--version", "1.12.1", "-f", "external-dns/values.yaml", "-f", "external-dns/secrets.yaml"}).Stdout(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Println(helm)
	default:
		panic("Unknown action. 'plan', 'apply' and 'destroy' are supported")
	}
}
