package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

func TestAccOpensearchDetector_cronSchedule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchDetectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDetectorCronSchedule,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDetectorExists("opensearch_detector.test_detector_cron"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_cron", "name", "test-detector-cron"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_cron", "detector_type", "windows"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_cron", "enabled", "true"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_cron", "schedule.0.cron.0.expression", "0 */6 * * *"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_cron", "schedule.0.cron.0.timezone", "UTC"),
				),
			},
		},
	})
}

func TestAccOpensearchDetector_updateIntervalToCron(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchDetectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDetectorScheduleChangeInitial,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDetectorExists("opensearch_detector.test_detector_schedule_change"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_schedule_change", "schedule.0.period.0.interval", "10"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_schedule_change", "schedule.0.period.0.unit", "MINUTES"),
				),
			},
			{
				Config: testAccOpensearchDetectorPeriodToCron,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDetectorExists("opensearch_detector.test_detector_schedule_change"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_schedule_change", "schedule.0.cron.0.expression", "0 */12 * * *"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_schedule_change", "schedule.0.cron.0.timezone", "America/New_York"),
				),
			},
		},
	})
}

func TestAccOpensearchDetector_updateCronToInterval(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchDetectorDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchDetectorPeriodToCron,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDetectorExists("opensearch_detector.test_detector_schedule_change"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_schedule_change", "schedule.0.cron.0.expression", "0 */12 * * *"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_schedule_change", "schedule.0.cron.0.timezone", "America/New_York"),
				),
			},
			{
				Config: testAccOpensearchDetectorCronToPeriod,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchDetectorExists("opensearch_detector.test_detector_schedule_change"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_schedule_change", "schedule.0.period.0.interval", "15"),
					resource.TestCheckResourceAttr("opensearch_detector.test_detector_schedule_change", "schedule.0.period.0.unit", "MINUTES"),
				),
			},
		},
	})
}

// TestBuildDetectorBody_cronSchedule verifies buildDetectorBody serializes a cron schedule correctly.
func TestBuildDetectorBody_cronSchedule(t *testing.T) {
	r := resourceOpenSearchDetector()
	if err := schema.InternalMap(r.Schema).InternalValidate(nil); err != nil {
		t.Fatalf("schema validation failed: %s", err)
	}

	state := &terraform.InstanceState{
		ID: "test-id",
		Attributes: map[string]string{
			"name":                         "test-detector",
			"detector_type":                "windows",
			"enabled":                      "true",
			"schedule.#":                   "1",
			"schedule.0.period.#":          "0",
			"schedule.0.cron.#":            "1",
			"schedule.0.cron.0.expression": "0 */6 * * *",
			"schedule.0.cron.0.timezone":   "UTC",
			"inputs.#":                     "0",
			"triggers.#":                   "0",
		},
	}

	rd, err := schema.InternalMap(r.Schema).Data(state, nil)
	if err != nil {
		t.Fatalf("failed to create ResourceData: %s", err)
	}

	body := buildDetectorBody(rd)

	scheduleRaw, ok := body["schedule"]
	if !ok {
		t.Fatal("expected 'schedule' key in body")
	}
	schedule, ok := scheduleRaw.(map[string]interface{})
	if !ok {
		t.Fatal("expected schedule to be map[string]interface{}")
	}

	cronRaw, ok := schedule["cron"]
	if !ok {
		t.Fatal("expected 'cron' key in schedule")
	}
	cron, ok := cronRaw.(map[string]interface{})
	if !ok {
		t.Fatal("expected cron to be map[string]interface{}")
	}

	if cron["expression"] != "0 */6 * * *" {
		t.Errorf("expected expression %q, got %q", "0 */6 * * *", cron["expression"])
	}
	if cron["timezone"] != "UTC" {
		t.Errorf("expected timezone %q, got %q", "UTC", cron["timezone"])
	}

	if _, hasPeriod := schedule["period"]; hasPeriod {
		t.Error("expected no 'period' key in schedule for cron schedule")
	}
}

// TestBuildDetectorBody_periodSchedule verifies buildDetectorBody serializes a period schedule correctly.
func TestBuildDetectorBody_periodSchedule(t *testing.T) {
	r := resourceOpenSearchDetector()
	if err := schema.InternalMap(r.Schema).InternalValidate(nil); err != nil {
		t.Fatalf("schema validation failed: %s", err)
	}

	state := &terraform.InstanceState{
		ID: "test-id",
		Attributes: map[string]string{
			"name":                         "test-detector",
			"detector_type":                "windows",
			"enabled":                      "true",
			"schedule.#":                   "1",
			"schedule.0.period.#":          "1",
			"schedule.0.period.0.interval": "10",
			"schedule.0.period.0.unit":     "MINUTES",
			"schedule.0.cron.#":            "0",
			"inputs.#":                     "0",
			"triggers.#":                   "0",
		},
	}

	rd, err := schema.InternalMap(r.Schema).Data(state, nil)
	if err != nil {
		t.Fatalf("failed to create ResourceData: %s", err)
	}

	body := buildDetectorBody(rd)

	scheduleRaw, ok := body["schedule"]
	if !ok {
		t.Fatal("expected 'schedule' key in body")
	}
	schedule, ok := scheduleRaw.(map[string]interface{})
	if !ok {
		t.Fatal("expected schedule to be map[string]interface{}")
	}

	periodRaw, ok := schedule["period"]
	if !ok {
		t.Fatal("expected 'period' key in schedule")
	}
	period, ok := periodRaw.(map[string]interface{})
	if !ok {
		t.Fatal("expected period to be map[string]interface{}")
	}

	if period["interval"] != 10 {
		t.Errorf("expected interval 10, got %v", period["interval"])
	}
	if period["unit"] != "MINUTES" {
		t.Errorf("expected unit %q, got %q", "MINUTES", period["unit"])
	}

	if _, hasCron := schedule["cron"]; hasCron {
		t.Error("expected no 'cron' key in schedule for period schedule")
	}
}

