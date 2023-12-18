package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws/credentials"
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
		err := d.Set("url", "http://admin:myStrongPassword123!@127.0.0.1:9200")
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
// 1. AWS credentials are specified via environment variables
// 2. aws access key and secret access key are specified via the provider configuration
// 3. a named profile is specified via the provider config
//
// this tests that:  the configured provider access key / secret key are used over the other options (ie: #2)
func TestAWSCredsManualKey(t *testing.T) {
	envAccessKeyID := "ENV_ACCESS_KEY"
	testRegion := "us-east-1"
	manualAccessKeyID := "MANUAL_ACCESS_KEY"
	namedProfile := "testing"

	os.Setenv("AWS_ACCESS_KEY_ID", envAccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", "ENV_SECRET")

	// first, check that if we set aws_profile with aws_access_key_id - the latter takes precedence
	testConfig := &ProviderConf{
		awsAccessKeyId:     manualAccessKeyID,
		awsSecretAccessKey: "MANUAL_SECRET_KEY",
		awsProfile:         namedProfile,
	}

	creds := getCreds(t, testRegion, testConfig, "")

	if creds.AccessKeyID != manualAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", manualAccessKeyID, creds.AccessKeyID)
	}
}

// Given:
// 1. AWS credentials are specified via environment variables
// 2. a named profile is specified via the provider config
//
// this tests that:  the named profile credentials are used over the env vars
func TestAWSCredsNamedProfile(t *testing.T) {
	envAccessKeyID := "ENV_ACCESS_KEY"
	testRegion := "us-east-1"
	namedProfile := "testing"
	profileAccessKeyID := "PROFILE_ACCESS_KEY"

	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", "./test-fixtures/test_aws_credentials") // set credentials file so we can ensure the profile we want to test exists
	os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	os.Setenv("AWS_ACCESS_KEY_ID", envAccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", "ENV_SECRET")

	testConfig := &ProviderConf{
		awsProfile: namedProfile,
	}

	creds := getCreds(t, testRegion, testConfig, "")

	if creds.AccessKeyID != profileAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", profileAccessKeyID, creds.AccessKeyID)
	}

	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
	os.Unsetenv("AWS_SDK_LOAD_CONFIG")
}

// Given:
// 1. AWS credentials are specified via environment variables
// 2. No configuration provided to the provider
//
// This tests that: we get the credentials from the environment variables (ie: from the default credentials provider chain)
func TestAWSCredsEnv(t *testing.T) {
	envAccessKeyID := "ENV_ACCESS_KEY"
	testRegion := "us-east-1"

	os.Setenv("AWS_ACCESS_KEY_ID", envAccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", "ENV_SECRET")

	testConfig := &ProviderConf{}

	creds := getCreds(t, testRegion, testConfig, "")

	if creds.AccessKeyID != envAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", envAccessKeyID, creds.AccessKeyID)
	}

	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
}

// Given:
// 1. AWS profile is specified via environment variables
// 2. No configuration provided to the provider
//
// This tests that: we get the credentials from the environment variables (ie: from the default credentials provider chain)
func TestAWSCredsEnvNamedProfile(t *testing.T) {
	namedProfile := "testing"
	testRegion := "us-east-1"
	profileAccessKeyID := "PROFILE_ACCESS_KEY"

	os.Setenv("AWS_PROFILE", namedProfile)
	os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", "./test-fixtures/test_aws_credentials") // set credentials file so we can ensure the profile we want to test exists

	testConfig := &ProviderConf{}

	creds := getCreds(t, testRegion, testConfig, "")

	if creds.AccessKeyID != profileAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", profileAccessKeyID, creds.AccessKeyID)
	}
	os.Unsetenv("AWS_PROFILE")
	os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
	os.Unsetenv("AWS_SDK_LOAD_CONFIG")
}

// Given:
// 1. AWS credentials are specified via environment variables
// 2. An AWS role ARN and External ID are specified via the provider configuration
//
// This tests that: we can get the credentials after having assumed the given role from the specified AWS credentials.
func TestAWSCredsAssumeRole(t *testing.T) {
	envAccessKeyID := "ENV_ACCESS_KEY"
	testRegion := "us-east-1"
	assumeRoleArn := "arn:aws:iam::123456789012:role/demo/TestAR"
	assumeRoleExternalId := "secret_id"
	assumeRoleAccessKeyID := "ASIAIOSFODNN7EXAMPLE"

	os.Setenv("AWS_ACCESS_KEY_ID", envAccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", "ENV_SECRET")

	server := mockServer{
		ResponseFixturePath: "./test-fixtures/api_assume_role_response.xml",
		ExpectedAccessKeyId: envAccessKeyID,
		ExpectedRoleArn:     assumeRoleArn,
		ExpectedExternalId:  assumeRoleExternalId,
	}

	server.Start(t)
	defer server.Stop()

	testConfig := &ProviderConf{
		awsAssumeRoleArn:        assumeRoleArn,
		awsAssumeRoleExternalID: assumeRoleExternalId,
	}

	creds := getCreds(t, testRegion, testConfig, server.Endpoint)

	if creds.AccessKeyID != assumeRoleAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", assumeRoleAccessKeyID, creds.AccessKeyID)
	}

	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
}

