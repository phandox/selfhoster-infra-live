plan-dev:
	cd dev
	terragrunt run-all plan --terragrunt-source ~/coding/selfhoster/infra-modules/ --terragrunt-source-update
	cd $$OLD_PWD

plan-dev-debug:
	cd dev
	terragrunt run-all plan --terragrunt-source ~/coding/selfhoster/infra-modules/ --terragrunt-source-update --terragrunt-log-level debug --terragrunt-debug
	cd $$OLD_PWD

apply-dev:
	cd dev
	terragrunt run-all apply --terragrunt-source ~/coding/selfhoster/infra-modules/ --terragrunt-source-update
	cd $$OLD_PWD

apply-dev-debug:
	cd dev
	terragrunt run-all apply --terragrunt-source ~/coding/selfhoster/infra-modules/ --terragrunt-source-update --terragrunt-log-level debug --terragrunt-debug
	cd $$OLD_PWD

destroy-dev:
	cd dev
	terragrunt run-all destroy --terragrunt-source ~/coding/selfhoster/infra-modules/ --terragrunt-source-update --terragrunt-exclude-dir dev/volumes
	cd $$OLD_PWD

# Destroy also persistent resources
destroy-dev-all:
	cd dev
	terragrunt run-all destroy --terragrunt-source ~/coding/selfhoster/infra-modules/ --terragrunt-source-update
	cd $$OLD_PWD

plan-dev-no-k8s:
	cd dev
	terragrunt run-all plan --terragrunt-source ~/coding/selfhoster/infra-modules/ --terragrunt-source-update --terragrunt-exclude-dir dev/doks
	cd $$OLD_PWD

apply-dev-no-k8s:
	cd dev
	terragrunt run-all apply --terragrunt-source ~/coding/selfhoster/infra-modules/ --terragrunt-source-update --terragrunt-exclude-dir dev/doks
	cd $$OLD_PWD

ansible-run:
	source ansible/.venv/bin/activate
	ansible-playbook -i dev/postgres-vm/do_hosts.yml ansible/db.yml
	deactive
