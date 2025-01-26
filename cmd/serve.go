package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"

	"github.com/kalbasit/signal-api-receiver/pkg/mqtt"
	"github.com/kalbasit/signal-api-receiver/pkg/receiver"
	"github.com/kalbasit/signal-api-receiver/pkg/server"
)

var (
	// ErrInvalidSignalAccount is retruned if the given signal-account is not valid.
	ErrInvalidSignalAccount = errors.New("invalid signal account")

	// ErrSchemeMissing is returned if the given signal-api-url is missing a scheme.
	ErrSchemeMissing = errors.New("scheme is missing")

	// https://regex101.com/r/sxO3RG/1
	accountRegex = regexp.MustCompile(`^\+[0-9]+$`)
)

func makeRandomClientId() string {

	parts := []string{"signal-api-receiver"}

	netInterfaces, err := net.Interfaces()
	if err == nil {
		parts = append(parts, strings.ReplaceAll(netInterfaces[0].HardwareAddr.String(), ":", ""))
	}

	return strings.Join(parts, "-")
}

func serveCommand() *cli.Command {
	randomClientId := makeRandomClientId()

	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "start the signal-api-receiver HTTP server",
		Action:  serveAction(),
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name: "record-message-type",
				Usage: fmt.Sprintf(
					"Which message types to record? Valid message types: %v",
					receiver.AllMessageTypes(),
				),
				Value: []string{
					receiver.MessageTypeDataMessage.String(),
				},
				Validator: func(mts []string) error {
					for _, mt := range mts {
						_, err := receiver.ParseMessageType(mt)
						if err != nil {
							return fmt.Errorf("could not parse message type %q: %w", mt, err)
						}
					}

					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "repeat-last-message",
				Usage:   "Repeat the last message if there are no new messages (applies to /receive/pop)",
				Sources: cli.EnvVars("REPEAT_LAST_MESSAGE"),
			},
			&cli.StringFlag{
				Name:     "signal-account",
				Usage:    "The account number for signal",
				Sources:  cli.EnvVars("SIGNAL_ACCOUNT"),
				Required: true,
				Validator: func(a string) error {
					if !accountRegex.MatchString(a) {
						return fmt.Errorf(
							"%w: phone number must have leading + followed only by numbers",
							ErrInvalidSignalAccount,
						)
					}

					return nil
				},
			},
			&cli.StringFlag{
				Name:     "signal-api-url",
				Usage:    "The URL of the Signal api including the scheme. e.g wss://signal-api.example.com",
				Sources:  cli.EnvVars("SIGNAL_API_URL"),
				Required: true,
				Validator: func(u string) error {
					uri, err := url.Parse(u)
					if err != nil {
						return err
					}

					if uri.Scheme == "" {
						return ErrSchemeMissing
					}

					return nil
				},
			},
			&cli.StringFlag{
				Name:    "server-addr",
				Usage:   "The address of the server",
				Sources: cli.EnvVars("SERVER_ADDR"),
				Value:   ":8105",
			},
			&cli.StringFlag{
				Name:    "mqtt-server",
				Usage:   "MQTT Server Host and Port",
				Sources: cli.EnvVars("MQTT_PASSWORD"),
			},
			&cli.StringFlag{
				Name:    "mqtt-client-id",
				Usage:   "MQTT Client ID",
				Sources: cli.EnvVars("MQTT_CLIENT_ID"),
				Value:   randomClientId,
			},
			&cli.StringFlag{
				Name:    "mqtt-user",
				Usage:   "MQTT Username",
				Sources: cli.EnvVars("MQTT_USER"),
			},
			&cli.StringFlag{
				Name:    "mqtt-password",
				Usage:   "MQTT Password",
				Sources: cli.EnvVars("MQTT_PASSWORD"),
			},
			&cli.StringFlag{
				Name:    "mqtt-topic-prefix",
				Usage:   "MQTT Topic Prefix. {topic-prefix}/message",
				Sources: cli.EnvVars("MQTT_TOPIC_PREFIX"),
				Value:   "signal-api-receiver",
			},
			&cli.IntFlag{
				Name:    "mqtt-qos",
				Usage:   "MQTT Quality of Service (QoS) value",
				Sources: cli.EnvVars("MQTT_QOS"),
				Value:   1,
				Validator: func(q int64) error {
					allowedValues := []int64{1, 2, 3}

					if !slices.Contains(allowedValues, q) {
						return fmt.Errorf("MQTT QoS value '%d' is not allowed, allowed values: %v", q, allowedValues)
					}

					return nil
				},
			},
		},
	}
}

func serveAction() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		logger := zerolog.Ctx(ctx).With().Str("cmd", "serve").Logger()

		ctx = logger.WithContext(ctx)

		ctx, cancel := context.WithCancel(ctx)

		g, ctx := errgroup.WithContext(ctx)
		defer func() {
			if err := g.Wait(); err != nil {
				logger.Error().Err(err).Msg("error returned from g.Wait()")
			}
		}()

		// NOTE: Reminder that defer statements run last to first so the first
		// thing that happens here is the context is canceled which triggers the
		// errgroup 'g' to start exiting.
		defer cancel()

		g.Go(func() error {
			return autoMaxProcs(ctx, 30*time.Second, logger)
		})

		signalAPIURL := cmd.String("signal-api-url")

		uri, err := url.Parse(signalAPIURL)
		if err != nil {
			return fmt.Errorf("error parsing the url %q: %w", signalAPIURL, err)
		}

		uri = uri.JoinPath(fmt.Sprintf("/v1/receive/%s", cmd.String("signal-account")))

		logger.Info().
			Str("signal-api-url", uri.String()).
			Msg("the fully qualified signal-api URL was computed")

		sarc, err := receiver.New(ctx, uri, cmd.StringSlice("record-message-type")...)
		if err != nil {
			return fmt.Errorf("error creating a new receiver: %w", err)
		}

		if cmd.IsSet("mqtt-server") {
			err := mqtt.Init(
				ctx,
				cmd.String("mqtt-server"),
				cmd.String("mqtt-client-id"),
				cmd.String("mqtt-user"),
				cmd.String("mqtt-password"),
				cmd.String("mqtt-topic-prefix"),
				cmd.Int("mqtt-qos"),
			)

			if err != nil {
				return fmt.Errorf("error initializing MQTT: %w", err)
			}
		}

		srv := server.New(ctx, sarc, cmd.Bool("repeat-last-message"))

		server := &http.Server{
			Addr:              cmd.String("server-addr"),
			Handler:           srv,
			ReadHeaderTimeout: 10 * time.Second,
		}

		logger.Info().
			Str("server-addr", cmd.String("server-addr")).
			Msg("Server started")

		if err := server.ListenAndServe(); err != nil {
			return fmt.Errorf("error starting the HTTP listener: %w", err)
		}

		return nil
	}
}
