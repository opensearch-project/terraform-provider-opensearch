package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchOpenDistroChannelConfiguration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchChannelConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchOpenDistroWebhookChannelConfiguration,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.webhook_channel_configuration"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccOpensearchOpenDistroSlackChannelConfiguration,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.slack_channel_configuration"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccOpensearchOpenDistroChimeChannelConfiguration,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.chime_channel_configuration"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccOpensearchOpenDistroSnsChannelConfiguration,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.sns_channel_configuration"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccOpensearchOpenDistroSmtpEmailChannelConfiguration,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.smtp_account_configuration"),
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.email_channel_configuration"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccOpensearchOpenDistroSesEmailChannelConfiguration,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.ses_account_configuration"),
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.email_channel_configuration"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccOpensearchOpenDistroSesEmailGroupChannelConfiguration,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.ses_account_configuration"),
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.email_group_configuration"),
					testCheckOpensearchChannelConfigurationExists("opensearch_channel_configuration.email_channel_configuration"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckOpensearchChannelConfigurationExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No channel configuration ID is set")
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchOpenDistroGetChannelConfiguration(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchChannelConfigurationDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_channel_configuration" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchOpenDistroGetChannelConfiguration(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("ChannelConfiguration %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchOpenDistroWebhookChannelConfiguration = `
resource "opensearch_channel_configuration" "webhook_channel_configuration" {
  body = <<EOF
{
  "id": "sample-webhook-id",
  "name": "sample-name",
  "config": {
    "name": "Sample Webhook Channel",
    "description": "Sample webhook description",
    "config_type": "webhook",
    "is_enabled": true,
    "webhook": {
      "url": "https://www.example.com"
    }
  }
}
EOF
}
`

var testAccOpensearchOpenDistroSlackChannelConfiguration = `
resource "opensearch_channel_configuration" "slack_channel_configuration" {
  body = <<EOF
{
  "id": "sample-slack-id",
  "name": "sample-name",
  "config": {
    "name": "Sample Slack Channel",
    "description": "Sample slack description",
    "config_type": "slack",
    "is_enabled": true,
    "slack": {
      "url": "https://www.example.com"
    }
  }
}
EOF
}
`

var testAccOpensearchOpenDistroChimeChannelConfiguration = `
resource "opensearch_channel_configuration" "chime_channel_configuration" {
  body = <<EOF
{
  "id": "sample-chime-id",
  "name": "sample-name",
  "config": {
    "name": "Sample Chime Channel",
    "description": "Sample chime description",
    "config_type": "chime",
    "is_enabled": true,
    "chime": {
      "url": "https://www.example.com"
    }
  }
}
EOF
}
`

var testAccOpensearchOpenDistroSnsChannelConfiguration = `
resource "opensearch_channel_configuration" "sns_channel_configuration" {
  body = <<EOF
{
  "id": "sample-sns-id",
  "name": "sample-name",
  "config": {
    "name": "Sample Sns Channel",
    "description": "Sample chime description",
    "config_type": "sns",
    "is_enabled": true,
    "sns": {
      "topic_arn": "arn:aws:sns:us-east-1:123456789012:MyTopic",
      "role_arn": "arn:aws:iam::123456789012:role/MyRole"
    }
  }
}
EOF
}
`

var testAccOpensearchOpenDistroSmtpEmailChannelConfiguration = `
resource "opensearch_channel_configuration" "smtp_account_configuration" {
  body = <<EOF
{
  "id": "sample-smtp-account-id",
  "name": "sample-name",
  "config": {
    "name": "Sample Smtp Account Channel",
    "description": "Sample smtp account description",
    "config_type": "smtp_account",
    "is_enabled": true,
    "smtp_account": {
      "host": "example.com",
      "port": 123,
      "method": "start_tls",
      "from_address": "test@example.com"
    }
  }
}
EOF
}

resource "opensearch_channel_configuration" "email_channel_configuration" {
  body = <<EOF
{
  "id": "sample-email-id",
  "name": "sample-name",
  "config": {
    "name": "Sample Email Channel",
    "description": "Sample email description",
    "config_type": "email",
    "is_enabled": true,
    "email": {
      "email_account_id": "${opensearch_channel_configuration.smtp_account_configuration.id}",
      "recipient_list": [{
        "recipient": "recipient@example.com"
      }]
    }
  }
}
EOF
}
`

var testAccOpensearchOpenDistroSesEmailChannelConfiguration = `
resource "opensearch_channel_configuration" "ses_account_configuration" {
  body = <<EOF
{
  "id": "sample-ses-account-id",
  "name": "sample-name",
  "config": {
    "name": "Sample SES Account Channel",
    "description": "Sample ses account description",
    "config_type": "ses_account",
    "is_enabled": true,
    "ses_account": {
      "region": "us-east-1",
      "role_arn": "arn:aws:iam::123456789012:role/MyRole",
      "from_address": "test@example.com"
    }
  }
}
EOF
}

resource "opensearch_channel_configuration" "email_channel_configuration" {
  body = <<EOF
{
  "id": "sample-email-id",
  "name": "sample-name",
  "config": {
    "name": "Sample Email Channel",
    "description": "Sample email description",
    "config_type": "email",
    "is_enabled": true,
    "email": {
      "email_account_id": "${opensearch_channel_configuration.ses_account_configuration.id}",
      "recipient_list": [{
        "recipient": "recipient@example.com"
      }]
    }
  }
}
EOF
}
`

var testAccOpensearchOpenDistroSesEmailGroupChannelConfiguration = `
resource "opensearch_channel_configuration" "ses_account_configuration" {
  body = <<EOF
{
  "id": "sample-ses-account-id",
  "name": "sample-name",
  "config": {
    "name": "Sample SES Account Channel",
    "description": "Sample ses account description",
    "config_type": "ses_account",
    "is_enabled": true,
    "ses_account": {
      "region": "us-east-1",
      "role_arn": "arn:aws:iam::123456789012:role/MyRole",
      "from_address": "test@example.com"
    }
  }
}
EOF
}

resource "opensearch_channel_configuration" "email_group_configuration" {
  body = <<EOF
{
  "id": "sample-email-group-id",
  "name": "sample-name",
  "config": {
    "name": "Sample Email Group Channel",
    "description": "Sample email group description",
    "config_type": "email_group",
    "is_enabled": true,
    "email_group": {
      "recipient_list": [{
         "recipient": "recipient1@example.com"
      },{
         "recipient": "recipient2@example.com"
      }]
    }
  }
}
EOF
}

resource "opensearch_channel_configuration" "email_channel_configuration" {
  body = <<EOF
{
  "id": "sample-email-id",
  "name": "sample-name",
  "config": {
    "name": "Sample Email Channel",
    "description": "Sample email description",
    "config_type": "email",
    "is_enabled": true,
    "email": {
      "email_account_id": "${opensearch_channel_configuration.ses_account_configuration.id}",
      "email_group_id_list": ["${opensearch_channel_configuration.email_group_configuration.id}"]
    }
  }
}
EOF
}
`
