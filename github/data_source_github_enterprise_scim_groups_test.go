package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubEnterpriseSCIMGroupsDataSource(t *testing.T) {
	config := fmt.Sprintf(`
		data "github_enterprise_scim_groups" "test" {
			enterprise = "%s"
		}
	`, testAccConf.enterpriseSlug)

	check := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttrSet("data.github_enterprise_scim_groups.test", "id"),
		resource.TestCheckResourceAttrSet("data.github_enterprise_scim_groups.test", "total_results"),
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
