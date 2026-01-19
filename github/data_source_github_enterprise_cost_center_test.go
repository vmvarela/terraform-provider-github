package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubEnterpriseCostCenterDataSource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	if testAccConf.username == "" {
		t.Skip("Skipping because `GITHUB_USERNAME` is not set")
	}

	// Use username for testing
	user := testAccConf.username

	config := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-%s"

			users = [%q]
		}

		data "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			cost_center_id  = github_enterprise_cost_center.test.id
		}
	`, testAccConf.enterpriseSlug, randomID, user)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { skipUnlessEnterprise(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair("data.github_enterprise_cost_center.test", "cost_center_id", "github_enterprise_cost_center.test", "id"),
				resource.TestCheckResourceAttrPair("data.github_enterprise_cost_center.test", "name", "github_enterprise_cost_center.test", "name"),
				resource.TestCheckResourceAttr("data.github_enterprise_cost_center.test", "state", "active"),
				resource.TestCheckResourceAttr("data.github_enterprise_cost_center.test", "users.#", "1"),
				resource.TestCheckTypeSetElemAttr("data.github_enterprise_cost_center.test", "users.*", user),
			),
		}},
	})
}
