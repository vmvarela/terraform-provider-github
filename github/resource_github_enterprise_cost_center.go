package github

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGithubEnterpriseCostCenter() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGithubEnterpriseCostCenterCreate,
		ReadContext:   resourceGithubEnterpriseCostCenterRead,
		UpdateContext: resourceGithubEnterpriseCostCenterUpdate,
		DeleteContext: resourceGithubEnterpriseCostCenterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGithubEnterpriseCostCenterImport,
		},

		Schema: map[string]*schema.Schema{
			"enterprise_slug": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The slug of the enterprise.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
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
			"resources": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The resource type.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The resource identifier (username, organization name, or repo full name).",
						},
					},
				},
			},
		},
	}
}

func resourceGithubEnterpriseCostCenterCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	name := d.Get("name").(string)

	ctx = context.WithValue(ctx, ctxId, fmt.Sprintf("%s/%s", enterpriseSlug, name))
	log.Printf("[INFO] Creating enterprise cost center: %s (%s)", name, enterpriseSlug)

	cc, err := enterpriseCostCenterCreate(ctx, client, enterpriseSlug, name)
	if err != nil {
		return diag.FromErr(err)
	}

	if cc == nil || cc.ID == "" {
		return diag.FromErr(fmt.Errorf("failed to create cost center: missing id in response"))
	}

	d.SetId(cc.ID)
	return resourceGithubEnterpriseCostCenterRead(ctx, d, meta)
}

func resourceGithubEnterpriseCostCenterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Id()

	ctx = context.WithValue(ctx, ctxId, fmt.Sprintf("%s/%s", enterpriseSlug, costCenterID))

	cc, err := enterpriseCostCenterGet(ctx, client, enterpriseSlug, costCenterID)
	if err != nil {
		if is404(err) {
			// If the API starts returning 404 for archived cost centers, we remove it from state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", cc.Name)

	state := strings.ToLower(cc.State)
	if state == "" {
		state = "active"
	}
	_ = d.Set("state", state)
	_ = d.Set("azure_subscription", cc.AzureSubscription)

	resources := make([]map[string]any, 0)
	for _, r := range cc.Resources {
		resources = append(resources, map[string]any{
			"type": r.Type,
			"name": r.Name,
		})
	}
	_ = d.Set("resources", resources)

	return nil
}

func resourceGithubEnterpriseCostCenterUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Id()

	ctx = context.WithValue(ctx, ctxId, fmt.Sprintf("%s/%s", enterpriseSlug, costCenterID))

	cc, err := enterpriseCostCenterGet(ctx, client, enterpriseSlug, costCenterID)
	if err != nil {
		return diag.FromErr(err)
	}
	if strings.EqualFold(cc.State, "deleted") {
		return diag.FromErr(fmt.Errorf("cannot update cost center %q because it is archived", costCenterID))
	}

	if d.HasChange("name") {
		name := d.Get("name").(string)
		log.Printf("[INFO] Updating enterprise cost center: %s/%s", enterpriseSlug, costCenterID)
		_, err := enterpriseCostCenterUpdate(ctx, client, enterpriseSlug, costCenterID, name)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceGithubEnterpriseCostCenterRead(ctx, d, meta)
}

func resourceGithubEnterpriseCostCenterDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Id()

	ctx = context.WithValue(ctx, ctxId, fmt.Sprintf("%s/%s", enterpriseSlug, costCenterID))
	log.Printf("[INFO] Archiving enterprise cost center: %s/%s", enterpriseSlug, costCenterID)

	_, err := enterpriseCostCenterArchive(ctx, client, enterpriseSlug, costCenterID)
	if err != nil {
		if is404(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubEnterpriseCostCenterImport(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid import specified: supplied import must be written as <enterprise_slug>/<cost_center_id>")
	}

	enterpriseSlug, costCenterID := parts[0], parts[1]
	d.SetId(costCenterID)
	_ = d.Set("enterprise_slug", enterpriseSlug)

	return []*schema.ResourceData{d}, nil
}
