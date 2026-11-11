package provider

import (
	"net"
	"net/url"
	"testing"
)

// A request that fails before the server answers (here: connection refused)
// makes PerformRequest return a nil response. Reading the user must return
// that error instead of dereferencing the nil response.
func TestResourceOpensearchGetOpenDistroUserTransportError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	rawURL := "http://" + ln.Addr().String()
	// Close the listener so every connection to the address is refused.
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	conf := &ProviderConf{
		rawUrl:         rawURL,
		parsedUrl:      parsed,
		sniffing:       false,
		healthchecking: false,
		// Skip the version ping in getClient so the read itself hits the
		// refused connection.
		osVersion: "2.11.0",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("resourceOpensearchGetOpenDistroUser panicked: %v", r)
		}
	}()

	if _, err := resourceOpensearchGetOpenDistroUser("someone", conf); err == nil {
		t.Fatal("expected an error for a refused connection, got nil")
	}
}
