package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchDetector_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchDetectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDetectorBasic,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDetectorExists("opensearch_detector.test_detector"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector", "name", "test-detector"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector", "detector_type", "windows"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector", "enabled", "true"),
				),
			},
		},
	})
}

func TestAccOpensearchDetector_withTriggers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchDetectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDetectorWithTriggers,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDetectorExists("opensearch_detector.test_detector_with_triggers"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_with_triggers", "name", "test-detector-with-triggers"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_with_triggers", "detector_type", "linux"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_with_triggers", "triggers.#", "1"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_with_triggers", "triggers.0.name", "test-trigger"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_with_triggers", "triggers.0.severity", "1"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_with_triggers", "triggers.0.detection_types.#", "1"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_with_triggers", "triggers.0.detection_types.0", "rules"),
				),
			},
		},
	})
}

func TestAccOpensearchDetector_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchDetectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDetectorBasic,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDetectorExists("opensearch_detector.test_detector"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector", "enabled", "true"),
				),
			},
			{
				Config: testAccOpensearchDetectorUpdated,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDetectorExists("opensearch_detector.test_detector"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector", "enabled", "false"),
				),
			},
		},
	})
}

func TestAccOpensearchDetector_detectionTypesDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchDetectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDetectorWithTriggersNoDetectionTypes,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDetectorExists("opensearch_detector.test_detector_default_detection_types"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_default_detection_types", "triggers.0.detection_types.#", "1"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_default_detection_types", "triggers.0.detection_types.0", "rules"),
				),
			},
		},
	})
}

func testCheckOpensearchDetectorExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No detector ID is set")
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchGetDetector(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchDetectorDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_detector" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchGetDetector(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Detector %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchDetectorBasic = `
resource "opensearch_detector" "test_detector" {
  name         = "test-detector"
  detector_type = "windows"
  enabled      = true

  schedule {
    period {
      interval = 10
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input {
      description = "Test detector for windows logs"
      indices     = ["windows-logs"]
      
      pre_packaged_rules {
        id = "06724a9a-52fc-11ed-bdc3-0242ac120002"
      }
    }
  }
}
`

var testAccOpensearchDetectorUpdated = `
resource "opensearch_detector" "test_detector" {
  name         = "test-detector"
  detector_type = "windows"
  enabled      = false

  schedule {
    period {
      interval = 10
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input {
      description = "Test detector for windows logs"
      indices     = ["windows-logs"]
      
      pre_packaged_rules {
        id = "06724a9a-52fc-11ed-bdc3-0242ac120002"
      }
    }
  }
}
`

var testAccOpensearchDetectorWithTriggers = `
resource "opensearch_detector" "test_detector_with_triggers" {
  name         = "test-detector-with-triggers"
  detector_type = "linux"
  enabled      = true

  schedule {
    period {
      interval = 5
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input {
      description = "Test detector for linux logs with triggers"
      indices     = ["linux-logs"]
      
      pre_packaged_rules {
        id = "847def9e-924d-4e90-b7c4-5f581395a2b4"
      }
    }
  }

  triggers {
    name     = "test-trigger"
    severity = "1"
    
    ids = [
      "847def9e-924d-4e90-b7c4-5f581395a2b4"
    ]
    
    sev_levels = ["critical"]
    tags       = ["attack.t1003.002"]
    
    actions {
      id             = "test-action-id"
      name           = "test-action"
      destination_id = "test-destination-id"
      
      subject_template {
        source = "Security Alert: {{ctx.trigger.name}}"
        lang   = "mustache"
      }
      
      message_template {
        source = "Detector {{ctx.detector.name}} triggered an alert"
        lang   = "mustache"
      }
      
      throttle_enabled = true
      
      throttle {
        unit  = "MINUTES"
        value = 10
      }
    }
  }
}
`

var testAccOpensearchDetectorWithTriggersNoDetectionTypes = `
resource "opensearch_detector" "test_detector_default_detection_types" {
  name         = "test-detector-default-detection-types"
  detector_type = "apache_access"
  enabled      = true

  schedule {
    period {
      interval = 10
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input {
      description = "Test detector to verify detection_types defaults"
      indices     = ["apache-logs"]
      
      custom_rules {
        id = "test-custom-rule-id"
      }
    }
  }

  triggers {
    name     = "test-trigger-default-detection-types"
    severity = "2"
    
    ids = [
      "test-custom-rule-id"
    ]
    
    sev_levels = ["high"]
    tags       = ["attack.discovery"]
    
    # Note: detection_types is not specified here - should default to ["rules"]
    
    actions {
      id             = "test-action-default"
      name           = "test-action-default"
      destination_id = "test-destination-default"
      
      subject_template {
        source = "Default Detection Types Test"
        lang   = "mustache"
      }
      
      message_template {
        source = "Testing default detection_types behavior"
        lang   = "mustache"
      }
    }
  }
}
`
