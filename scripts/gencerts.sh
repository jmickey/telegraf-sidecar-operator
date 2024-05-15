#! /bin/sh
set -eu -o pipefail

# generate temporary files in config/local/certs directory
readonly SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd "${SCRIPT_DIR}/../config/local/certs"

cfssl gencert -config ca-config.json -initca ca-csr.json | cfssljson -bare ca

cfssl gencert \
  -config=ca-config.json \
  -ca-key=ca-key.pem \
  -ca=ca.pem \
  -hostname=telegraf-sidecar-operator-webhook-service.telegraf-sidecar-operator.svc,telegraf-sidecar-operator-webhook-service.telegraf-sidecar-operator,telegraf-sidecar-operator-webhook-service \
  -profile=server \
  server-csr.json | cfssljson -bare tls

if [[ $(uname) = "Darwin" ]]; then
  sed -i '' -e "s/  caBundle: .*\$/  caBundle: $(cat ca.pem | base64 | tr -d '\n')/" ../webhook_cert_patch.yaml
else
  sed -i "s/  caBundle: .*\$/  caBundle: $(cat ca.pem | base64 | tr -d '\n')/" ../webhook_cert_patch.yaml
fi

kubectl create secret tls webhook-server-cert \
  --namespace telegraf-sidecar-operator \
  --cert=tls.pem \
  --key=tls-key.pem \
  --dry-run=client \
  --output=yaml > ../secret.yaml
