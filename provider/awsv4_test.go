package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"
)

type MockRoundTripper struct {
}

func (roundTripper *MockRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return new(http.Response), nil
}

func TestRoundTripWithEmptyBody(t *testing.T) {
	sut := Wrap(new(MockRoundTripper))

	request := new(http.Request)
	request.Header = make(http.Header)

	_, err := sut.RoundTrip(request)

	if err != nil {
		t.Fatal(err)
	}

	if header, contains := request.Header["X-Amz-Content-Sha256"]; !contains || len(header) != 1 || header[0] != emptyStringSHA256 {
		t.Fatal("Request with empty body doesn't contain X-Amz-Content-Sha256 header with empty string hash value.")
	}
}

func TestRoundTripWithBody(t *testing.T) {
	sut := Wrap(new(MockRoundTripper))
	request := new(http.Request)
	request.Header = make(http.Header)

	body := "body"
	request.Body = io.NopCloser(strings.NewReader(body))

	hasher := sha256.New()
	hasher.Write([]byte(body))
	hashBytes := hasher.Sum(nil)
	expectedHash := hex.EncodeToString(hashBytes)

	_, err := sut.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}

	if header, contains := request.Header["X-Amz-Content-Sha256"]; !contains || len(header) != 1 || header[0] != expectedHash {
		t.Fatal("Request with body doesn't contain X-Amz-Content-Sha256 header with correct hash value.")
	}
}
