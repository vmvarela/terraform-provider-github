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

	if testAccConf.owner == "" {
		t.Skip("Skipping because `GITHUB_OWNER` is not set")
	}
	if testAccConf.testOrgRepository == "" {
		t.Skip("Skipping because `GH_TEST_ORG_REPOSITORY` is not set")
	}
	if testAccConf.testOrgUser == "" {
		t.Skip("Skipping because `GH_TEST_ORG_USER` is not set")
	}

	// Use testOrgUser and username for testing with two users
	user1 := testAccConf.testOrgUser
	user2 := testAccConf.username
	org := testAccConf.owner
	repo := testAccConf.testOrgRepository

	configBefore := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-%s"

			users         = [%q, %q]
			organizations = [%q]
			repositories  = [%q]
		}
	`, testAccConf.enterpriseSlug, randomID, user1, user2, org, repo)

	configAfter := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-updated-%s"

			users         = [%q]
			organizations = []
			repositories  = []
		}
	`, testAccConf.enterpriseSlug, randomID, user1)

	configEmpty := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-%s"

			users         = []
			organizations = []
			repositories  = []
		}
	`, testAccConf.enterpriseSlug, randomID)

	checkBefore := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "enterprise_slug", testAccConf.enterpriseSlug),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "state", "active"),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "organizations.#", "1"),
		resource.TestCheckTypeSetElemAttr("github_enterprise_cost_center.test", "organizations.*", org),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "repositories.#", "1"),
		resource.TestCheckTypeSetElemAttr("github_enterprise_cost_center.test", "repositories.*", repo),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "users.#", "2"),
		resource.TestCheckTypeSetElemAttr("github_enterprise_cost_center.test", "users.*", user1),
		resource.TestCheckTypeSetElemAttr("github_enterprise_cost_center.test", "users.*", user2),
	)

	checkAfter := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "name", fmt.Sprintf("tf-acc-test-updated-%s", randomID)),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "state", "active"),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "organizations.#", "0"),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "repositories.#", "0"),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "users.#", "1"),
		resource.TestCheckTypeSetElemAttr("github_enterprise_cost_center.test", "users.*", user1),
	)

	checkEmpty := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "organizations.#", "0"),
		resource.TestCheckResourceAttr("github_enterprise_cost_center.test", "repositories.#", "0"),
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
