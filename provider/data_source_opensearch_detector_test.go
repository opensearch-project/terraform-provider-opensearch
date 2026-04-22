package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOpensearchDetector_byId(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccOpendistroProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpensearchDetectorConfigById,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.opensearch_detector.test_detector_data", "id"),
					resource.TestCheckResourceAttr("data.opensearch_detector.test_detector_data", "name", "test-detector-for-data-source"),
					resource.TestCheckResourceAttr("data.opensearch_detector.test_detector_data", "detector_type", "network"),
					resource.TestCheckResourceAttr("data.opensearch_detector.test_detector_data", "enabled", "true"),
					resource.TestCheckResourceAttrSet("data.opensearch_detector.test_detector_data", "last_update_time"),
				),
			},
		},
	})
}

func TestAccDataSourceOpensearchDetector_byName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccOpendistroProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpensearchDetectorConfigByName,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.opensearch_detector.test_detector_data", "id"),
					resource.TestCheckResourceAttr("data.opensearch_detector.test_detector_data", "name", "test-detector-by-name"),
					resource.TestCheckResourceAttr("data.opensearch_detector.test_detector_data", "detector_type", "dns"),
					resource.TestCheckResourceAttr("data.opensearch_detector.test_detector_data", "enabled", "true"),
				),
			},
		},
	})
}

func TestAccDataSourceOpensearchDetector_byType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccOpendistroProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpensearchDetectorConfigByType,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.opensearch_detector.test_detector_data", "id"),
					resource.TestCheckResourceAttr("data.opensearch_detector.test_detector_data", "detector_type", "apache_access"),
					resource.TestCheckResourceAttr("data.opensearch_detector.test_detector_data", "enabled", "true"),
				),
			},
		},
	})
}

var testAccDataSourceOpensearchDetectorConfigById = `
resource "opensearch_detector" "test_detector" {
  name         = "test-detector-for-data-source"
  detector_type = "network"
  enabled      = true

  schedule {
    period {
      interval = 15
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input {
      description = "Test detector for data source testing"
      indices     = ["network-logs"]
      
      pre_packaged_rules {
        id = "73a883d0-0348-4be4-a8d8-51031c2564f8"
      }
    }
  }
}

data "opensearch_detector" "test_detector_data" {
  detector_id = opensearch_detector.test_detector.id
}
`

var testAccDataSourceOpensearchDetectorConfigByName = `
resource "opensearch_detector" "test_detector" {
  name         = "test-detector-by-name"
  detector_type = "dns"
  enabled      = true

  schedule {
    period {
      interval = 20
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input {
      description = "Test detector for name-based search"
      indices     = ["dns-logs"]
      
      pre_packaged_rules {
        id = "1a4bd6e3-4c6e-405d-a9a3-53a116e341d4"
      }
    }
  }
}

data "opensearch_detector" "test_detector_data" {
  name = opensearch_detector.test_detector.name
}
`

var testAccDataSourceOpensearchDetectorConfigByType = `
resource "opensearch_detector" "test_detector" {
  name         = "test-detector-by-type"
  detector_type = "apache_access"
  enabled      = true

  schedule {
    period {
      interval = 30
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input {
      description = "Test detector for type-based search"
      indices     = ["apache-logs"]
      
      pre_packaged_rules {
        id = "847def9e-924d-4e90-b7c4-5f581395a2b4"
      }
    }
  }
}

data "opensearch_detector" "test_detector_data" {
  detector_type = opensearch_detector.test_detector.detector_type
}
`