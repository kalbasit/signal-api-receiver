package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"

	"github.com/kalbasit/signal-api-receiver/receiver"
	"github.com/kalbasit/signal-api-receiver/server"
)

func serveCommand(logger zerolog.Logger) *cli.Command {
	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "start the signal-api-receiver HTTP server",
		Action:  serveAction(logger.With().Str("cmd", "serve").Logger()),
		Flags: []cli.Flag{
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
				// TODO: validator
			},
			&cli.StringFlag{
				Name:     "signal-api-url",
				Usage:    "The URL of the Signal api including the scheme. e.g wss://signal-api.example.com",
				Sources:  cli.EnvVars("SIGNAL_API_URL"),
				Required: true,
				// TODO: validator
			},
			&cli.StringFlag{
				Name:    "server-addr",
				Usage:   "The address of the server",
				Sources: cli.EnvVars("SERVER_ADDR"),
				Value:   ":8105",
			},
		},
	}
}

func serveAction(logger zerolog.Logger) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
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

		signalAccount := cmd.String("signal-account")

		uri.Path = fmt.Sprintf("/v1/receive/%s", signalAccount)

		logger.Info().
			Str("signal-api-url", uri.String()).
			Msg("the fully qualified signal-api URL was computed")

		sarc, err := receiver.New(ctx, uri)
		if err != nil {
			return fmt.Errorf("error creating a new receiver: %w", err)
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
