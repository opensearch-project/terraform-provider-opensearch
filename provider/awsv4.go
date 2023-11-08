package provider

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
)

type awsV4SignerWrapper struct {
	internal http.RoundTripper
}

func hashPayload(hasher hash.Hash, payload []byte) string {
	hasher.Write(payload)
	hashBytes := hasher.Sum(nil)
	hash := hex.EncodeToString(hashBytes)
	return hash
}

func (client *awsV4SignerWrapper) RoundTrip(request *http.Request) (*http.Response, error) {
	hasher := sha256.New()
	var hash string
	if request.Body == nil {
		hash = hashPayload(hasher, []byte(""))
	} else {
		payload, error := io.ReadAll(request.Body)
		request.Body = io.NopCloser(bytes.NewReader(payload))
		if error != nil {
			return nil, error
		}

		hash = hashPayload(hasher, payload)
	}
	request.Header.Set("X-Amz-Content-Sha256", hash)

	return client.internal.RoundTrip(request)
}

func Wrap(internal http.RoundTripper) http.RoundTripper {
	return &awsV4SignerWrapper{internal: internal}
}
