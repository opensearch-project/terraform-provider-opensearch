package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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
// this tests that 401 error is returned by getClient
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
	_, err := getClient(testConfig)

	errString := "HTTP 401 Unauthorized: Please ensure that the correct credentials are being used to access the cluster"
	if err.Error() != errString {
		t.Errorf("Error thrown should be %s", errString)
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
// 1. AWS credentials are specified via environment variables
// 2. An AWS role ARN and a session name are specified via the provider configuration
//
// This tests that: the configured session name is applied to the plain
// AssumeRole call (not just the web identity path), matching the AWS CLI.
func TestAWSCredsAssumeRoleSessionName(t *testing.T) {
	envAccessKeyID := "ENV_ACCESS_KEY"
	testRegion := "us-east-1"
	assumeRoleArn := "arn:aws:iam::123456789012:role/demo/TestAR"
	assumeRoleSessionName := "my-session-name"
	assumeRoleAccessKeyID := "ASIAIOSFODNN7EXAMPLE"

	os.Setenv("AWS_ACCESS_KEY_ID", envAccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", "ENV_SECRET")

	server := mockServer{
		ResponseFixturePath:     "./test-fixtures/api_assume_role_response.xml",
		ExpectedAccessKeyId:     envAccessKeyID,
		ExpectedRoleArn:         assumeRoleArn,
		ExpectedRoleSessionName: assumeRoleSessionName,
	}

	server.Start(t)
	defer server.Stop()

	testConfig := &ProviderConf{
		awsAssumeRoleArn:         assumeRoleArn,
		awsAssumeRoleSessionName: assumeRoleSessionName,
	}

	creds := getCreds(t, testRegion, testConfig, server.Endpoint)

	if creds.AccessKeyID != assumeRoleAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", assumeRoleAccessKeyID, creds.AccessKeyID)
	}

	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
}

// Given:
// 1. Only one of the web identity role ARN / token file is configured
//
// This tests that: awsCredentialWarnings emits a warning for the partial web
// identity configuration, and stays silent when both or neither are set.
func TestAWSCredsWebIdentityPartialConfigWarning(t *testing.T) {
	cases := []struct {
		name        string
		conf        *ProviderConf
		wantWarning bool
	}{
		{
			name:        "only role arn",
			conf:        &ProviderConf{awsWebIdentityRoleArn: "arn:aws:iam::123456789012:role/demo/TestWebIdentity"},
			wantWarning: true,
		},
		{
			name:        "only token file",
			conf:        &ProviderConf{awsWebIdentityTokenFile: "./test-fixtures/web_identity_token"},
			wantWarning: true,
		},
		{
			name: "both set",
			conf: &ProviderConf{
				awsWebIdentityRoleArn:   "arn:aws:iam::123456789012:role/demo/TestWebIdentity",
				awsWebIdentityTokenFile: "./test-fixtures/web_identity_token",
			},
			wantWarning: false,
		},
		{
			name:        "neither set",
			conf:        &ProviderConf{},
			wantWarning: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diags := awsCredentialWarnings(tc.conf)
			if tc.wantWarning && !diags.HasError() && len(diags) == 0 {
				t.Errorf("expected a warning diagnostic, got none")
			}
			if !tc.wantWarning && len(diags) != 0 {
				t.Errorf("expected no diagnostics, got %v", diags)
			}
			if tc.wantWarning {
				if len(diags) != 1 || diags[0].Severity != diag.Warning {
					t.Errorf("expected exactly one warning diagnostic, got %v", diags)
				}
			}
		})
	}
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

// Given:
// 1. A web identity role ARN and token file are specified via the provider configuration
//
// This tests that: we assume the role via AssumeRoleWithWebIdentity using the token file.
func TestAWSCredsAssumeRoleWithWebIdentity(t *testing.T) {
	testRegion := "us-east-1"
	webIdentityRoleArn := "arn:aws:iam::123456789012:role/demo/TestWebIdentity"
	webIdentityAccessKeyID := "ASIAWEBIDENTITYEXAMPLE"

	server := mockServer{
		ResponseFixturePath:      "./test-fixtures/api_assume_role_with_web_identity_response.xml",
		ExpectedRoleArn:          webIdentityRoleArn,
		ExpectedWebIdentityToken: readTokenFixture(t, "./test-fixtures/web_identity_token"),
	}

	server.Start(t)
	defer server.Stop()

	testConfig := &ProviderConf{
		awsWebIdentityRoleArn:   webIdentityRoleArn,
		awsWebIdentityTokenFile: "./test-fixtures/web_identity_token",
	}

	creds := getCreds(t, testRegion, testConfig, server.Endpoint)

	if creds.AccessKeyID != webIdentityAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", webIdentityAccessKeyID, creds.AccessKeyID)
	}
}

