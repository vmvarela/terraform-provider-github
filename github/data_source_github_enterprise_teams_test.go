package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubEnterpriseTeamsDataSource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_team" "test" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			name            = "%s%s"
		}

		data "github_enterprise_teams" "all" {
			enterprise_slug = data.github_enterprise.enterprise.slug
			depends_on      = [github_enterprise_team.test]
		}
	`, testAccConf.enterpriseSlug, testResourcePrefix, randomID)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { skipUnlessEnterprise(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.github_enterprise_teams.all", "id"),
					resource.TestCheckResourceAttrSet("data.github_enterprise_teams.all", "teams.0.team_id"),
					resource.TestCheckResourceAttrSet("data.github_enterprise_teams.all", "teams.0.slug"),
					resource.TestCheckResourceAttrSet("data.github_enterprise_teams.all", "teams.0.name"),
				),
			},
		},
	})
}
