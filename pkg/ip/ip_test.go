package ip

import (
	"testing"
)

const (
	ipv4Local       = "127.0.0.1"
	ipv4InvalidAddr = "999.0.0.1"
	ipv6Local       = "::1"
	ipv6InvalidAddr = "fffff::1"

	portHTTP    = 80
	portInvalid = 200000
)

func TestIsValidIP(t *testing.T) {
	v := IsValidIP(ipv4Local)
	if v != true {
		t.Errorf("wrong result - %s", ipv4Local)
	}

	v = IsValidIP(ipv4InvalidAddr)
	if v != false {
		t.Errorf("wrong result - %s", ipv4InvalidAddr)
	}

	v = IsValidIP(ipv6Local)
	if v != true {
		t.Errorf("wrong result - %s", ipv6Local)
	}

	v = IsValidIP(ipv4InvalidAddr)
	if v != false {
		t.Errorf("wrong result - %s", ipv6InvalidAddr)
	}
}

func TestIsValidPort(t *testing.T) {
	v := IsValidPort(portHTTP)
	if v != true {
		t.Errorf("wrong result - %d", portHTTP)
	}

	v = IsValidPort(portInvalid)
	if v != false {
		t.Errorf("wrong result - %d", portInvalid)
	}
}
