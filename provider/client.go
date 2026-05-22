package provider

import (
	"crypto/tls"
	"crypto/x509"
	"log"
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

	// Check for URL-based credentials.
	if conf.parsedUrl != nil && conf.parsedUrl.User != nil {
		username := conf.parsedUrl.User.Username()
		password, _ := conf.parsedUrl.User.Password()
		if username != "" && cfg.Username == "" {
			cfg.Username = username
			cfg.Password = password
		}
	}

	// Authentication - Basic Auth. Do this second, so explicit config overrides credentials in the URL
	if conf.username != "" && conf.password != "" {
		cfg.Username = conf.username
		cfg.Password = conf.password
	}

	// Auto-detect AWS OpenSearch based on URL patterns (mirrors original getClient logic)
	awsRegion := conf.awsRegion
	awsService := conf.awsSig4Service
	signAwsRequests := conf.signAWSRequests

	if conf.parsedUrl != nil {
		if m := awsUrlRegexp.FindStringSubmatch(conf.parsedUrl.Hostname()); m != nil && conf.signAWSRequests {
			// AWS OpenSearch Service: *.es.amazonaws.com
			if awsRegion == "" {
				awsRegion = m[1] // Extract region from URL
			}
			log.Printf("[INFO] Using AWS OpenSearch Service in region: %s", awsRegion)
		} else if m := awsOpensearchServerlessUrlRegexp.FindStringSubmatch(conf.parsedUrl.Hostname()); (m != nil || (conf.awsSig4Service == "aoss" && conf.awsRegion != "")) && conf.signAWSRequests {
			// AWS OpenSearch Serverless: *.aoss.amazonaws.com
			awsService = "aoss"
			if m != nil && awsRegion == "" {
				awsRegion = m[1]
			} else if awsRegion == "" {
				awsRegion = conf.awsRegion
			}
			log.Printf("[INFO] Using AWS OpenSearch Serverless in region: %s", awsRegion)
		} else if awsRegion != "" && conf.signAWSRequests {
			// AWS region explicitly set
			log.Printf("[INFO] Using AWS: %s", awsRegion)
		} else if !signAwsRequests {
			// Not an AWS URL, disable AWS signing
			signAwsRequests = false
		}
	}

	// AWS SigV4 signing
	if signAwsRequests && awsRegion != "" {
		// Temporarily override region and service for signer
		originalRegion := conf.awsRegion
		originalService := conf.awsSig4Service
		conf.awsRegion = awsRegion
		conf.awsSig4Service = awsService

		signer, err := newAWSSigner(conf)

		// Restore original values
		conf.awsRegion = originalRegion
		conf.awsSig4Service = originalService

		if err != nil {
			return nil, err
		}
		cfg.Signer = signer

		// Set flavor and version for Serverless (matching original getClient behavior)
		// Serverless is always OpenSearch flavor with minimum version 2.0.0
		if awsService == "aoss" {
			conf.flavor = OpenSearch
			if conf.osVersion == "" {
				conf.osVersion = minimalOpensearchServerlessVersion
			}
		}
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
