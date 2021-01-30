package ip

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func IsValidIP(addr string) bool {
	return net.ParseIP(addr) != nil
}

func IsValidPort(port int32) bool {
	return port >= 0 && port <= 65535
}

func GetIPPort(ipPort string) (string, int32, error) {
	// Split ip and port
	tokens := strings.Split(ipPort, "/")
	if len(tokens) != 2 {
		return "", 0, fmt.Errorf("worng IP/Port")
	}
	ip := tokens[0]
	port := tokens[1]

	// Port type casting
	nPort, err := strconv.ParseInt(port, 10, 32)
	if err != nil {
		return "", 0, err
	}

	return ip, int32(nPort), nil
}

func IsValidIPPort(ipPort string) bool {
	// Get IP, Port
	ip, port, err := GetIPPort(ipPort)
	if err != nil {
		return false
	}

	// Check IP, Port
	return IsValidIP(ip) && IsValidPort(port)
}
