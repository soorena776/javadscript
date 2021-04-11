#!/bin/bash
set -e

CERTSDIR=$(dirname "$0")
DEPLOYDIR=$CERTSDIR/../deploy

function cleanup {
  echo "Cleaning up..."
  
  kubectl delete --ignore-not-found csr mycsr
  rm -f $CERTSDIR/csr.cnf
  rm -f $CERTSDIR/user.csr
}
trap cleanup EXIT # cleanup on exit


context=
while [ $# -gt 0 ]; do
  case ${1} in
      --context)
          context="$2"
          shift
          ;;
      *)
          echo "unknown key"
          ;;
  esac
  shift
done
[ -z "${context}" ] && echo "ERROR: --context flag is required to target the kubernetes cluster" && exit

# the context should be configured against a kind cluster. 
if [ `kubectl config current-context` != "${context}" ]; then echo "not configured against the specified context ${context}" && exit 1; fi
kubectl cluster-info

# get the localhost ip address within a docker container (where kube-apiserver is running)
# TODO: make this more robust robust and validate
localhostIP=$(sudo ip addr show docker0 | sed -n 's|.*inet \([0-9\.]*\).*|\1|p')
servicePort=8099

# replace the url in the webhook configuration yaml for localhost
sed -i "s|url: .*|url: https://${localhostIP}:${servicePort}/hook|" $DEPLOYDIR/localhost/webhook.yaml

username=testing-wh
# create the csr config
cat << EOF > $CERTSDIR/csr.cnf
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = req_ext
x509_extensions = v3_req
[ dn ]
CN = ${username}
O = system:masters
[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=serverAuth,clientAuth
[req_ext]
subjectAltName = @alt_names
[v3_req]
subjectAltName = @alt_names
[alt_names]
DNS.1   = localhost
DNS.2 = testing-wh.default.svc
IP.1   = ${localhostIP}
EOF

openssl genrsa -out $CERTSDIR/user.key 4096
openssl req -config $CERTSDIR/csr.cnf -new -key $CERTSDIR/user.key -nodes -out $CERTSDIR/user.csr

# create a csr object and submit it to the cluster
kubectl delete --ignore-not-found csr mycsr
cat << EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: mycsr
spec:
  groups:
  - system:authenticated
  request: $(cat $CERTSDIR/user.csr | base64 | tr -d '\n')
  usages:
  - digital signature
  - key encipherment
  - server auth
  - client auth
EOF

# aprove the csr and wait a bit
kubectl certificate approve mycsr
sleep 5

# generate the cert file
user_cert_data=$(kubectl get csr mycsr -o json | jq -r '.status.certificate')
echo "${user_cert_data}" | base64 --decode > $CERTSDIR/user.cert

user_key_data=$(cat $CERTSDIR/user.key | base64 | tr -d '\n')

# replace the caBundle in both localhost and service webhook configuration yaml
sed -i "s|caBundle:.*|caBundle: ${user_cert_data}|" $DEPLOYDIR/localhost/webhook.yaml
sed -i "s|caBundle:.*|caBundle: ${user_cert_data}|" $DEPLOYDIR/service/webhook.yaml

# replace the secrets in the service webhook configuration yaml
sed -i "s|tls.crt: .*|tls.crt: ${user_cert_data}|" $DEPLOYDIR/service/deployment.yaml
sed -i "s|tls.key: .*|tls.key: ${user_key_data}|" $DEPLOYDIR/service/deployment.yaml
