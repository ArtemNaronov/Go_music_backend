package addresses

import (
	"strings"
	"testing"
)

func TestForPortReturnsUniqueIPs(t *testing.T) {
	endpoints := ForPort(8080)
	seen := make(map[string]struct{})
	for _, ep := range endpoints {
		if _, ok := seen[ep.Addr]; ok {
			t.Fatalf("duplicate addr: %s", ep.Addr)
		}
		seen[ep.Addr] = struct{}{}
		if !strings.Contains(ep.Addr, ":8080") {
			t.Fatalf("expected port in addr, got %q", ep.Addr)
		}
	}
}

func TestForPortSkipsVirtualInterfaces(t *testing.T) {
	endpoints := ForPort(8080)
	for _, ep := range endpoints {
		if strings.Contains(strings.ToLower(ep.Label), "vethernet") {
			t.Fatalf("unexpected virtual endpoint: %+v", ep)
		}
	}
}
