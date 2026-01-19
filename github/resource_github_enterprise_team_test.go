package github

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubEnterpriseTeam(t *testing.T) {
	t.Run("creates and updates resource without error", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessMode(t, enterprise) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
						data "github_enterprise" "enterprise" {
							slug = "%s"
						}

						resource "github_enterprise_team" "test" {
							enterprise_slug             = data.github_enterprise.enterprise.slug
							name                        = "%s%s"
							description                 = "team for acceptance testing"
							organization_selection_type = "disabled"
						}
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("github_enterprise_team.test", "slug"),
						resource.TestCheckResourceAttrSet("github_enterprise_team.test", "team_id"),
						resource.TestCheckResourceAttr("github_enterprise_team.test", "organization_selection_type", "disabled"),
					),
				},
				{
					Config: fmt.Sprintf(`
						data "github_enterprise" "enterprise" {
							slug = "%s"
						}

						resource "github_enterprise_team" "test" {
							enterprise_slug             = data.github_enterprise.enterprise.slug
							name                        = "%s%s"
							description                 = "updated description"
							organization_selection_type = "selected"
						}
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("github_enterprise_team.test", "description", "updated description"),
						resource.TestCheckResourceAttr("github_enterprise_team.test", "organization_selection_type", "selected"),
					),
				},
			},
		})
	})

	t.Run("imports resource without error", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessMode(t, enterprise) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
						data "github_enterprise" "enterprise" {
							slug = "%s"
						}

						resource "github_enterprise_team" "test" {
							enterprise_slug             = data.github_enterprise.enterprise.slug
							name                        = "%s%s"
							description                 = "team for import testing"
							organization_selection_type = "disabled"
						}
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID),
				},
				{
					ResourceName:        "github_enterprise_team.test",
					ImportState:         true,
					ImportStateVerify:   true,
					ImportStateIdPrefix: fmt.Sprintf(`%s/`, testAccConf.enterpriseSlug),
				},
			},
		})
	})
}

func TestAccGithubEnterpriseTeamOrganizations(t *testing.T) {
	t.Run("assigns organizations to team without error", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessMode(t, enterprise) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
						data "github_enterprise" "enterprise" {
							slug = "%s"
						}

						resource "github_enterprise_team" "test" {
							enterprise_slug             = data.github_enterprise.enterprise.slug
							name                        = "%s%s"
							organization_selection_type = "selected"
						}

						resource "github_enterprise_team_organizations" "test" {
							enterprise_slug    = data.github_enterprise.enterprise.slug
							team_slug          = github_enterprise_team.test.slug
							organization_slugs = ["%s"]
						}
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID, testAccConf.owner),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("github_enterprise_team_organizations.test", "organization_slugs.#", "1"),
						resource.TestCheckTypeSetElemAttr("github_enterprise_team_organizations.test", "organization_slugs.*", testAccConf.owner),
					),
				},
			},
		})
	})

	t.Run("imports resource without error", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessMode(t, enterprise) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
						data "github_enterprise" "enterprise" {
							slug = "%s"
						}

						resource "github_enterprise_team" "test" {
							enterprise_slug             = data.github_enterprise.enterprise.slug
							name                        = "%s%s"
							organization_selection_type = "selected"
						}

						resource "github_enterprise_team_organizations" "test" {
							enterprise_slug    = data.github_enterprise.enterprise.slug
							team_slug          = github_enterprise_team.test.slug
							organization_slugs = ["%s"]
						}
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID, testAccConf.owner),
				},
				{
					ResourceName:      "github_enterprise_team_organizations.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})

	t.Run("errors on empty organizations", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessMode(t, enterprise) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
						data "github_enterprise" "enterprise" {
							slug = "%s"
						}

						resource "github_enterprise_team" "test" {
							enterprise_slug             = data.github_enterprise.enterprise.slug
							name                        = "%s%s"
							organization_selection_type = "selected"
						}

						resource "github_enterprise_team_organizations" "test" {
							enterprise_slug    = data.github_enterprise.enterprise.slug
							team_slug          = github_enterprise_team.test.slug
							organization_slugs = []
						}
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID),
					ExpectError: regexp.MustCompile(`Attribute organization_slugs requires 1 item minimum`),
				},
			},
		})
	})
}

func TestAccGithubEnterpriseTeamMembership(t *testing.T) {
	t.Run("adds member to team without error", func(t *testing.T) {
		if testAccConf.testOrgUser == "" {
			t.Skip("Skipping because GH_TEST_ORG_USER is not set")
		}
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessMode(t, enterprise) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
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
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID, testAccConf.testOrgUser),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("github_enterprise_team_membership.test", "username", testAccConf.testOrgUser),
					),
				},
			},
		})
	})

	t.Run("imports resource without error", func(t *testing.T) {
		if testAccConf.testOrgUser == "" {
			t.Skip("Skipping because GH_TEST_ORG_USER is not set")
		}
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessMode(t, enterprise) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
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
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID, testAccConf.testOrgUser),
				},
				{
					ResourceName:      "github_enterprise_team_membership.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}
