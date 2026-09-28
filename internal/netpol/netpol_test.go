package netpol

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestKrakendNetpolEgress(t *testing.T) {
	np := KrakendNetpol("krakend", "default", map[string]string{
		"app.kubernetes.io/name": "krakend",
	})

	if len(np.Spec.Egress) != 2 {
		t.Fatalf("expected 2 egress rules, got %d", len(np.Spec.Egress))
	}

	anyDestinationHTTPS := np.Spec.Egress[0]
	if anyDestinationHTTPS.To != nil {
		t.Fatalf("expected HTTPS rule to omit destinations, got %#v", anyDestinationHTTPS.To)
	}
	if len(anyDestinationHTTPS.Ports) != 1 {
		t.Fatalf("expected 1 port in HTTPS rule, got %d", len(anyDestinationHTTPS.Ports))
	}
	port := anyDestinationHTTPS.Ports[0]
	if port.Protocol == nil || *port.Protocol != corev1.ProtocolTCP {
		t.Fatalf("expected HTTPS rule protocol TCP, got %v", port.Protocol)
	}
	if port.Port == nil || port.Port.IntVal != 443 {
		t.Fatalf("expected HTTPS rule port 443, got %v", port.Port)
	}

	allPods := np.Spec.Egress[1]
	if len(allPods.To) != 1 {
		t.Fatalf("expected 1 peer in all-pods rule, got %d", len(allPods.To))
	}
	peer := allPods.To[0]
	if peer.NamespaceSelector == nil || peer.PodSelector == nil {
		t.Fatalf("expected all-pods peer to select pods in all namespaces, got %#v", peer)
	}
	if len(allPods.Ports) != 0 {
		t.Fatalf("expected all-pods rule to allow every port, got %#v", allPods.Ports)
	}
}
