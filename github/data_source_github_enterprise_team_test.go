package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubEnterpriseTeamDataSource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_team" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-ds-team-%s"
		}

		data "github_enterprise_team" "by_slug" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			slug            = github_enterprise_team.test.slug
		}

		data "github_enterprise_team" "by_id" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			team_id         = github_enterprise_team.test.team_id
		}
	`, testAccConf.enterpriseSlug, randomID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { skipUnlessMode(t, enterprise) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.github_enterprise_team.by_slug", "id"),
					resource.TestCheckResourceAttrPair("data.github_enterprise_team.by_slug", "team_id", "github_enterprise_team.test", "team_id"),
					resource.TestCheckResourceAttrPair("data.github_enterprise_team.by_slug", "slug", "github_enterprise_team.test", "slug"),
					resource.TestCheckResourceAttrPair("data.github_enterprise_team.by_slug", "name", "github_enterprise_team.test", "name"),
					resource.TestCheckResourceAttrSet("data.github_enterprise_team.by_id", "id"),
					resource.TestCheckResourceAttrPair("data.github_enterprise_team.by_id", "team_id", "github_enterprise_team.test", "team_id"),
					resource.TestCheckResourceAttrPair("data.github_enterprise_team.by_id", "slug", "github_enterprise_team.test", "slug"),
				),
			},
		},
	})
}

func TestAccGithubEnterpriseTeamOrganizationsDataSource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	if testAccConf.enterpriseTestOrganization == "" {
		t.Skip("Skipping because `ENTERPRISE_TEST_ORGANIZATION` is not set")
	}

	config := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_team" "test" {
			enterprise_slug             = data.github_enterprise.enterprise.slug
			name                        = "%s%s"
			organization_selection_type = "selected"
		}

		resource "github_enterprise_team_organizations" "assign" {
			enterprise_slug    = data.github_enterprise.enterprise.slug
			team_slug          = github_enterprise_team.test.slug
			organization_slugs = ["%s"]
		}

		data "github_enterprise_team_organizations" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			team_slug       = github_enterprise_team.test.slug
			depends_on      = [github_enterprise_team_organizations.assign]
		}
	`, testAccConf.enterpriseSlug, testResourcePrefix, randomID, testAccConf.enterpriseTestOrganization)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { skipUnlessMode(t, enterprise) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.github_enterprise_team_organizations.test", "id"),
					resource.TestCheckResourceAttr("data.github_enterprise_team_organizations.test", "organization_slugs.#", "1"),
					resource.TestCheckTypeSetElemAttr("data.github_enterprise_team_organizations.test", "organization_slugs.*", testAccConf.enterpriseTestOrganization),
				),
			},
		},
	})
}

func TestAccGithubEnterpriseTeamMembershipDataSource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	if testAccConf.enterpriseTestUsers == "" {
		t.Skip("Skipping because `ENTERPRISE_TEST_USERS` is not set")
	}

	users := splitCommaSeparated(testAccConf.enterpriseTestUsers)
	if len(users) == 0 {
		t.Skip("Skipping because `ENTERPRISE_TEST_USERS` must contain at least one username")
	}
	username := users[0]

	config := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_team" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "%s%s"
		}

		resource "github_enterprise_team_membership" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			team_slug       = github_enterprise_team.test.slug
			username        = "%s"
		}

		data "github_enterprise_team_membership" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			team_slug       = github_enterprise_team.test.slug
			username        = "%s"
			depends_on      = [github_enterprise_team_membership.test]
		}
	`, testAccConf.enterpriseSlug, testResourcePrefix, randomID, username, username)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { skipUnlessMode(t, enterprise) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.github_enterprise_team_membership.test", "id"),
					resource.TestCheckResourceAttr("data.github_enterprise_team_membership.test", "username", username),
				),
			},
		},
	})
}
