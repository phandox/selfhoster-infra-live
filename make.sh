#!/usr/bin/env bash
set -euo pipefail

declare -i verbosity=0

ingress_nginx_version="4.5.2"
firefly_core_version="1.4.0"
firefly_importer_version="1.3.1"
external_dns_version="1.12.1"
cert_manager_version="v1.11.0"

[[ -z ${TF_VAR_do_token:-} ]] && (>&2 echo "TF_VAR_do_token not loaded. Load with DO_TOKEN" ; exit 1)

function googleauth() {
  [[ -f "${HOME}/.config/gcloud/application_default_credentials.json" ]] && return 0
  verbose "Authenticating to Google Cloud..."
  gcloud auth application-default login
}

function verbose() {
  (( verbosity == 0 )) && return 0
  echo "$*"
}

function ansible_playbook() {
  local exec_env=$1
  cd ansible || (>&2 echo "Can't change dir to ansible" ; exit 1)
  source .venv/bin/activate
  ansible-playbook -i "../${exec_env}/postgres-vm/do_hosts.yml" --extra-vars "exec_env=${exec_env}" db.yml
  deactivate
  cd -
}

function helm_platform() {
  local cluster=$1
  doctl kubernetes cluster kubeconfig save "$cluster"
  cd charts || (>&2 echo "Can't change dir to dev/charts" ; exit 1)
  helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
    --create-namespace \
    -n ingress-nginx \
    --version "$ingress_nginx_version" \
    -f ingress-nginx/values.yaml
  helm secrets upgrade --install external-dns external-dns/external-dns \
    --create-namespace \
    -n external-dns \
    --version "$external_dns_version" \
    -f external-dns/values.yaml \
    -f external-dns/secrets.yaml
  cd -
}
function cert_manager() {
  local cluster=$1
  doctl kubernetes cluster kubeconfig save "$cluster"
  cd charts || (>&2 echo "Can't change dir to dev/charts" ; exit 1)
  kubectl apply --server-side -f cert-manager/cert-manager.crds.yaml
  helm upgrade --install cert-manager jetstack/cert-manager \
    --create-namespace \
    -n cert-manager \
    --set 'extraArgs={--acme-http01-solver-nameservers=8.8.8.8:53\,1.1.1.1:53}' \
    --version "$cert_manager_version" \
    -f cert-manager/values.yaml
  kubectl apply --server-side -f cert-manager/cluster-issuer.yaml
  cd -
}

function helm_workload() {
  local cluster=$1
  local exec_env=$2
  doctl kubernetes cluster kubeconfig save "$cluster"
  cd charts || (>&2 echo "Can't change dir to dev/charts" ; exit 1)
  kubectl apply --server-side -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: firefly-iii
EOF
  if [[ -f "cert-manager/stages/${exec_env}/firefly-iii-tls.secret.yaml" ]]; then
    sops --decrypt "cert-manager/stages/${exec_env}/firefly-iii-tls.secret.yaml" \
      | yq 'del(.metadata.annotations."kubectl.kubernetes.io/last-applied-configuration"), (.metadata.creationTimestamp, .metadata.resourceVersion, .metadata.uid) |= null' \
      | kubectl apply --server-side -f -
  fi
  helm secrets upgrade --install core firefly-iii/firefly-iii \
    -n firefly-iii \
    --version "$firefly_core_version" \
    -f firefly-iii/values.yaml \
    -f "firefly-iii/stages/${exec_env}/env.yaml" \
    -f "firefly-iii/stages/${exec_env}/secrets.yaml"
  cd -
}

case $1 in
  tgrunt-plan-dev)
    cd dev/
    verbose "Running terragrunt run-all plan target"
    googleauth
    terragrunt run-all plan --terragrunt-source ~/coding/selfhoster/ --terragrunt-source-update
    cd -
    ;;
  tgrunt-apply-dev)
    cd dev/
    verbose "Running terragrunt run-all apply target"
    googleauth
    terragrunt run-all apply --terragrunt-source ~/coding/selfhoster/ --terragrunt-source-update
    cd -
    ;;
  tgrunt-destroy-dev)
    cd dev/
    verbose "Running terragrunt run-all destroy target"
    googleauth
    # force removing Digital ocean resources, created in K8S cluster
    helm uninstall -n ingress-nginx ingress-nginx --wait
    helm uninstall -n external-dns external-dns --wait
    terragrunt run-all destroy --terragrunt-source ~/coding/selfhoster --terragrunt-source-update --terragrunt-exclude-dir volumes/
    rm "${HOME}/.config/gcloud/application_default_credentials.json"
    cd -
    ;;
  tgrunt-plan-prod)
    cd prod/
    verbose "Running terragrunt run-all plan target - prod"
    googleauth
    terragrunt run-all plan
    cd -
    ;;
  tgrunt-apply-prod)
    cd prod/
    verbose "Running terragrunt run-all apply target - prod"
    googleauth
    terragrunt run-all apply
    cd -
    ;;
  tgrunt-destroy-prod)
    cd prod/
    verbose "Running terragrunt run-all destroy target"
    googleauth
    # force removing Digital ocean resources, created in K8S cluster
    helm uninstall -n ingress-nginx ingress-nginx --wait
    helm uninstall -n external-dns external-dns --wait
    terragrunt run-all destroy --terragrunt-exclude-dir volumes/
    rm "${HOME}/.config/gcloud/application_default_credentials.json"
    cd -
    ;;
  ansible-run-dev)
    verbose "Running Ansible target"
    ansible_playbook dev
    ;;
  ansible-run-prod)
    verbose "Running Ansible target"
    ansible_playbook prod
    ;;
  helm-platform)
    verbose "Installing K8S platform Helm charts"
    helm_platform doks-fra1-001 dev
    ;;
  helm-workload)
    verbose "Installing K8S workload Helm charts"
    helm_workload doks-fra1-001 dev
    ;;
  cert-manager)
    verbose "Installing cert-manager"
    cert_manager doks-fra1-001 dev
    ;;
  helm-platform-prod)
    verbose "Installing K8S platform Helm charts"
    helm_platform doks-fra1-prod-001 prod
    ;;
  helm-workload-prod)
    verbose "Installing K8S workload Helm charts"
    helm_workload doks-fra1-prod-001 prod
    ;;
  cert-manager-prod)
    verbose "Installing cert-manager"
    cert_manager doks-fra1-prod-001 prod
    ;;
  *)
    echo "Unknown target"
    exit 1
    ;;
esac
