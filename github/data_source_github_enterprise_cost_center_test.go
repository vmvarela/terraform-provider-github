package github

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubEnterpriseCostCenterDataSource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	if testAccConf.enterpriseCostCenterOrg == "" {
		t.Skip("Skipping because `ENTERPRISE_TEST_ORGANIZATION` is not set")
	}
	if testAccConf.enterpriseCostCenterRepo == "" {
		t.Skip("Skipping because `ENTERPRISE_TEST_REPOSITORY` is not set")
	}
	if testAccConf.enterpriseCostCenterUsers == "" {
		t.Skip("Skipping because `ENTERPRISE_TEST_USERS` is not set")
	}

	users := splitCommaSeparated(testAccConf.enterpriseCostCenterUsers)
	if len(users) == 0 {
		t.Skip("Skipping because `ENTERPRISE_TEST_USERS` must contain at least one username")
	}

	userList := fmt.Sprintf("%q", users[0])
	usersInConfig := []string{users[0]}
	orgsInConfig := []string{testAccConf.enterpriseCostCenterOrg}
	reposInConfig := []string{testAccConf.enterpriseCostCenterRepo}

	config := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-test-%s"

			users         = [%s]
			organizations = [%q]
			repositories  = [%q]
		}

		data "github_enterprise_cost_center" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			cost_center_id  = github_enterprise_cost_center.test.id
		}
	`, testAccConf.enterpriseSlug, randomID, userList, testAccConf.enterpriseCostCenterOrg, testAccConf.enterpriseCostCenterRepo)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { skipUnlessEnterprise(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair("data.github_enterprise_cost_center.test", "cost_center_id", "github_enterprise_cost_center.test", "id"),
				resource.TestCheckResourceAttrPair("data.github_enterprise_cost_center.test", "name", "github_enterprise_cost_center.test", "name"),
				resource.TestCheckResourceAttr("data.github_enterprise_cost_center.test", "state", "active"),
				resource.TestCheckResourceAttr("data.github_enterprise_cost_center.test", "organizations.#", strconv.Itoa(len(orgsInConfig))),
				resource.TestCheckTypeSetElemAttr("data.github_enterprise_cost_center.test", "organizations.*", testAccConf.enterpriseCostCenterOrg),
				resource.TestCheckResourceAttr("data.github_enterprise_cost_center.test", "repositories.#", strconv.Itoa(len(reposInConfig))),
				resource.TestCheckTypeSetElemAttr("data.github_enterprise_cost_center.test", "repositories.*", testAccConf.enterpriseCostCenterRepo),
				resource.TestCheckResourceAttr("data.github_enterprise_cost_center.test", "users.#", strconv.Itoa(len(usersInConfig))),
				resource.TestCheckTypeSetElemAttr("data.github_enterprise_cost_center.test", "users.*", users[0]),
			),
		}},
	})
}
