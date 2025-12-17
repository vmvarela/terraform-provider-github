package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubEnterpriseCostCenterDataSource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	if isEnterprise != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to false")
	}
	if testEnterprise == "" {
		t.Skip("Skipping because `ENTERPRISE_SLUG` is not set")
	}

	config := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-%s"
		}

		data "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			cost_center_id  = github_enterprise_cost_center.test.id
		}
	`, testEnterprise, randomID)

	check := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttrPair("data.github_enterprise_cost_center.test", "cost_center_id", "github_enterprise_cost_center.test", "id"),
		resource.TestCheckResourceAttrPair("data.github_enterprise_cost_center.test", "name", "github_enterprise_cost_center.test", "name"),
		resource.TestCheckResourceAttr("data.github_enterprise_cost_center.test", "state", "active"),
	)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { skipUnlessMode(t, enterprise) },
		Providers: testAccProviders,
		Steps:     []resource.TestStep{{Config: config, Check: check}},
	})
}
