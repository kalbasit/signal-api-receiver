package mqtt_test

import (
	"net"
	"strings"
	"testing"

	"github.com/kalbasit/signal-api-receiver/pkg/mqtt"
)

func TestMakeRandomClientID(t *testing.T) {
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
