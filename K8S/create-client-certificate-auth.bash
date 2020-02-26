#!/usr/bin/env bash
set -euo pipefail

#############################
# This bash creates client certificates for authentication, with admin
# privilages. It assuems that kubectl is configured, and the user has the right
# privilages to create the csr and approve it for the permissions configured in
# csr.conf.

function cleanup {
  echo "Cleaning up..."
  
  kubectl delete --ignore-not-found csr mycsr
}

trap cleanup EXIT # cleanup on exit

# read the arguments
username=
newcontext=

while (( "$#" )); do
  case "$1" in
    -u|--username)
      username=$2
      shift 2
      ;;
    -c|--newcontext)
      newcontext=$2
      shift 2
      ;;
    *) 
      shift
      ;;
  esac
done

# make sure the variables are all set
help=`cat << EOF
-----------------------
Usage:
  create-client-certificate-auth -u username -c contextname

    -u,--username: required.  The new alias for the current user that will be used as the new user in the kubeconfig.
    -c,--newcontext: required. The name of the new context created in the kubeconfig.
-----------------------
EOF
`

if test -z "$username"; then
    echo "the username is not provided"
    echo "$help"
    exit 1
fi

if test -z "$newcontext"; then
    echo "the newcontext is not provided"
    echo "$help"
    exit 1
fi

cert_dir="~/.kube/certs/${newcontext}"
mkdir -p ${cert_dir}
cd "${cert_dir}"

cat << EOF > csr.cnf
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
[ dn ]
CN = ${username}
O = system:masters
[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=serverAuth,clientAuth
EOF

openssl genrsa -out ./user.key 4096
openssl req -config ./csr.cnf -new -key ./user.key -nodes -out ./user.csr

cat << EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: mycsr
spec:
  groups:
  - system:authenticated
  request: $(cat ./user.csr | base64 | tr -d '\n')
  usages:
  - digital signature
  - key encipherment
  - server auth
  - client auth
EOF

kubectl certificate approve mycsr

# give some time 
sleep 5

user_cert_data=`kubectl get csr mycsr -o json | jq '.status.certificate'` 
user_key_data=`cat user.key | base64 | tr -d '\n'`

kubectl get csr mycsr -o json | jq -r '.status.certificate' | base64 --decode > user.cert

context=`kubectl config current-context| tr -d '\n'`
cluster=$(kubectl config view --raw -o json | jq -r --arg context "${context}" '.contexts[] | select(.name == $context) | .context.cluster')

kubectl config set-credentials "${username}" --client-certificate=./user.cert --client-key=./user.key
kubectl config set-context "${newcontext}" --cluster="${cluster}" --user="${username}"

echo "Successfully created the context '${newcontext}' using client certificates at '${cert_dir}' for the current user with alias '${username}'"