// Given:
// 1. AWS_ROLE_ARN and AWS_WEB_IDENTITY_TOKEN_FILE are set via environment variables (as EKS IRSA injects them)
// 2. No AWS configuration is provided to the provider
//
// This tests that: the web identity provider args default from the standard AWS
// environment variables, so IRSA autodiscovery works with no configuration.
func TestAWSCredsWebIdentityAutodiscovery(t *testing.T) {
	testRegion := "us-east-1"
	webIdentityRoleArn := "arn:aws:iam::123456789012:role/demo/TestWebIdentity"
	webIdentityAccessKeyID := "ASIAWEBIDENTITYEXAMPLE"

	server := mockServer{
		ResponseFixturePath:      "./test-fixtures/api_assume_role_with_web_identity_response.xml",
		ExpectedRoleArn:          webIdentityRoleArn,
		ExpectedWebIdentityToken: readTokenFixture(t, "./test-fixtures/web_identity_token"),
	}

	server.Start(t)
	defer server.Stop()

	os.Setenv("AWS_ROLE_ARN", webIdentityRoleArn)
	os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "./test-fixtures/web_identity_token")

	// With no explicit web identity configuration, resolveAWSWebIdentityEnv
	// populates the fields from the standard AWS_ROLE_ARN /
	// AWS_WEB_IDENTITY_TOKEN_FILE env vars, exactly as it does at runtime.
	testConfig := &ProviderConf{}
	resolveAWSWebIdentityEnv(testConfig)

	creds := getCreds(t, testRegion, testConfig, server.Endpoint)

	if creds.AccessKeyID != webIdentityAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", webIdentityAccessKeyID, creds.AccessKeyID)
	}

	os.Unsetenv("AWS_ROLE_ARN")
	os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
}

// Given:
// 1. AWS_ROLE_ARN and AWS_WEB_IDENTITY_TOKEN_FILE are set via environment variables
// 2. An explicit aws_profile is configured
//
// This tests that: the explicit profile suppresses the env-sourced web identity,
// mirroring the AWS CLI (botocore) disable_env_vars rule. resolveAWSWebIdentityEnv
// must leave the web identity fields empty so the profile credentials are used.
func TestAWSCredsWebIdentityEnvSuppressedByProfile(t *testing.T) {
	webIdentityRoleArn := "arn:aws:iam::123456789012:role/demo/TestWebIdentity"

	os.Setenv("AWS_ROLE_ARN", webIdentityRoleArn)
	os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "./test-fixtures/web_identity_token")

	testConfig := &ProviderConf{
		awsProfile: "testing",
	}
	resolveAWSWebIdentityEnv(testConfig)

	if testConfig.awsWebIdentityRoleArn != "" {
		t.Errorf("expected web identity role arn to be suppressed by explicit profile, got %s", testConfig.awsWebIdentityRoleArn)
	}
	if testConfig.awsWebIdentityTokenFile != "" {
		t.Errorf("expected web identity token file to be suppressed by explicit profile, got %s", testConfig.awsWebIdentityTokenFile)
	}

	os.Unsetenv("AWS_ROLE_ARN")
	os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
}

