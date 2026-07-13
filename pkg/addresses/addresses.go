package addresses

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

// Endpoint is an address clients can use to reach the server.
type Endpoint struct {
	Label string `json:"label"`
	Addr  string `json:"addr"`
	Hint  string `json:"hint,omitempty"`
}

// ForPort returns connection endpoints for the given listen port.
func ForPort(port int) []Endpoint {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var endpoints []Endpoint
	seen := make(map[string]struct{})

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if shouldSkipInterface(iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}

			key := ip.String()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			label, hint := describeInterface(iface.Name)
			endpoints = append(endpoints, Endpoint{
				Label: label,
				Addr:  fmt.Sprintf("%s:%d", ip.String(), port),
				Hint:  hint,
			})
		}
	}

	sort.Slice(endpoints, func(i, j int) bool {
		return endpointRank(endpoints[i]) < endpointRank(endpoints[j])
	})

	return endpoints
}

func shouldSkipInterface(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "vethernet"),
		strings.Contains(lower, "wsl"),
		strings.Contains(lower, "hyper-v"),
		strings.Contains(lower, "bluetooth"),
		strings.Contains(lower, "virtualbox"),
		strings.Contains(lower, "vmware"),
		strings.Contains(lower, "loopback"):
		return true
	default:
		return false
	}
}

func describeInterface(name string) (label, hint string) {
	lower := strings.ToLower(name)

	switch {
	case strings.Contains(lower, "amnezia"),
		strings.Contains(lower, "wireguard"),
		strings.Contains(lower, "openvpn"),
		strings.Contains(lower, "tun"),
		strings.Contains(lower, "tap"):
		return "VPN (" + name + ")", "С телефона через Amnezia / VPN"
	case strings.Contains(lower, "wi-fi"),
		strings.Contains(lower, "wifi"),
		strings.Contains(lower, "wlan"),
		strings.Contains(lower, "беспроводн"):
		return "Wi‑Fi (" + name + ")", "В домашней сети"
	case strings.Contains(lower, "ethernet"):
		return "Ethernet (" + name + ")", "В домашней сети по кабелю"
	default:
		return name, "Для подключения с другого устройства"
	}
}

func endpointRank(ep Endpoint) int {
	lower := strings.ToLower(ep.Label)
	switch {
	case strings.Contains(lower, "vpn"),
		strings.Contains(lower, "amnezia"),
		strings.Contains(lower, "wireguard"):
		return 0
	case strings.Contains(lower, "ethernet"):
		return 1
	case strings.Contains(lower, "wi"):
		return 2
	default:
		return 3
	}
}
