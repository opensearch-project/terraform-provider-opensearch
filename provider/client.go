package provider

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// OpenSearchClient wraps the official opensearch-go/v4 client
type OpenSearchClient struct {
	*opensearchapi.Client
	config *ProviderConf
}

// NewOpenSearchClient creates a new OpenSearch client using the official SDK
func NewOpenSearchClient(conf *ProviderConf) (*OpenSearchClient, error) {
	cfg := opensearch.Config{
		Addresses: []string{conf.rawUrl},
	}

	// Configure transport with TLS settings
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: conf.insecure,
		},
	}

	// Handle client certificates
	if conf.certPemPath != "" && conf.keyPemPath != "" {
		certPem, _, err := readPathOrContent(conf.certPemPath)
		if err != nil {
			return nil, err
		}
		keyPem, _, err := readPathOrContent(conf.keyPemPath)
		if err != nil {
			return nil, err
		}
		cert, err := tls.X509KeyPair([]byte(certPem), []byte(keyPem))
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	// Handle CA certificate
	if conf.cacertFile != "" {
		caCert, _, err := readPathOrContent(conf.cacertFile)
		if err != nil {
			return nil, err
		}
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			caCertPool = x509.NewCertPool()
		}
		caCertPool.AppendCertsFromPEM([]byte(caCert))
		transport.TLSClientConfig.RootCAs = caCertPool
	}

	// Handle host override for ServerName
	if conf.hostOverride != "" {
		transport.TLSClientConfig.ServerName = conf.hostOverride
	}

	// Configure proxy if specified
	if conf.proxy != "" {
		proxyURL, err := url.Parse(conf.proxy)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	// Authentication - Basic Auth
	if conf.username != "" && conf.password != "" {
		cfg.Username = conf.username
		cfg.Password = conf.password
	}

	// Check for URL-based credentials
	if conf.parsedUrl != nil && conf.parsedUrl.User != nil {
		username := conf.parsedUrl.User.Username()
		password, _ := conf.parsedUrl.User.Password()
		if username != "" && cfg.Username == "" {
			cfg.Username = username
			cfg.Password = password
		}
	}

	// AWS SigV4 signing
	if conf.signAWSRequests {
		signer, err := newAWSSigner(conf)
		if err != nil {
			return nil, err
		}
		cfg.Signer = signer
	}

	// Token-based authentication (if no AWS signing and no basic auth)
	if !conf.signAWSRequests && conf.token != "" && cfg.Username == "" {
		// Token auth will be handled via custom transport wrapper
		cfg.Transport = &tokenTransport{
			base:         transport,
			tokenName:    conf.tokenName,
			token:        conf.token,
			hostOverride: conf.hostOverride,
		}
	} else if conf.hostOverride != "" {
		// Host override without token auth
		cfg.Transport = &hostOverrideTransport{
			base:         transport,
			hostOverride: conf.hostOverride,
		}
	} else {
		cfg.Transport = transport
	}

	// Create the API client using the config
	client, err := opensearchapi.NewClient(opensearchapi.Config{
		Client: cfg,
	})
	if err != nil {
		return nil, err
	}

	return &OpenSearchClient{
		Client: client,
		config: conf,
	}, nil
}

// tokenTransport wraps an http.RoundTripper to add token authentication
type tokenTransport struct {
	base         *http.Transport
	tokenName    string
	token        string
	hostOverride string
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.tokenName+" "+t.token)
	if t.hostOverride != "" {
		req.Host = t.hostOverride
	}
	return t.base.RoundTrip(req)
}

// hostOverrideTransport wraps an http.RoundTripper to override the host header
type hostOverrideTransport struct {
	base         *http.Transport
	hostOverride string
}

func (t *hostOverrideTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.hostOverride != "" {
		req.Host = t.hostOverride
	}
	return t.base.RoundTrip(req)
}