// Given:
// 1. An AWS profile, role ARN and External ID are specified via the provider configuration
//
// This tests that: we can get the credentials after having assumed the given role from the specified profile.
func TestAWSCredsAssumeRoleFromProfile(t *testing.T) {
	testRegion := "us-east-1"
	assumeRoleArn := "arn:aws:iam::123456789012:role/demo/TestAR"
	assumeRoleExternalId := "secret_id"
	namedProfile := "testing"
	assumeRoleAccessKeyID := "ASIAIOSFODNN7EXAMPLE"

	os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", "./test-fixtures/test_aws_credentials") // set credentials file so we can ensure the profile we want to test exists

	server := mockServer{
		ResponseFixturePath: "./test-fixtures/api_assume_role_response.xml",
		ExpectedAccessKeyId: "PROFILE_ACCESS_KEY", // from the test-fixture config file
		ExpectedRoleArn:     assumeRoleArn,
		ExpectedExternalId:  assumeRoleExternalId,
	}

	server.Start(t)
	defer server.Stop()

	testConfig := &ProviderConf{
		awsAssumeRoleArn:        assumeRoleArn,
		awsAssumeRoleExternalID: assumeRoleExternalId,
		awsProfile:              namedProfile,
	}

	creds := getCreds(t, testRegion, testConfig, server.Endpoint)

	if creds.AccessKeyID != assumeRoleAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", assumeRoleAccessKeyID, creds.AccessKeyID)
	}

	os.Unsetenv("AWS_SDK_LOAD_CONFIG")
	os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
}

// Given:
// 1. An AWS role ARN and External ID are specified via the provider configuration
//
// This tests that: we can get the credentials after having assumed the given role from the default profile.
func TestAWSCredsAssumeRoleFromDefaultProfile(t *testing.T) {
	testRegion := "us-east-1"
	assumeRoleArn := "arn:aws:iam::123456789012:role/demo/TestAR"
	assumeRoleExternalId := "secret_id"
	assumeRoleAccessKeyID := "ASIAIOSFODNN7EXAMPLE"

	os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", "./test-fixtures/test_aws_credentials") // set credentials file so we can ensure the profile we want to test exists

	server := mockServer{
		ResponseFixturePath: "./test-fixtures/api_assume_role_response.xml",
		ExpectedAccessKeyId: "PROFILE_DEFAULT_ACCESS_KEY", // from the test-fixture config file
		ExpectedRoleArn:     assumeRoleArn,
		ExpectedExternalId:  assumeRoleExternalId,
	}

	server.Start(t)
	defer server.Stop()

	testConfig := &ProviderConf{
		awsAssumeRoleArn:        assumeRoleArn,
		awsAssumeRoleExternalID: assumeRoleExternalId,
	}

	creds := getCreds(t, testRegion, testConfig, server.Endpoint)

	if creds.AccessKeyID != assumeRoleAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", assumeRoleAccessKeyID, creds.AccessKeyID)
	}

	os.Unsetenv("AWS_SDK_LOAD_CONFIG")
	os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
}

func getCreds(t *testing.T, region string, config *ProviderConf, endpoint string) credentials.Value {
	s := awsSession(region, config, endpoint)
	if s == nil {
		t.Fatalf("awsSession returned nil")
	}
	creds, err := s.Config.Credentials.Get()
	if err != nil {
		t.Fatalf("Failed fetching credentials: %v", err)
	}
	return creds
}

// Given:
// 1. A proxy URL is specified.
// 2. No additional AWS configuration is provided to the provider
//
// This tests that: the proxy value is set for the transport. Note we cannot get the credentials, because that requires connecting to AWS.
func TestAWSSocksProxy(t *testing.T) {
	testRegion := "us-east-1"

	testConfig := map[string]interface{}{
		"proxy": "socks://127.0.0.1:8080",
	}

	testConfigData := schema.TestResourceDataRaw(t, Provider().Schema, testConfig)

	conf := &ProviderConf{
		proxy: testConfigData.Get("proxy").(string),
	}
	s := awsSession(testRegion, conf, "")
	if s == nil {
		t.Fatalf("awsSession returned nil")
	}
}

type mockServer struct {
	ResponseFixturePath string
	ExpectedAccessKeyId string
	ExpectedRoleArn     string
	ExpectedExternalId  string
	Endpoint            string
	server              *httptest.Server
}

func (s *mockServer) Start(t *testing.T) {
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		auth := r.Header.Get("Authorization")
		if !strings.Contains(auth, s.ExpectedAccessKeyId) {
			t.Errorf("Could not find expected access key id %s in authorization header %s", s.ExpectedAccessKeyId, auth)
		}

		err := r.ParseForm()
		if err != nil {
			t.Errorf("Error while parsing form: %v", err)
		}

		if r.PostForm.Get("RoleArn") != s.ExpectedRoleArn {
			t.Errorf("expected RoleArn to be equal to %s, but got %s", s.ExpectedRoleArn, r.PostForm.Get("RoleArn"))
		}

		if r.PostForm.Get("ExternalId") != s.ExpectedExternalId {
			t.Errorf("expected ExternalId to be equal to %s, but got %s", s.ExpectedExternalId, r.PostForm.Get("ExternalId"))
		}

		response, err := os.ReadFile(s.ResponseFixturePath)
		if err != nil {
			t.Errorf("Error while reading mockResponse %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(response)
		if err != nil {
			t.Errorf("Error while writing mock server response %v", err)
		}
	}))

	s.Endpoint = s.server.URL
}

func (s *mockServer) Stop() {
	s.server.Close()
}
