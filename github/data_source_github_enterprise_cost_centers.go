package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v81/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceGithubEnterpriseCostCenters() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGithubEnterpriseCostCentersRead,

		Schema: map[string]*schema.Schema{
			"enterprise_slug": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The slug of the enterprise.",
			},
			"state": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: toDiagFunc(validation.StringInSlice([]string{"active", "deleted"}, false), "state"),
				Description:      "Filter cost centers by state.",
			},
			"cost_centers": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"state": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"azure_subscription": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"resources": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {Type: schema.TypeString, Computed: true},
									"name": {Type: schema.TypeString, Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceGithubEnterpriseCostCentersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	var state *string
	if v, ok := d.GetOk("state"); ok {
		s := v.(string)
		state = &s
	}

	ctx = context.WithValue(ctx, ctxId, fmt.Sprintf("%s/cost-centers", enterpriseSlug))
	result, _, err := client.Enterprise.ListCostCenters(ctx, enterpriseSlug, &github.ListCostCenterOptions{State: state})
	if err != nil {
		return diag.FromErr(err)
	}

	items := make([]any, 0, len(result.CostCenters))
	for _, cc := range result.CostCenters {
		if cc == nil {
			continue
		}
		resources := make([]map[string]any, 0)
		for _, r := range cc.Resources {
			if r == nil {
				continue
			}
			resources = append(resources, map[string]any{"type": r.Type, "name": r.Name})
		}
		items = append(items, map[string]any{
			"id":                 cc.ID,
			"name":               cc.Name,
			"state":              cc.GetState(),
			"azure_subscription": cc.GetAzureSubscription(),
			"resources":          resources,
		})
	}

	stateStr := ""
	if state != nil {
		stateStr = *state
	}
	d.SetId(fmt.Sprintf("%s/%s", enterpriseSlug, stateStr))
	_ = d.Set("cost_centers", items)
	return nil
}
