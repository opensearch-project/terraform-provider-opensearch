package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"

	elastic7 "github.com/olivere/elastic/v7"
	elastic6 "gopkg.in/olivere/elastic.v6"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchSnapshotRepository(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchSnapshotRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchSnapshotRepository,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchSnapshotRepositoryExists("opensearch_snapshot_repository.test"),
				),
			},
		},
	})
}

func TestAccOpensearchSnapshotRepository_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckOpensearchSnapshotRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchSnapshotRepository,
			},
			{
				ResourceName:      "opensearch_snapshot_repository.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testCheckOpensearchSnapshotRepositoryExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No snapshot repository ID is set")
		}

		meta := testAccProvider.Meta()

		var err error
		esClient, err := getClient(meta.(*ProviderConf))
		if err != nil {
			return err
		}
		switch client := esClient.(type) {
		case *elastic7.Client:
			_, err = client.SnapshotGetRepository(rs.Primary.ID).Do(context.TODO())
		case *elastic6.Client:
			_, err = client.SnapshotGetRepository(rs.Primary.ID).Do(context.TODO())
		default:
			return errors.New("opensearch version not supported")
		}

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchSnapshotRepositoryDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_snapshot_repository" {
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
			_, err = client.SnapshotGetRepository(rs.Primary.ID).Do(context.TODO())
		case *elastic6.Client:
			_, err = client.SnapshotGetRepository(rs.Primary.ID).Do(context.TODO())
		default:
			return errors.New("opensearch version not supported")
		}

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("Snapshot repository %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchSnapshotRepository = `
resource "opensearch_snapshot_repository" "test" {
  name = "terraform-test"
  type = "fs"

  settings = {
    location = "/tmp/opensearch"
  }
}
`
