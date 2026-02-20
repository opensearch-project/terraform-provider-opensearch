package provider

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProviderFactories func(providers *[]*schema.Provider) map[string]func() (*schema.Provider, error)
var testAccProvider *schema.Provider

var testAccOpendistroProviders map[string]*schema.Provider
var testAccOpendistroProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"opensearch": testAccProvider,
	}
	testAccProviderFactories = func(providers *[]*schema.Provider) map[string]func() (*schema.Provider, error) {
		// this is an SDKV2 compatible hack, the "factory" functions are
		// effectively singletons for the lifecycle of a resource.Test
		var factories = make(map[string]func() (*schema.Provider, error), len(testAccProviders))
		for name, p := range testAccProviders {
			factories[name] = func() (*schema.Provider, error) {
				return p, nil
			}
			*providers = append(*providers, p)
		}
		return factories
	}

	testAccOpendistroProvider = Provider()
	testAccOpendistroProviders = map[string]*schema.Provider{
		"opensearch": testAccOpendistroProvider,
	}

	opendistroOriginalConfigureFunc := testAccOpendistroProvider.ConfigureContextFunc
	testAccOpendistroProvider.ConfigureContextFunc = func(c context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		err := d.Set("url", os.Getenv("OPENSEARCH_URL"))
		if err != nil {
			return nil, diag.FromErr(err)
		}
		return opendistroOriginalConfigureFunc(c, d)
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("OPENSEARCH_URL"); v == "" {
		t.Fatal("OPENSEARCH_URL must be set for acceptance tests")
	}
}

// Given:
// 1. invalid username and password and healthcheck is false
//
// this tests that an error is returned by getOpenSearchClient for invalid credentials
func TestInvalidCredentials(t *testing.T) {
	parsedUrl, _ := url.Parse("http://127.0.0.1:9200")
	testConfig := &ProviderConf{
		username:           "1234",
		password:           "1234",
		healthchecking:     false,
		rawUrl:             "http://127.0.0.1:9200",
		sniffing:           false,
		parsedUrl:          parsedUrl,
		pingTimeoutSeconds: 10,
	}
	_, err := getOpenSearchClient(testConfig)

	if err == nil {
		t.Error("Expected an error to be returned for invalid credentials")
	}
}
