package network

import (
	"errors"
	"fmt"
	"net"
	"time"

	ipify "github.com/rdegges/go-ipify"
)

// LANIP gives you the first non-loopback IP address
func LANIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("network not found")
}

// WANIP gives you your WAN IP of the device
func WANIP() (string, error) {
	return ipify.GetIp()
}

// GetRelativeTime returns a human-readable relative time string
func GetRelativeTime(timestamp int64) string {
	now := time.Now().UnixMilli()
	diff := now - timestamp

	if diff < 60000 { // Less than 1 minute
		return "just now"
	} else if diff < 3600000 { // Less than 1 hour
		minutes := diff / 60000
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 86400000 { // Less than 1 day
		hours := diff / 3600000
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := diff / 86400000
		return fmt.Sprintf("%d days ago", days)
	}
}
