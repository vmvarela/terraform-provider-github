package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubEnterpriseSCIMUserDataSource(t *testing.T) {
	config := fmt.Sprintf(`
		data "github_enterprise_scim_users" "all" {
			enterprise = "%s"
		}

		data "github_enterprise_scim_user" "test" {
			enterprise   = "%[1]s"
			scim_user_id = data.github_enterprise_scim_users.all.resources[0].id
		}
	`, testAccConf.enterpriseSlug)

	check := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttrSet("data.github_enterprise_scim_user.test", "id"),
		resource.TestCheckResourceAttrSet("data.github_enterprise_scim_user.test", "user_name"),
		resource.TestCheckResourceAttrSet("data.github_enterprise_scim_user.test", "display_name"),
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { skipUnlessEnterprise(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: config,
			Check:  check,
		}},
	})
}
