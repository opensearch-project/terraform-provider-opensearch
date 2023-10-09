package provider

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

var emptyStringSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

type awsV4SignerWrapper struct {
	internal http.RoundTripper
}

func (client *awsV4SignerWrapper) RoundTrip(request *http.Request) (*http.Response, error) {
	var hash string
	if request.Body == nil {
		hash = emptyStringSHA256
	} else {
		payload, error := io.ReadAll(request.Body)
		request.Body = io.NopCloser(bytes.NewReader(payload))
		if error != nil {
			return nil, error
		}
		hasher := sha256.New()
		hasher.Write(payload)
		hashBytes := hasher.Sum(nil)
		hash = hex.EncodeToString(hashBytes)
	}
	request.Header.Set("X-Amz-Content-Sha256", hash)

	return client.internal.RoundTrip(request)
}

func Wrap(internal http.RoundTripper) http.RoundTripper {
	return &awsV4SignerWrapper{internal: internal}
}
