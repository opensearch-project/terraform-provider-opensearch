package provider

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ServerFlavor int64

// OpenSearch
const (
	Unknown ServerFlavor = iota
	OpenSearch
	Default = 2
)

var awsUrlRegexp = regexp.MustCompile(`([a-z0-9-]+).es.amazonaws.com$`)
var awsOpensearchServerlessUrlRegexp = regexp.MustCompile(`([a-z0-9-]+).aoss.amazonaws.com$`)
var minimalOpensearchServerlessVersion = "2.0.0"

type ProviderConf struct {
	rawUrl                  string
	insecure                bool
	sniffing                bool
	healthchecking          bool
	cacertFile              string
	username                string
	password                string
	token                   string
	tokenName               string
	parsedUrl               *url.URL
	signAWSRequests         bool
	osVersion               string
	pingTimeoutSeconds      int
	awsRegion               string
	awsAssumeRoleArn        string
	awsAssumeRoleExternalID string
	awsAccessKeyId          string
	awsSecretAccessKey      string
	awsSessionToken         string
	awsSig4Service          string
	awsProfile              string
	certPemPath             string
	keyPemPath              string
	hostOverride            string
	proxy                   string
	// determined after connecting to the server
	flavor ServerFlavor
	// New opensearch-go/v4 client (Phase 2 migration)
	opensearchClient *OpenSearchClient
}

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_URL", nil),
				Description: "OpenSearch URL",
			},
			"sniff": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_SNIFF", false),
				Description: "Set the node sniffing option for the OpenSearch client. Client won't work with sniffing if nodes are not routable.",
			},
			"healthcheck": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_HEALTH", true),
				Description: "Set the client healthcheck option for the OpenSearch client. Healthchecking is designed for direct access to the cluster.",
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_USERNAME", nil),
				Description: "Username for OpenSearch basic auth",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_PASSWORD", nil),
				Description: "Password for OpenSearch basic auth",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_TOKEN", nil),
				Description: "Authorization token for OpenSearch",
			},
			"token_name": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_TOKEN_NAME", "ApiKey"),
				Description: "Authorization token name for OpenSearch",
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_INSECURE", false),
				Description: "Disable SSL verification",
			},
			"cacert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_CACERT", nil),
				Description: "Path to a CA certificate file to verify the server's certificate",
			},
			"client_cert_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envFallbackDefault("OPENSEARCH_CLIENT_CERT_PATH", "OS_CLIENT_CERTIFICATE_PATH"),
				Description: "Path to a client certificate file",
			},
			"client_key_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envFallbackDefault("OPENSEARCH_CLIENT_KEY_PATH", "OS_CLIENT_KEY_PATH"),
				Description: "Path to a client key file",
			},
			"sign_aws_requests": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_SIGN_AWS", true),
				Description: "Enable AWS request signing",
			},
			"opensearch_version": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_VERSION", nil),
				Description: "OpenSearch version",
			},
			"ping_timeout_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_PING_TIMEOUT", 5),
				Description: "Timeout for OpenSearch pings in seconds",
			},
			"version_ping_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_PING_TIMEOUT", nil),
				Description: "Version ping timeout in seconds. Deprecated: use ping_timeout_seconds instead.",
				Deprecated:  "Use ping_timeout_seconds instead",
			},
			"aws_region": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AWS_REGION", nil),
				Description: "AWS region for request signing",
			},
			"aws_signature_service": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_AWS_SIGNATURE_SERVICE", nil),
				Description: "AWS service name used in the credential scope of signed requests. Deprecated: AWS service is now auto-detected from the URL. Use 'aoss' for OpenSearch Serverless.",
				Deprecated:  "AWS service is now auto-detected from the URL. Set opensearch_version for Serverless.",
			},
			"aws_assume_role_arn": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_AWS_ASSUME_ROLE_ARN", nil),
				Description: "AWS IAM Role to assume for request signing",
			},
			"aws_assume_role_external_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_AWS_ASSUME_ROLE_EXTERNAL_ID", nil),
				Description: "AWS IAM Role external ID for request signing",
			},
			"aws_access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AWS_ACCESS_KEY_ID", nil),
				Description: "AWS access key for request signing",
			},
			"aws_secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AWS_SECRET_ACCESS_KEY", nil),
				Description: "AWS secret key for request signing",
			},
			"aws_session_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AWS_SESSION_TOKEN", nil),
				Description: "AWS session token for request signing",
			},
			"aws_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AWS_SESSION_TOKEN", nil),
				Description: "AWS session token for request signing. Deprecated: use aws_session_token instead.",
				Deprecated:  "Use aws_session_token instead",
			},
			"aws_profile": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AWS_PROFILE", nil),
				Description: "AWS profile for request signing",
			},
			"host_override": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_HOST_OVERRIDE", nil),
				Description: "Override the host header for requests",
			},
			"proxy": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OPENSEARCH_PROXY", nil),
				Description: "Proxy URL for requests",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"opensearch_audit_config":              resourceOpenSearchAuditConfig(),
			"opensearch_cluster_settings":          resourceOpensearchClusterSettings(),
			"opensearch_composable_index_template": resourceOpensearchComposableIndexTemplate(),
			"opensearch_component_template":        resourceOpensearchComponentTemplate(),
			"opensearch_dashboard_tenant":          resourceOpenSearchDashboardTenant(),
			"opensearch_dashboard_object":          resourceOpensearchDashboardObject(),
			"opensearch_data_stream":               resourceOpensearchDataStream(),
			"opensearch_index_template":            resourceOpensearchIndexTemplate(),
			"opensearch_index":                     resourceOpensearchIndex(),
			"opensearch_ingest_pipeline":           resourceOpensearchIngestPipeline(),
			"opensearch_role":                      resourceOpenSearchRole(),
			"opensearch_roles_mapping":             resourceOpenSearchRolesMapping(),
			"opensearch_script":                    resourceOpensearchScript(),
			"opensearch_snapshot_repository":       resourceOpensearchSnapshotRepository(),
			"opensearch_sm_policy":                 resourceOpenSearchSMPolicy(),
			"opensearch_user":                      resourceOpenSearchUser(),
			"opensearch_channel_configuration":     resourceOpenSearchChannelConfiguration(),
			"opensearch_ism_policy_mapping":        resourceOpenSearchISMPolicyMapping(),
			"opensearch_ism_policy":                resourceOpenSearchISMPolicy(),
			"opensearch_monitor":                   resourceOpenSearchMonitor(),
			"opensearch_anomaly_detection":         resourceOpenSearchAnomalyDetection(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"opensearch_host": dataSourceOpensearchHost(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	rawUrl := d.Get("url").(string)
	_, err := url.Parse(rawUrl)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	if d.Get("username").(string) != "" {
		parsedUrl.User = url.UserPassword(d.Get("username").(string), d.Get("password").(string))
	}

	conf := &ProviderConf{
		rawUrl:                  rawUrl,
		parsedUrl:               parsedUrl,
		sniffing:                d.Get("sniff").(bool),
		healthchecking:          d.Get("healthcheck").(bool),
		username:                d.Get("username").(string),
		password:                d.Get("password").(string),
		token:                   d.Get("token").(string),
		tokenName:               d.Get("token_name").(string),
		insecure:                d.Get("insecure").(bool),
		cacertFile:              d.Get("cacert_file").(string),
		signAWSRequests:         d.Get("sign_aws_requests").(bool),
		osVersion:               d.Get("opensearch_version").(string),
		pingTimeoutSeconds:      resolveIntField(d, "ping_timeout_seconds", "version_ping_timeout", 5),
		awsRegion:               d.Get("aws_region").(string),
		awsAssumeRoleArn:        d.Get("aws_assume_role_arn").(string),
		awsAssumeRoleExternalID: d.Get("aws_assume_role_external_id").(string),
		awsAccessKeyId:          d.Get("aws_access_key").(string),
		awsSecretAccessKey:      d.Get("aws_secret_key").(string),
		awsSessionToken:         resolveStringField(d, "aws_session_token", "aws_token"),
		awsSig4Service:          d.Get("aws_signature_service").(string),
		awsProfile:              d.Get("aws_profile").(string),
		certPemPath:             d.Get("client_cert_path").(string),
		keyPemPath:              d.Get("client_key_path").(string),
		hostOverride:            d.Get("host_override").(string),
		proxy:                   d.Get("proxy").(string),
	}

	return conf, diags
}

// getOpenSearchClient returns the opensearch-go/v4 client, creating it if necessary
// This is the new client getter for Phase 2 migration
func getOpenSearchClient(conf *ProviderConf) (*OpenSearchClient, error) {
	// Return existing client if already created
	if conf.opensearchClient != nil {
		return conf.opensearchClient, nil
	}

	// Create new client
	client, err := NewOpenSearchClient(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenSearch client: %w", err)
	}

	// Perform version detection if not already set
	if conf.osVersion == "" {
		log.Printf("[INFO] Getting server info to determine version %s with timeout %ds", conf.rawUrl, conf.pingTimeoutSeconds)
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.pingTimeoutSeconds)*time.Second)
		defer cancel()

		// Use the new client's Info method to get version
		info, err := client.Client.Info(ctx, nil)
		if err != nil {
			// Check for specific error types
			if os.IsTimeout(err) {
				return nil, fmt.Errorf("timeout after %d seconds while getting info from '%s' to determine server version, please consider setting 'opensearch_version' to avoid this lookup", conf.pingTimeoutSeconds, conf.rawUrl)
			}
			return nil, fmt.Errorf("failed to get OpenSearch info: %w", err)
		}

		conf.osVersion = info.Version.Number
		log.Printf("[INFO] OpenSearch version %s (distribution: %s)", info.Version.Number, info.Version.Distribution)

		// Determine flavor based on distribution
		switch info.Version.Distribution {
		case "opensearch":
			conf.flavor = OpenSearch
		default:
			conf.flavor = Unknown
		}
	}

	// Store client in config for reuse
	conf.opensearchClient = client
	return client, nil
}

// resolveStringField returns the value of newField if non-empty, otherwise falls back to oldField.
// This supports backward compatibility when provider fields are renamed.
func resolveStringField(d *schema.ResourceData, newField, oldField string) string {
	if v := d.Get(newField).(string); v != "" {
		return v
	}
	return d.Get(oldField).(string)
}

// resolveIntField returns the value of newField if non-zero, otherwise falls back to oldField.
// If both are zero, returns the provided default value.
func resolveIntField(d *schema.ResourceData, newField, oldField string, defaultVal int) int {
	if v := d.Get(newField).(int); v != 0 {
		return v
	}
	if v := d.Get(oldField).(int); v != 0 {
		return v
	}
	return defaultVal
}

// envFallbackDefault returns a DefaultFunc that checks the primary env var first,
// then falls back to the legacy env var. If neither is set, returns nil (no default).
func envFallbackDefault(primaryEnvVar, legacyEnvVar string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(primaryEnvVar); v != "" {
			return v, nil
		}
		if v := os.Getenv(legacyEnvVar); v != "" {
			return v, nil
		}
		return nil, nil
	}
}
