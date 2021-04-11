FROM golang:1.16-alpine AS build-stage

WORKDIR /go/src/testing-wh
COPY . .

RUN CGO_ENABLED=0 go build -o /bin/app --ldflags "-w -extldflags '-static'"

# Final image.
FROM alpine:latest
RUN apk --no-cache add \
  ca-certificates
COPY --from=build-stage /bin/app /usr/local/bin/app
ENTRYPOINT ["/usr/local/bin/app"]