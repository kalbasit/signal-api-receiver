# signal-api-receiver

## Introduction

### Problem statement

I use the excellent [signal-cli-rest-api] by [@bbernhard] on
my server, and my home-assistant is configured to send me and the Home group
notifications of all kinds. Sending messages quickly is crucial for many of my
automations, so I'm running the API [in `json-rpc` mode][exec-mode].

Recently, I wanted to add a way for Home Assistant to receive messages from us
to trigger automations (or stop others). However, when the signal-api is
running in `json-rpc` mode, the `/v1/receive` endpoint becomes websocket-only.
This is not supported by [Home Assistant's signal_messenger
integration][signal_messenger], which relies on REST API calls.

### Solution

This project, `signal-api-receiver`, provides a solution by creating a
lightweight wrapper that:

- Consumes the websocket stream from the `/v1/receive` endpoint.
- Stores received messages in memory.
- Exposes a REST API for retrieving those messages.

This approach allows Home Assistant to easily receive Signal messages and
trigger automations without requiring modifications to the existing
`signal-cli-rest-api` or the Home Assistant integration.

### Alternative Solutions

While developing `signal-api-receiver` solved my immediate need, there were
other potential approaches to this problem:

1. Improve the Home Assistant integration with Signal to function properly with a Websocket.
1. Propose a new endpoint to the `signal-cli-rest-api` that responds to REST.

These alternatives might be more comprehensive solutions in the long term, but
creating the wrapper provided a more immediate and focused solution for my
specific use case.

## API Endpoints

`signal-api-receiver` exposes the following API endpoints:

- `GET /receive/pop`:
  - Returns one message at a time from the queue.
  - If no messages are available, it returns a `204 No Content` status.
- `GET /receive/flush`:
  - Returns all available messages as a list.
  - If no messages are available, it returns an empty list (`[]`).

## Usage

To run `signal-api-receiver`, you need to provide the following command-line flags:

- `-signal-account string`: The account number for Signal.
- `-signal-api-url string`: The URL of the Signal API, including the scheme (e.g., `wss://signal-api.example.com`).

By default, the server starts on `:8105`. You can change this using the `-addr` flag (e.g., `-addr :8080`).

### Kubernetes Deployment Example

Here's an example of how to deploy `signal-api-receiver` on Kubernetes:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: signal-api-receiver
  labels:
    app: signal-receiver
    tier: api
spec:
  replicas: 1
  selector:
    matchLabels:
      app: signal-receiver
      tier: api
  template:
    metadata:
      labels:
        app: signal-receiver
        tier: api
    spec:
      containers:
        - image: kalbasit/signal-receiver:latest
          name: signal-receiver
          args:
            - /bin/signal-api-receiver
            - serve
            - --signal-api-url=wss://signal-api.example.com
            - --signal-account=+19876543210
          ports:
            - containerPort: 8105
              name: receiver-web
          livenessProbe:
            httpGet:
              path: /healthz
              port: receiver-web
            initialDelaySeconds: 15
            periodSeconds: 20
          readinessProbe:
            httpGet:
              path: /healthz
              port: receiver-web
            initialDelaySeconds: 5
            periodSeconds: 10
```

## License

This project is licensed under the MIT License - see the [LICENSE](/LICENSE) file for details.

[@bbernhard]: https://github.com/bbernhard
[exec-mode]: https://github.com/bbernhard/signal-cli-rest-api?tab=readme-ov-file#execution-modes
[signal-cli-rest-api]: https://github.com/bbernhard/signal-cli-rest-api
[signal_messenger]: https://www.home-assistant.io/integrations/signal_messenger/#sending-messages-to-signal-to-trigger-events