// TestFlattenDetector_cronSchedule verifies flattenDetector correctly sets state for a cron schedule.
func TestFlattenDetector_cronSchedule(t *testing.T) {
	r := resourceOpenSearchDetector()

	state := &terraform.InstanceState{
		ID: "test-id",
		Attributes: map[string]string{
			"name":          "test-detector",
			"detector_type": "windows",
			"enabled":       "true",
			"schedule.#":    "0",
			"inputs.#":      "0",
			"triggers.#":    "0",
		},
	}

	rd, err := schema.InternalMap(r.Schema).Data(state, nil)
	if err != nil {
		t.Fatalf("failed to create ResourceData: %s", err)
	}

	detector := map[string]interface{}{
		"name":          "test-detector",
		"detector_type": "windows",
		"enabled":       true,
		"schedule": map[string]interface{}{
			"cron": map[string]interface{}{
				"expression": "0 */6 * * *",
				"timezone":   "UTC",
			},
		},
		"inputs": []interface{}{},
	}

	if err := flattenDetector(rd, detector); err != nil {
		t.Fatalf("flattenDetector failed: %s", err)
	}

	scheduleList, ok := rd.GetOk("schedule")
	if !ok {
		t.Fatal("expected 'schedule' to be set in state")
	}
	schedules := scheduleList.([]interface{})
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule entry, got %d", len(schedules))
	}
	schedule := schedules[0].(map[string]interface{})

	cronList, ok := schedule["cron"].([]interface{})
	if !ok || len(cronList) == 0 {
		t.Fatal("expected 'cron' to be set in schedule")
	}
	cron := cronList[0].(map[string]interface{})
	if cron["expression"] != "0 */6 * * *" {
		t.Errorf("expected expression %q, got %q", "0 */6 * * *", cron["expression"])
	}
	if cron["timezone"] != "UTC" {
		t.Errorf("expected timezone %q, got %q", "UTC", cron["timezone"])
	}

	periodList := schedule["period"].([]interface{})
	if len(periodList) != 0 {
		t.Errorf("expected empty 'period' list for cron schedule, got %d items", len(periodList))
	}
}

var testAccOpensearchDetectorCronSchedule = `
resource "opensearch_detector" "test_detector_cron" {
  name         = "test-detector-cron"
  detector_type = "windows"
  enabled      = true

  schedule {
    cron {
      expression = "0 */6 * * *"
      timezone   = "UTC"
    }
  }

  inputs {
    detector_input {
      description = "Test detector for cron schedule"
      indices     = ["windows-logs"]
      pre_packaged_rules {
        id = "06724a9a-52fc-11ed-bdc3-0242ac120002"
      }
    }
  }
}
`

var testAccOpensearchDetectorScheduleChangeInitial = `
resource "opensearch_detector" "test_detector_schedule_change" {
  name         = "test-detector-schedule-change"
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
      description = "Test detector for schedule change"
      indices     = ["windows-logs"]
      pre_packaged_rules {
        id = "06724a9a-52fc-11ed-bdc3-0242ac120002"
      }
    }
  }
}
`

var testAccOpensearchDetectorPeriodToCron = `
resource "opensearch_detector" "test_detector_schedule_change" {
  name         = "test-detector-schedule-change"
  detector_type = "windows"
  enabled      = true

  schedule {
    cron {
      expression = "0 */12 * * *"
      timezone   = "America/New_York"
    }
  }

  inputs {
    detector_input {
      description = "Test detector for schedule change"
      indices     = ["windows-logs"]
      pre_packaged_rules {
        id = "06724a9a-52fc-11ed-bdc3-0242ac120002"
      }
    }
  }
}
`

var testAccOpensearchDetectorCronToPeriod = `
resource "opensearch_detector" "test_detector_schedule_change" {
  name         = "test-detector-schedule-change"
  detector_type = "windows"
  enabled      = true

  schedule {
    period {
      interval = 15
      unit     = "MINUTES"
    }
  }

  inputs {
    detector_input {
      description = "Test detector for schedule change"
      indices     = ["windows-logs"]
      pre_packaged_rules {
        id = "06724a9a-52fc-11ed-bdc3-0242ac120002"
      }
    }
  }
}
`
