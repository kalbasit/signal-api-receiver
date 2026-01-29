package mqtt_test

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/kalbasit/signal-api-receiver/pkg/mqtt"
)

func TestMakeRandomClientID(t *testing.T) {
	t.Parallel()

	clientID := mqtt.MakeClientID(&net.TCPAddr{IP: net.IPv4(10, 0, 0, 1)})

	if !strings.HasPrefix(clientID, mqtt.ClientPrefix+"-") {
		t.Fatalf("client ID should have prefix, got %q", clientID)
	}

	suffix := strings.TrimPrefix(clientID, mqtt.ClientPrefix+"-")
	if suffix == "" {
		t.Fatalf("client ID suffix should not be empty")
	}

	if strings.Contains(suffix, ":") {
		t.Fatalf("client ID suffix should not contain colons, got %q", suffix)
	}
}

func TestValidateFlags(t *testing.T) {
	t.Parallel()

	t.Run("none-set", func(t *testing.T) {
		t.Parallel()

		cmd := newCommand()

		_, err := mqtt.ValidateFlags(context.Background(), cmd)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("all-set", func(t *testing.T) {
		t.Parallel()

		cmd := newCommand()
		setFlag(t, cmd, "mqtt-server", "mqtt://broker.srv:1883")
		setFlag(t, cmd, "mqtt-user", "tester")
		setFlag(t, cmd, "mqtt-password", "secret")

		_, err := mqtt.ValidateFlags(context.Background(), cmd)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("partial-set", func(t *testing.T) {
		t.Parallel()

		cmd := newCommand()
		setFlag(t, cmd, "mqtt-server", "mqtt://broker.srv:1883")

		_, err := mqtt.ValidateFlags(context.Background(), cmd)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if !errors.Is(err, mqtt.ErrMqttUserAndPasswordRequired) {
			t.Fatalf("expected ErrMqttUserAndPasswordRequired, got %v", err)
		}

		if !strings.Contains(err.Error(), "mqtt-user") || !strings.Contains(err.Error(), "mqtt-password") {
			t.Fatalf("expected missing flags in error, got %v", err)
		}
	})
}

func newCommand() *cli.Command {
	return &cli.Command{
		Name:      "test",
		Writer:    io.Discard,
		ErrWriter: io.Discard,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "mqtt-server"},
			&cli.StringFlag{Name: "mqtt-user"},
			&cli.StringFlag{Name: "mqtt-password"},
		},
	}
}

func setFlag(t *testing.T, cmd *cli.Command, name, value string) {
	t.Helper()

	if err := cmd.Set(name, value); err != nil {
		t.Fatalf("failed to set %s: %v", name, err)
	}
}
