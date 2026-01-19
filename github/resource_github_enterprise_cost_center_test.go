package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGithubEnterpriseCostCenter(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	if testAccConf.username == "" {
		t.Skip("Skipping because `GITHUB_USERNAME` is not set")
	}

	// Use username for testing
	user := testAccConf.username

	configBefore := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-%s"

			users = [%q]
		}
	`, testAccConf.enterpriseSlug, randomID, user)

	configAfter := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-updated-%s"

			users = [%q]
		}
	`, testAccConf.enterpriseSlug, randomID, user)

	configEmpty := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-%s"

			users = []
		}
	`, testAccConf.enterpriseSlug, randomID)

	checkBefore := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "enterprise_slug", testAccConf.enterpriseSlug),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "state", "active"),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "users.#", "1"),
		resource.TestCheckTypeSetElemAttr("github_enterprise_cost_center.test", "users.*", user),
	)

	checkAfter := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "name", fmt.Sprintf("tf-acc-test-updated-%s", randomID)),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "state", "active"),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "users.#", "1"),
		resource.TestCheckTypeSetElemAttr("github_enterprise_cost_center.test", "users.*", user),
	)

	checkEmpty := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "users.#", "0"),
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { skipUnlessEnterprise(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: configBefore,
				Check:  checkBefore,
			},
			{
				Config: configAfter,
				Check:  checkAfter,
			},
			{
				Config: configEmpty,
				Check:  checkEmpty,
			},
			{
				ResourceName:      "github_enterprise_cost_center.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["github_enterprise_cost_center.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return fmt.Sprintf("%s:%s", testAccConf.enterpriseSlug, rs.Primary.ID), nil
				},
			},
		},
	})
}
