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
	terragrunt run-all destroy --terragrunt-source ~/coding/selfhoster/infra-modules/ --terragrunt-source-update
	cd $$OLD_PWD
