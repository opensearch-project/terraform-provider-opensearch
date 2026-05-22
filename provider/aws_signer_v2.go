package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/opensearch-project/opensearch-go/v4/signer"
	"github.com/opensearch-project/opensearch-go/v4/signer/awsv2"
)

// newAWSSigner creates an AWS SigV4 signer for OpenSearch requests using AWS SDK v2
func newAWSSigner(conf *ProviderConf) (signer.Signer, error) {
	ctx := context.Background()

	// Determine service name (es for OpenSearch Service, aoss for Serverless)
	service := conf.awsSig4Service
	if service == "" {
		service = "es" // Default to OpenSearch Service
	}

	// Build config options
	var opts []func(*config.LoadOptions) error

	// Set region
	if conf.awsRegion != "" {
		opts = append(opts, config.WithRegion(conf.awsRegion))
	}

	// Handle credentials in priority order:
	// 1. Access keys take priority
	// 2. Assume role configuration
	// 3. Named profile
	// 4. Default credential chain (env, ec2, etc.)
	if conf.awsAccessKeyId != "" {
		// Use static credentials
		creds := credentials.NewStaticCredentialsProvider(
			conf.awsAccessKeyId,
			conf.awsSecretAccessKey,
			conf.awsSessionToken,
		)
		opts = append(opts, config.WithCredentialsProvider(creds))
	} else if conf.awsAssumeRoleArn != "" {
		// Load base config first, then create assume role provider
		baseCfg, err := config.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("error loading AWS config for assume role: %w", err)
		}

		// Create STS client for assume role
		stsClient := sts.NewFromConfig(baseCfg)

		// Create assume role credentials provider
		var assumeRoleOpts []func(*stscreds.AssumeRoleOptions)
		if conf.awsAssumeRoleExternalID != "" {
			assumeRoleOpts = append(assumeRoleOpts, func(o *stscreds.AssumeRoleOptions) {
				o.ExternalID = &conf.awsAssumeRoleExternalID
			})
		}

		creds := stscreds.NewAssumeRoleProvider(stsClient, conf.awsAssumeRoleArn, assumeRoleOpts...)
		opts = append(opts, config.WithCredentialsProvider(creds))
	} else if conf.awsProfile != "" {
		// Use named profile
		opts = append(opts, config.WithSharedConfigProfile(conf.awsProfile))
	}
	// If none of the above, LoadDefaultConfig will use the default credential chain

	// Load the AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: %w", err)
	}

	// Create the signer with the appropriate service
	var signer signer.Signer
	if service == "aoss" {
		// OpenSearch Serverless uses "aoss" service name
		signer, err = awsv2.NewSignerWithService(cfg, service)
	} else {
		// Standard OpenSearch Service uses "es" service name
		signer, err = awsv2.NewSigner(cfg)
	}

	if err != nil {
		return nil, fmt.Errorf("error creating AWS signer: %w", err)
	}

	return signer, nil
}
