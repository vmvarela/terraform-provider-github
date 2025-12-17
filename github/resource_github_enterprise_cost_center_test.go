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

	if isEnterprise != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to false")
	}
	if testEnterprise == "" {
		t.Skip("Skipping because `ENTERPRISE_SLUG` is not set")
	}

	configBefore := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-%s"
		}
	`, testEnterprise, randomID)

	configAfter := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-updated-%s"
		}
	`, testEnterprise, randomID)

	checkBefore := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "enterprise_slug", testEnterprise),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "state", "active"),
	)

	checkAfter := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "name", fmt.Sprintf("tf-acc-test-updated-%s", randomID)),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "state", "active"),
	)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { skipUnlessMode(t, enterprise) },
		Providers: testAccProviders,
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
				ResourceName:      "github_enterprise_cost_center.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["github_enterprise_cost_center.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return fmt.Sprintf("%s/%s", testEnterprise, rs.Primary.ID), nil
				},
			},
		},
	})
}
