# Example webhook using KIND

## Setup as a local host service

In this case, the service will run in the same host as the kube-apiserver. For
KIND clusters, the IP address of the host from docker0 perspective is used.

- Generate the certs, by running

  ```bash
  ./certs/gencerts.sh --context [kind cluster context]
  ```

- Deploy the webhook configuration:

  ```bash
  kubectl apply -f ./deploy/localhost/webhook.yaml
  ```

- Run the server:

  ```bash
  go run main.go
  ```

  Alternatively, the program could run in debug mode as well.

## Setup as a cluster service

To be done later
