package provider

import (
	"context"
	"fmt"
	"testing"

	elastic7 "github.com/olivere/elastic/v7"
	elastic6 "gopkg.in/olivere/elastic.v6"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchScript(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchScriptDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchScript,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchScriptExists("opensearch_script.test_script"),
				),
			},
		},
	})
}

func testCheckOpensearchScriptExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No script ID is set")
		}

		meta := testAccProvider.Meta()

		var err error
		esClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		switch client := esClient.(type) {
		case *elastic7.Client:
			_, err = client.GetScript().Id("my_script").Do(context.TODO())
		case *elastic6.Client:
			_, err = client.GetScript().Id("my_script").Do(context.TODO())
		default:
		}

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchScriptDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_script" {
			continue
		}

		meta := testAccProvider.Meta()

		var err error
		esClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		switch client := esClient.(type) {
		case *elastic7.Client:
			_, err = client.GetScript().Id("my_script").Do(context.TODO())
		case *elastic6.Client:
			_, err = client.GetScript().Id("my_script").Do(context.TODO())
		default:
		}

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Script %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchScript = `
resource "opensearch_script" "test_script" {
  script_id = "my_script"
  lang      = "painless"
  source    = "Math.log(_score * 2) + params.my_modifier"
}
`
