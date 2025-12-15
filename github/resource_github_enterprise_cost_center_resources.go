package github

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGithubEnterpriseCostCenterResources() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubEnterpriseCostCenterResourcesCreate,
		Read:   resourceGithubEnterpriseCostCenterResourcesRead,
		Update: resourceGithubEnterpriseCostCenterResourcesUpdate,
		Delete: resourceGithubEnterpriseCostCenterResourcesDelete,
		Importer: &schema.ResourceImporter{
			State: resourceGithubEnterpriseCostCenterResourcesImport,
		},

		Schema: map[string]*schema.Schema{
			"enterprise_slug": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The slug of the enterprise.",
			},
			"cost_center_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The cost center ID.",
			},
			"users": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The usernames assigned to this cost center.",
			},
			"organizations": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The organization logins assigned to this cost center.",
			},
			"repositories": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The repositories (full name) assigned to this cost center.",
			},
		},
	}
}

func resourceGithubEnterpriseCostCenterResourcesCreate(d *schema.ResourceData, meta any) error {
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Get("cost_center_id").(string)

	d.SetId(buildTwoPartID(enterpriseSlug, costCenterID))
	return resourceGithubEnterpriseCostCenterResourcesUpdate(d, meta)
}

func resourceGithubEnterpriseCostCenterResourcesRead(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Get("cost_center_id").(string)

	ctx := context.WithValue(context.Background(), ctxId, fmt.Sprintf("%s/%s", enterpriseSlug, costCenterID))

	cc, err := enterpriseCostCenterGet(ctx, client, enterpriseSlug, costCenterID)
	if err != nil {
		if is404(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	users, orgs, repos := enterpriseCostCenterSplitResources(cc.Resources)
	sort.Strings(users)
	sort.Strings(orgs)
	sort.Strings(repos)

	_ = d.Set("users", stringSliceToAnySlice(users))
	_ = d.Set("organizations", stringSliceToAnySlice(orgs))
	_ = d.Set("repositories", stringSliceToAnySlice(repos))

	return nil
}

func resourceGithubEnterpriseCostCenterResourcesUpdate(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Get("cost_center_id").(string)

	ctx := context.WithValue(context.Background(), ctxId, fmt.Sprintf("%s/%s", enterpriseSlug, costCenterID))

	cc, err := enterpriseCostCenterGet(ctx, client, enterpriseSlug, costCenterID)
	if err != nil {
		if is404(err) {
			return fmt.Errorf("cost center %q not found in enterprise %q (check enterprise_slug matches the cost center's enterprise)", costCenterID, enterpriseSlug)
		}
		return err
	}
	if strings.EqualFold(cc.State, "deleted") {
		return fmt.Errorf("cannot modify cost center %q resources because it is archived", costCenterID)
	}

	desiredUsers := expandStringSet(getStringSetOrEmpty(d, "users"))
	desiredOrgs := expandStringSet(getStringSetOrEmpty(d, "organizations"))
	desiredRepos := expandStringSet(getStringSetOrEmpty(d, "repositories"))

	currentUsers, currentOrgs, currentRepos := enterpriseCostCenterSplitResources(cc.Resources)

	toAddUsers, toRemoveUsers := diffStringSlices(currentUsers, desiredUsers)
	toAddOrgs, toRemoveOrgs := diffStringSlices(currentOrgs, desiredOrgs)
	toAddRepos, toRemoveRepos := diffStringSlices(currentRepos, desiredRepos)

	if len(toRemoveUsers)+len(toRemoveOrgs)+len(toRemoveRepos) > 0 {
		log.Printf("[INFO] Removing enterprise cost center resources: %s/%s", enterpriseSlug, costCenterID)
		_, err := enterpriseCostCenterRemoveResources(ctx, client, enterpriseSlug, costCenterID, enterpriseCostCenterResourcesRequest{
			Users:         toRemoveUsers,
			Organizations: toRemoveOrgs,
			Repositories:  toRemoveRepos,
		})
		if err != nil {
			return err
		}
	}

	if len(toAddUsers)+len(toAddOrgs)+len(toAddRepos) > 0 {
		log.Printf("[INFO] Assigning enterprise cost center resources: %s/%s", enterpriseSlug, costCenterID)
		_, err := enterpriseCostCenterAssignResources(ctx, client, enterpriseSlug, costCenterID, enterpriseCostCenterResourcesRequest{
			Users:         toAddUsers,
			Organizations: toAddOrgs,
			Repositories:  toAddRepos,
		})
		if err != nil {
			return err
		}
	}

	return resourceGithubEnterpriseCostCenterResourcesRead(d, meta)
}

func resourceGithubEnterpriseCostCenterResourcesDelete(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Get("cost_center_id").(string)

	ctx := context.WithValue(context.Background(), ctxId, fmt.Sprintf("%s/%s", enterpriseSlug, costCenterID))

	cc, err := enterpriseCostCenterGet(ctx, client, enterpriseSlug, costCenterID)
	if err != nil {
		if is404(err) {
			return nil
		}
		return err
	}

	// If the cost center is archived, treat deletion as a no-op.
	if strings.EqualFold(cc.State, "deleted") {
		return nil
	}

	users, orgs, repos := enterpriseCostCenterSplitResources(cc.Resources)
	if len(users)+len(orgs)+len(repos) == 0 {
		return nil
	}

	_, err = enterpriseCostCenterRemoveResources(ctx, client, enterpriseSlug, costCenterID, enterpriseCostCenterResourcesRequest{
		Users:         users,
		Organizations: orgs,
		Repositories:  repos,
	})
	return err
}

func resourceGithubEnterpriseCostCenterResourcesImport(d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid import specified: supplied import must be written as <enterprise_slug>/<cost_center_id>")
	}

	enterpriseSlug, costCenterID := parts[0], parts[1]
	_ = d.Set("enterprise_slug", enterpriseSlug)
	_ = d.Set("cost_center_id", costCenterID)
	d.SetId(buildTwoPartID(enterpriseSlug, costCenterID))

	return []*schema.ResourceData{d}, nil
}

func expandStringSet(set *schema.Set) []string {
	if set == nil {
		return nil
	}

	list := set.List()
	out := make([]string, 0, len(list))
	for _, v := range list {
		out = append(out, v.(string))
	}
	sort.Strings(out)
	return out
}

func getStringSetOrEmpty(d *schema.ResourceData, key string) *schema.Set {
	v, ok := d.GetOk(key)
	if !ok || v == nil {
		return schema.NewSet(schema.HashString, []any{})
	}

	set, ok := v.(*schema.Set)
	if !ok || set == nil {
		return schema.NewSet(schema.HashString, []any{})
	}

	return set
}

func diffStringSlices(current []string, desired []string) (toAdd []string, toRemove []string) {
	cur := schema.NewSet(schema.HashString, stringSliceToAnySlice(current))
	des := schema.NewSet(schema.HashString, stringSliceToAnySlice(desired))

	for _, v := range des.Difference(cur).List() {
		toAdd = append(toAdd, v.(string))
	}
	for _, v := range cur.Difference(des).List() {
		toRemove = append(toRemove, v.(string))
	}

	sort.Strings(toAdd)
	sort.Strings(toRemove)
	return toAdd, toRemove
}
