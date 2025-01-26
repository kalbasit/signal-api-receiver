# Signal API Receiver

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

### Running with Docker

`signal-api-receiver` is available as a Docker image on [Docker Hub]. This is the recommended way to run the application.

```bash
docker pull kalbasit/signal-api-receiver:latest
```

Here's an example docker run command:

```bash
docker run -p 8105:8105 \
  -e SIGNAL_ACCOUNT="your_signal_account" \
  -e SIGNAL_API_URL="wss://your-signal-api-url" \
  kalbasit/signal-api-receiver:latest
```

**Explanation**:

- `-p 8105:8105`: Maps port 8105 on the host to port 8105 in the container.
- `-e SIGNAL_ACCOUNT="your_signal_account"`: Sets the `SIGNAL_ACCOUNT` environment variable. Replace with your actual Signal account.
- `-e SIGNAL_API_URL="wss://your-signal-api-url"`: Sets the `SIGNAL_API_URL` environment variable. Replace with the URL of your Signal API.

Refer to the [Docker Hub] page for more information.

### Running from Source

To run `signal-api-receiver` from source, you need to provide the following command-line flags:

**Global Options:**

- `--log-level <value>`: Sets the logging level (default: "info"). Can be set using the `$LOG_LEVEL` environment variable.

**Options for the `serve` command:**

- `--record-message-type <value>`: Specifies which message types to record. Valid types are: "receipt", "typing", "data", "data-message", and "sync". This flag can be repeated to record multiple types (default: "data-message").

- `--repeat-last-message`: If enabled, repeats the last message if no new messages are available (applies to `/receive/pop`). This can be set using the `$REPEAT_LAST_MESSAGE` environment variable (default: false).

- `--signal-account <value>`: **Required.** Specifies your Signal account number. Can be set using the `$SIGNAL_ACCOUNT` environment variable.

- `--signal-api-url <value>`: **Required.** Specifies the URL of your Signal API, including the scheme (e.g., `wss://signal-api.example.com`). Can be set using the `$SIGNAL_API_URL` environment variable.

- `--server-addr <value>`: Sets the address where the server will listen (default: ":8105"). Can be set using the `$SERVER_ADDR` environment variable.

- `--mqtt-server <value>`: Server address to your MQTT Broker (must include the port e.g., `mqtt://broker.srv.local:1883`). Can be set using the `$MQTT_SERVER` environment variable.

- `--mqtt-user <value>` User used for authentication. Can be set using the `$MQTT_USER` environment variable.

- `--mqtt-password <value>` Password of the user used for authentication. Can be set using the `$MQTT_PASSWORD` environment variable.

- `--mqtt-client-id <value>`: A custom client-id. This should be unique on your broker. (default: `signal-api-receiver-<mac-address>`) Can be set using the `$MQTT_CLIENT_ID` environment variable.

- `--mqtt-topic-prefix <value>`: Define a custom topic-prefix to publish messages (default: `signal-api-receiver`). Topic resolves to `<topic-prefix>/message`. Can be set using the `$MQTT_TOPIC_PREFIX` environment variable.

- `--mqtt-qos <value>` Change the quality of service. Possible options are `0`, `1`, `2`. Can be set using the `MQTT_QOS` environment variable.

You can see all available options by running:

```bash
signal-api-receiver serve --help
```

### Kubernetes Deployment Example

Here's an example of how to deploy `signal-api-receiver` on Kubernetes alongside existing `signal-cli-rest-api` deployment that is not shown here:

<details>
<summary>Deployment</summary>

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
            - --signal-api-url=ws://signal-api.ns.svc:8080
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

</details>

<details>
<summary>Service</summary>

```yaml
apiVersion: v1
kind: Service
metadata:
  name: signal-api-receiver
  labels:
    app: signal-receiver
    tier: api
spec:
  type: ClusterIP
  ports:
    - name: receiver-web
      port: 8105
  selector:
    app: signal-receiver
    tier: api
```

</details>

<details>
<summary>Traefik IngressRoute</summary>

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: IngressRoute
metadata:
  name: signal-api
spec:
  entryPoints:
    - web
    - websecure
  routes:
    # This rule is for existing signal-cli-rest-api service that is not shown here.
    - kind: Rule
      match: Host(`signal-api.example.com`)
      priority: 10
      services:
        - name: signal-api
          port: http-web
    # The new rule for signal-api-receiver.
    - kind: Rule
      match: Host(`signal-api.example.com`) && Path(`/receive`)
      priority: 20
      services:
        - name: signal-api-receiver
          port: receiver-web
  tls:
    secretName: signal-api-tls
```

</details>

## License

This project is licensed under the MIT License - see the [LICENSE](/LICENSE) file for details.

[@bbernhard]: https://github.com/bbernhard
[docker hub]: https://hub.docker.com/r/kalbasit/signal-api-receiver
[exec-mode]: https://github.com/bbernhard/signal-cli-rest-api?tab=readme-ov-file#execution-modes
[signal-cli-rest-api]: https://github.com/bbernhard/signal-cli-rest-api
[signal_messenger]: https://www.home-assistant.io/integrations/signal_messenger/#sending-messages-to-signal-to-trigger-events
