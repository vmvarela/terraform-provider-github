package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubEnterpriseTeam(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config1 := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_team" "test" {
			enterprise_slug             = data.github_enterprise.enterprise.slug
			name                        = "tf-acc-team-%s"
			description                 = "team for acceptance testing"
			organization_selection_type = "disabled"
		}
	`, testAccConf.enterpriseSlug, randomID)

	config2 := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_team" "test" {
			enterprise_slug             = data.github_enterprise.enterprise.slug
			name                        = "tf-acc-team-%s"
			description                 = "updated description"
			organization_selection_type = "selected"
		}
	`, testAccConf.enterpriseSlug, randomID)

	check1 := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttrSet("github_enterprise_team.test", "slug"),
		resource.TestCheckResourceAttrSet("github_enterprise_team.test", "team_id"),
		resource.TestCheckResourceAttr("github_enterprise_team.test", "organization_selection_type", "disabled"),
	)
	check2 := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_team.test", "description", "updated description"),
		resource.TestCheckResourceAttr("github_enterprise_team.test", "organization_selection_type", "selected"),
	)

	testCase := func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:  func() { skipUnlessMode(t, enterprise) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{Config: config1, Check: check1},
				{Config: config2, Check: check2},
				{
					ResourceName:        "github_enterprise_team.test",
					ImportState:         true,
					ImportStateVerify:   true,
					ImportStateIdPrefix: fmt.Sprintf(`%s/`, testAccConf.enterpriseSlug),
				},
			},
		})
	}

	t.Run("with an enterprise account", func(t *testing.T) {
		testCase(t)
	})
}

func TestAccGithubEnterpriseTeamOrganizations(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config1 := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_team" "test" {
			enterprise_slug             = data.github_enterprise.enterprise.slug
			name                        = "tf-acc-team-orgs-%s"
			organization_selection_type = "selected"
		}

		resource "github_enterprise_team_organizations" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			enterprise_team = github_enterprise_team.test.slug
			organization_slugs = ["%s"]
		}
	`, testAccConf.enterpriseSlug, randomID, testAccConf.owner)

	config2 := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_team" "test" {
			enterprise_slug             = data.github_enterprise.enterprise.slug
			name                        = "tf-acc-team-orgs-%s"
			organization_selection_type = "selected"
		}

		resource "github_enterprise_team_organizations" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			enterprise_team = github_enterprise_team.test.slug
			organization_slugs = []
		}
	`, testAccConf.enterpriseSlug, randomID)

	check1 := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_team_organizations.test", "organization_slugs.#", "1"),
		resource.TestCheckTypeSetElemAttr("github_enterprise_team_organizations.test", "organization_slugs.*", testAccConf.owner),
	)
	check2 := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_team_organizations.test", "organization_slugs.#", "0"),
	)

	testCase := func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:  func() { skipUnlessMode(t, enterprise) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{Config: config1, Check: check1},
				{Config: config2, Check: check2},
				{
					ResourceName:      "github_enterprise_team_organizations.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	}

	t.Run("with an enterprise account", func(t *testing.T) {
		testCase(t)
	})
}

func TestAccGithubEnterpriseTeamMembership(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	username := testAccConf.username

	config := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_team" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "tf-acc-team-member-%s"
		}

		resource "github_enterprise_team_membership" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			enterprise_team = github_enterprise_team.test.slug
			username        = "%s"
		}
	`, testAccConf.enterpriseSlug, randomID, username)

	check := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr("github_enterprise_team_membership.test", "username", username),
	)

	testCase := func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:  func() { skipUnlessMode(t, enterprise) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{Config: config, Check: check},
				{
					ResourceName:      "github_enterprise_team_membership.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	}

	t.Run("with an enterprise account", func(t *testing.T) {
		testCase(t)
	})
}
