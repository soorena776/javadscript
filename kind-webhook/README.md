# Example webhook using KIND

## Local setup

- Generate the certs, by running `./certs/gencerts.sh`
- Deploy the webhook configuration:

  ```bash
  kubectl apply -f ./deploy/webhook.yaml
  ```

- Run the server:

  ```bash
  go run main.go
  ```

  Alternatively, the program could run in debug mode as well.

## In cluster deployment

To be done later
