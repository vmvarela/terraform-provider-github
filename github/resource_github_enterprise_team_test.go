package github

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccGithubEnterpriseTeam(t *testing.T) {
	t.Run("creates and updates resource without error", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessEnterprise(t) },
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
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_enterprise_team.test", tfjsonpath.New("slug"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue("github_enterprise_team.test", tfjsonpath.New("team_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue("github_enterprise_team.test", tfjsonpath.New("organization_selection_type"), knownvalue.StringExact("disabled")),
					},
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
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_enterprise_team.test", tfjsonpath.New("description"), knownvalue.StringExact("updated description")),
						statecheck.ExpectKnownValue("github_enterprise_team.test", tfjsonpath.New("organization_selection_type"), knownvalue.StringExact("selected")),
					},
				},
			},
		})
	})

	t.Run("imports resource without error", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessEnterprise(t) },
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
					ResourceName:            "github_enterprise_team.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"group_id"},
					ImportStateIdPrefix:     fmt.Sprintf(`%s/`, testAccConf.enterpriseSlug),
				},
			},
		})
	})
}

func TestAccGithubEnterpriseTeamOrganizations(t *testing.T) {
	orgSlug := os.Getenv("ENTERPRISE_TEST_ORGANIZATION")
	if orgSlug == "" {
		t.Skip("ENTERPRISE_TEST_ORGANIZATION not set")
	}

	t.Run("assigns organizations to team without error", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessEnterprise(t) },
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
							organization_slugs = [%q]
						}
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID, orgSlug),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_enterprise_team_organizations.test", tfjsonpath.New("organization_slugs"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue("github_enterprise_team_organizations.test", tfjsonpath.New("organization_slugs"), knownvalue.SetPartial([]knownvalue.Check{knownvalue.StringExact(orgSlug)})),
					},
				},
			},
		})
	})

	t.Run("imports resource without error", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessEnterprise(t) },
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
							organization_slugs = [%q]
						}
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID, orgSlug),
				},
				{
					ResourceName:      "github_enterprise_team_organizations.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}

func TestAccGithubEnterpriseTeamMembership(t *testing.T) {
	username := os.Getenv("ENTERPRISE_TEST_USER")
	if username == "" {
		t.Skip("ENTERPRISE_TEST_USER not set")
	}

	t.Run("adds member to team without error", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessEnterprise(t) },
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
							username        = %q
						}
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID, username),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("github_enterprise_team_membership.test", tfjsonpath.New("username"), knownvalue.StringExact(username)),
					},
				},
			},
		})
	})

	t.Run("imports resource without error", func(t *testing.T) {
		randomID := acctest.RandString(5)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessEnterprise(t) },
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
							username        = %q
						}
					`, testAccConf.enterpriseSlug, testResourcePrefix, randomID, username),
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

func TestAccGithubEnterpriseTeamRenameAndClearDescription(t *testing.T) {
	name := testResourcePrefix + acctest.RandString(5)
	config := func(name, description string) string {
		return fmt.Sprintf(`
resource "github_enterprise_team" "test" {
  enterprise_slug = %q
  name = %q
  %s
}
`, testAccConf.enterpriseSlug, name, description)
	}
	sameTeamID := statecheck.CompareValue(compare.ValuesSame())
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { skipUnlessEnterprise(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{Config: config(name, `description = "Remove this description"`), ConfigStateChecks: []statecheck.StateCheck{sameTeamID.AddStateValue("github_enterprise_team.test", tfjsonpath.New("team_id"))}},
			{
				Config: config(name+"-renamed", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					sameTeamID.AddStateValue("github_enterprise_team.test", tfjsonpath.New("team_id")),
					statecheck.ExpectKnownValue("github_enterprise_team.test", tfjsonpath.New("name"), knownvalue.StringExact(name+"-renamed")),
					statecheck.ExpectKnownValue("github_enterprise_team.test", tfjsonpath.New("description"), knownvalue.StringExact("")),
				},
			},
			{Config: config(name+"-renamed", ""), PlanOnly: true},
		},
	})
}

func TestAccGithubEnterpriseTeamOrganizationsUpdateAndRename(t *testing.T) {
	orgA, orgB := os.Getenv("ENTERPRISE_TEST_ORGANIZATION"), os.Getenv("ENTERPRISE_TEST_ORGANIZATION_2")
	if orgA == "" || orgB == "" || orgA == orgB {
		t.Skip("two distinct organizations in the test enterprise are required: ENTERPRISE_TEST_ORGANIZATION and ENTERPRISE_TEST_ORGANIZATION_2")
	}
	name := testResourcePrefix + acctest.RandString(5)
	config := func(teamName, org string) string {
		return fmt.Sprintf(`
resource "github_enterprise_team" "test" {
  enterprise_slug = %q
  name = %q
  organization_selection_type = "selected"
}
resource "github_enterprise_team_organizations" "test" {
  enterprise_slug = %q
  team_id = github_enterprise_team.test.team_id
  organization_slugs = [%q]
}
`, testAccConf.enterpriseSlug, teamName, testAccConf.enterpriseSlug, org)
	}
	sameTeamID := statecheck.CompareValue(compare.ValuesSame())
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { skipUnlessEnterprise(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{Config: config(name, orgA), ConfigStateChecks: []statecheck.StateCheck{sameTeamID.AddStateValue("github_enterprise_team_organizations.test", tfjsonpath.New("resolved_team_id"))}},
			{
				Config: config(name+"-renamed", orgB),
				ConfigStateChecks: []statecheck.StateCheck{
					sameTeamID.AddStateValue("github_enterprise_team_organizations.test", tfjsonpath.New("resolved_team_id")),
					statecheck.ExpectKnownValue("github_enterprise_team_organizations.test", tfjsonpath.New("organization_slugs"), knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact(orgB)})),
				},
			},
			{Config: config(name+"-renamed", orgB), PlanOnly: true},
		},
	})
}