// Given:
// 1. A web identity role ARN and token file are configured
// 2. aws_assume_role_arn is also configured
//
// This tests that: the web identity credentials are resolved first and then
// used to sign the AssumeRole call for the second role (role chaining). We
// assert on the ordering of STS actions and that the second call is signed with
// the access key returned by the first (web identity) call.
func TestAWSCredsWebIdentityChainedToAssumeRole(t *testing.T) {
	testRegion := "us-east-1"
	webIdentityRoleArn := "arn:aws:iam::123456789012:role/demo/TestWebIdentity"
	chainedRoleArn := "arn:aws:iam::123456789012:role/demo/TestChainedRole"
	webIdentityAccessKeyID := "ASIAWEBIDENTITYEXAMPLE"
	chainedAccessKeyID := "ASIACHAINEDROLEEXAMPLE"
	expectedToken := readTokenFixture(t, "./test-fixtures/web_identity_token")

	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("Error while parsing form: %v", err)
		}
		action := r.PostForm.Get("Action")
		actions = append(actions, action)

		var fixture string
		switch action {
		case "AssumeRoleWithWebIdentity":
			if r.PostForm.Get("RoleArn") != webIdentityRoleArn {
				t.Errorf("web identity call: expected RoleArn %s, got %s", webIdentityRoleArn, r.PostForm.Get("RoleArn"))
			}
			if r.PostForm.Get("WebIdentityToken") != expectedToken {
				t.Errorf("web identity call: expected token %s, got %s", expectedToken, r.PostForm.Get("WebIdentityToken"))
			}
			fixture = "./test-fixtures/api_assume_role_with_web_identity_response.xml"
		case "AssumeRole":
			if r.PostForm.Get("RoleArn") != chainedRoleArn {
				t.Errorf("assume role call: expected RoleArn %s, got %s", chainedRoleArn, r.PostForm.Get("RoleArn"))
			}
			// The chained AssumeRole call must be signed with the web identity
			// credentials, not the ambient environment. Verify the SigV4
			// Authorization header carries the web identity access key.
			auth := r.Header.Get("Authorization")
			if !strings.Contains(auth, webIdentityAccessKeyID) {
				t.Errorf("assume role call should be signed with web identity key %s, authorization header was %s", webIdentityAccessKeyID, auth)
			}
			fixture = "./test-fixtures/api_chained_assume_role_response.xml"
		default:
			t.Errorf("unexpected STS action: %s", action)
			return
		}

		response, err := os.ReadFile(fixture)
		if err != nil {
			t.Errorf("Error while reading mockResponse %v", err)
		}
		w.WriteHeader(http.StatusOK)
		if _, err = w.Write(response); err != nil {
			t.Errorf("Error while writing mock server response %v", err)
		}
	}))
	defer server.Close()

	testConfig := &ProviderConf{
		awsAssumeRoleArn:        chainedRoleArn,
		awsWebIdentityRoleArn:   webIdentityRoleArn,
		awsWebIdentityTokenFile: "./test-fixtures/web_identity_token",
	}

	creds := getCreds(t, testRegion, testConfig, server.URL)

	if creds.AccessKeyID != chainedAccessKeyID {
		t.Errorf("access key id should have been %s (we got %s)", chainedAccessKeyID, creds.AccessKeyID)
	}

	if len(actions) != 2 || actions[0] != "AssumeRoleWithWebIdentity" || actions[1] != "AssumeRole" {
		t.Errorf("expected STS actions [AssumeRoleWithWebIdentity AssumeRole], got %v", actions)
	}
}

func readTokenFixture(t *testing.T, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading token fixture %s: %v", path, err)
	}
	return string(data)
}

func getCreds(t *testing.T, region string, config *ProviderConf, endpoint string) credentials.Value {
	s := awsSession(region, config, endpoint)
	if s == nil {
		t.Fatalf("awsSession returned nil")
		return credentials.Value{}
	}
	if s.Config.Credentials == nil {
		t.Fatalf("awsSession returned session with nil credentials")
		return credentials.Value{}
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
	ResponseFixturePath      string
	ExpectedAccessKeyId      string
	ExpectedRoleArn          string
	ExpectedExternalId       string
	ExpectedWebIdentityToken string
	ExpectedRoleSessionName  string
	Endpoint                 string
	server                   *httptest.Server
}

func (s *mockServer) Start(t *testing.T) {
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if s.ExpectedAccessKeyId != "" {
			auth := r.Header.Get("Authorization")
			if !strings.Contains(auth, s.ExpectedAccessKeyId) {
				t.Errorf("Could not find expected access key id %s in authorization header %s", s.ExpectedAccessKeyId, auth)
			}
		}

		err := r.ParseForm()
		if err != nil {
			t.Errorf("Error while parsing form: %v", err)
		}

		if r.PostForm.Get("RoleArn") != s.ExpectedRoleArn {
			t.Errorf("expected RoleArn to be equal to %s, but got %s", s.ExpectedRoleArn, r.PostForm.Get("RoleArn"))
		}

		if s.ExpectedExternalId != "" && r.PostForm.Get("ExternalId") != s.ExpectedExternalId {
			t.Errorf("expected ExternalId to be equal to %s, but got %s", s.ExpectedExternalId, r.PostForm.Get("ExternalId"))
		}

		if s.ExpectedWebIdentityToken != "" && r.PostForm.Get("WebIdentityToken") != s.ExpectedWebIdentityToken {
			t.Errorf("expected WebIdentityToken to be equal to %s, but got %s", s.ExpectedWebIdentityToken, r.PostForm.Get("WebIdentityToken"))
		}

		if s.ExpectedRoleSessionName != "" && r.PostForm.Get("RoleSessionName") != s.ExpectedRoleSessionName {
			t.Errorf("expected RoleSessionName to be equal to %s, but got %s", s.ExpectedRoleSessionName, r.PostForm.Get("RoleSessionName"))
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
