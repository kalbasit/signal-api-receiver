package mqtt

import (
	"net"
	"strconv"
	"strings"
	"time"
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
