package mqtt

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	pahop "github.com/eclipse/paho.golang/packets"
)

//nolint:gochecknoglobals
var QosValues = []int{0, 1, 2}

const ClientPrefix = "signal-api-receiver"

func MakeClientID(localAddr *net.TCPAddr) string {
	suffix := strconv.FormatInt(time.Now().Unix(), 10)

	netInterfaces, err := net.Interfaces()
	if err != nil || len(netInterfaces) == 0 {
		return ClientPrefix + "-" + suffix
	}

	// try to determine interface by local address
	predictedInterface := interfaceForLocalAddr(netInterfaces, localAddr)

	if predictedInterface == nil {
		// the last option is to guess the interface
		for _, netInterface := range netInterfaces {
			flags := netInterface.Flags

			if flags&net.FlagUp != 0 && flags&net.FlagLoopback == 0 && len(netInterface.HardwareAddr) > 0 {
				predictedInterface = &netInterface

				break
			}
		}
	}

	if predictedInterface != nil {
		suffix = strings.ReplaceAll(
			predictedInterface.HardwareAddr.String(), ":", "",
		)
	}

	return ClientPrefix + "-" + suffix
}

func interfaceForLocalAddr(netInterfaces []net.Interface, localAddr *net.TCPAddr) *net.Interface {
	for _, netInterface := range netInterfaces {
		netAddresses, err := netInterface.Addrs()
		if err != nil {
			continue
		}

		for _, netAddress := range netAddresses {
			var aIP net.IP
			switch v := netAddress.(type) {
			case *net.IPNet:
				aIP = v.IP
			case *net.IPAddr:
				aIP = v.IP
			}

			if aIP != nil && aIP.Equal(localAddr.IP) {
				return &netInterface
			}
		}
	}

	return nil
}

func isUnrecoverableReasonCodeError(reasonCode byte) bool {
	switch reasonCode {
	case pahop.DisconnectProtocolError,
		pahop.DisconnectNotAuthorized,
		pahop.DisconnectRetainNotSupported,
		pahop.DisconnectQoSNotSupported,
		pahop.DisconnectUseAnotherServer,
		pahop.DisconnectServerMoved:
		return true
	default:
		return false
	}
}

var (
	// ErrMqttUserAndPasswordRequired is returned if command has some but not all flags (requiredFlagsForMqtt) given.
	ErrMqttUserAndPasswordRequired = errors.New("some of the required flags for mqtt are missing")

	// Flags required for a functional mqtt configuration
	// Unauthenticated broker connections are intentionally unsupported.
	requiredFlagsForMqtt = []string{"mqtt-server", "mqtt-user", "mqtt-password"}
)

func ValidateFlags(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	for i, n := range requiredFlagsForMqtt[1:] {
		prev := cmd.IsSet(requiredFlagsForMqtt[i]) && len(cmd.String(requiredFlagsForMqtt[i])) > 0
		curr := cmd.IsSet(n) && len(cmd.String(n)) > 0

		// FlagXor
		if (prev || curr) && !(prev && curr) {
			_ = cli.ShowSubcommandHelp(cmd)

			return nil, fmt.Errorf("%w: %v", ErrMqttUserAndPasswordRequired, requiredFlagsForMqtt)
		}
	}

	return ctx, nil
}
