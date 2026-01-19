package github

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGithubEnterpriseCostCenter() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to retrieve information about a specific enterprise cost center.",
		ReadContext: dataSourceGithubEnterpriseCostCenterRead,

		Schema: map[string]*schema.Schema{
			"enterprise_slug": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The slug of the enterprise.",
			},
			"cost_center_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the cost center.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the cost center.",
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The state of the cost center.",
			},
			"azure_subscription": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Azure subscription associated with the cost center.",
			},
			"users": {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The usernames assigned to this cost center.",
			},
			"organizations": {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The organization logins assigned to this cost center.",
			},
			"repositories": {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The repositories (full name) assigned to this cost center.",
			},
		},
	}
}

func dataSourceGithubEnterpriseCostCenterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Get("cost_center_id").(string)

	cc, _, err := client.Enterprise.GetCostCenter(ctx, enterpriseSlug, costCenterID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(costCenterID)
	_ = d.Set("name", cc.Name)

	state := strings.ToLower(cc.GetState())
	if state == "" {
		state = "active"
	}
	_ = d.Set("state", state)
	_ = d.Set("azure_subscription", cc.GetAzureSubscription())

	users, organizations, repositories := costCenterSplitResources(cc.Resources)
	sort.Strings(users)
	sort.Strings(organizations)
	sort.Strings(repositories)
	_ = d.Set("users", flattenStringList(users))
	_ = d.Set("organizations", flattenStringList(organizations))
	_ = d.Set("repositories", flattenStringList(repositories))

	return nil
}
