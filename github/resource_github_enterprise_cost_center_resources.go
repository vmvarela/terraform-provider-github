package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGithubEnterpriseCostCenterResources() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGithubEnterpriseCostCenterResourcesCreate,
		ReadContext:   resourceGithubEnterpriseCostCenterResourcesRead,
		UpdateContext: resourceGithubEnterpriseCostCenterResourcesUpdate,
		DeleteContext: resourceGithubEnterpriseCostCenterResourcesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGithubEnterpriseCostCenterResourcesImport,
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

func resourceGithubEnterpriseCostCenterResourcesCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Get("cost_center_id").(string)

	d.SetId(buildTwoPartID(enterpriseSlug, costCenterID))
	return resourceGithubEnterpriseCostCenterResourcesUpdate(ctx, d, meta)
}

func resourceGithubEnterpriseCostCenterResourcesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug, costCenterID, err := parseTwoPartID(d.Id(), "enterprise_slug", "cost_center_id")
	if err != nil {
		return diag.FromErr(err)
	}

	ctx = context.WithValue(ctx, ctxId, d.Id())

	cc, err := enterpriseCostCenterGet(ctx, client, enterpriseSlug, costCenterID)
	if err != nil {
		if is404(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	users, orgs, repos := enterpriseCostCenterSplitResources(cc.Resources)
	sort.Strings(users)
	sort.Strings(orgs)
	sort.Strings(repos)

	_ = d.Set("enterprise_slug", enterpriseSlug)
	_ = d.Set("cost_center_id", costCenterID)

	_ = d.Set("users", stringSliceToAnySlice(users))
	_ = d.Set("organizations", stringSliceToAnySlice(orgs))
	_ = d.Set("repositories", stringSliceToAnySlice(repos))

	return nil
}

func resourceGithubEnterpriseCostCenterResourcesUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug, costCenterID, err := parseTwoPartID(d.Id(), "enterprise_slug", "cost_center_id")
	if err != nil {
		return diag.FromErr(err)
	}

	ctx = context.WithValue(ctx, ctxId, d.Id())

	cc, err := enterpriseCostCenterGet(ctx, client, enterpriseSlug, costCenterID)
	if err != nil {
		if is404(err) {
			return diag.FromErr(fmt.Errorf("cost center %q not found in enterprise %q (check enterprise_slug matches the cost center's enterprise)", costCenterID, enterpriseSlug))
		}
		return diag.FromErr(err)
	}
	if strings.EqualFold(cc.State, "deleted") {
		return diag.FromErr(fmt.Errorf("cannot modify cost center %q resources because it is archived", costCenterID))
	}

	desiredUsers := expandStringSet(getStringSetOrEmpty(d, "users"))
	desiredOrgs := expandStringSet(getStringSetOrEmpty(d, "organizations"))
	desiredRepos := expandStringSet(getStringSetOrEmpty(d, "repositories"))

	currentUsers, currentOrgs, currentRepos := enterpriseCostCenterSplitResources(cc.Resources)

	toAddUsers, toRemoveUsers := diffStringSlices(currentUsers, desiredUsers)
	toAddOrgs, toRemoveOrgs := diffStringSlices(currentOrgs, desiredOrgs)
	toAddRepos, toRemoveRepos := diffStringSlices(currentRepos, desiredRepos)

	const maxResourcesPerRequest = 50
	const costCenterResourcesRetryTimeout = 5 * time.Minute

	retryRemove := func(req enterpriseCostCenterResourcesRequest) error {
		return resource.RetryContext(ctx, costCenterResourcesRetryTimeout, func() *resource.RetryError {
			_, err := enterpriseCostCenterRemoveResources(ctx, client, enterpriseSlug, costCenterID, req)
			if err == nil {
				return nil
			}
			if isRetryableGithubResponseError(err) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		})
	}

	retryAssign := func(req enterpriseCostCenterResourcesRequest) error {
		return resource.RetryContext(ctx, costCenterResourcesRetryTimeout, func() *resource.RetryError {
			_, err := enterpriseCostCenterAssignResources(ctx, client, enterpriseSlug, costCenterID, req)
			if err == nil {
				return nil
			}
			if isRetryableGithubResponseError(err) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		})
	}

	chunk := func(items []string, size int) [][]string {
		if len(items) == 0 {
			return nil
		}
		if size <= 0 {
			size = len(items)
		}
		chunks := make([][]string, 0, (len(items)+size-1)/size)
		for start := 0; start < len(items); start += size {
			end := start + size
			if end > len(items) {
				end = len(items)
			}
			chunks = append(chunks, items[start:end])
		}
		return chunks
	}

	if len(toRemoveUsers)+len(toRemoveOrgs)+len(toRemoveRepos) > 0 {
		log.Printf("[INFO] Removing enterprise cost center resources: %s/%s", enterpriseSlug, costCenterID)

		for _, batch := range chunk(toRemoveUsers, maxResourcesPerRequest) {
			if err := retryRemove(enterpriseCostCenterResourcesRequest{Users: batch}); err != nil {
				return diag.FromErr(err)
			}
		}
		for _, batch := range chunk(toRemoveOrgs, maxResourcesPerRequest) {
			if err := retryRemove(enterpriseCostCenterResourcesRequest{Organizations: batch}); err != nil {
				return diag.FromErr(err)
			}
		}
		for _, batch := range chunk(toRemoveRepos, maxResourcesPerRequest) {
			if err := retryRemove(enterpriseCostCenterResourcesRequest{Repositories: batch}); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if len(toAddUsers)+len(toAddOrgs)+len(toAddRepos) > 0 {
		log.Printf("[INFO] Assigning enterprise cost center resources: %s/%s", enterpriseSlug, costCenterID)

		for _, batch := range chunk(toAddUsers, maxResourcesPerRequest) {
			if err := retryAssign(enterpriseCostCenterResourcesRequest{Users: batch}); err != nil {
				return diag.FromErr(err)
			}
		}
		for _, batch := range chunk(toAddOrgs, maxResourcesPerRequest) {
			if err := retryAssign(enterpriseCostCenterResourcesRequest{Organizations: batch}); err != nil {
				return diag.FromErr(err)
			}
		}
		for _, batch := range chunk(toAddRepos, maxResourcesPerRequest) {
			if err := retryAssign(enterpriseCostCenterResourcesRequest{Repositories: batch}); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return resourceGithubEnterpriseCostCenterResourcesRead(ctx, d, meta)
}

func resourceGithubEnterpriseCostCenterResourcesDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug, costCenterID, err := parseTwoPartID(d.Id(), "enterprise_slug", "cost_center_id")
	if err != nil {
		return diag.FromErr(err)
	}

	ctx = context.WithValue(ctx, ctxId, d.Id())

	cc, err := enterpriseCostCenterGet(ctx, client, enterpriseSlug, costCenterID)
	if err != nil {
		if is404(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	// If the cost center is archived, treat deletion as a no-op.
	if strings.EqualFold(cc.State, "deleted") {
		return nil
	}

	users, orgs, repos := enterpriseCostCenterSplitResources(cc.Resources)
	if len(users)+len(orgs)+len(repos) == 0 {
		return nil
	}

	const maxResourcesPerRequest = 50
	const costCenterResourcesRetryTimeout = 5 * time.Minute

	retryRemove := func(req enterpriseCostCenterResourcesRequest) error {
		return resource.RetryContext(ctx, costCenterResourcesRetryTimeout, func() *resource.RetryError {
			_, err := enterpriseCostCenterRemoveResources(ctx, client, enterpriseSlug, costCenterID, req)
			if err == nil {
				return nil
			}
			if isRetryableGithubResponseError(err) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		})
	}

	chunk := func(items []string, size int) [][]string {
		if len(items) == 0 {
			return nil
		}
		if size <= 0 {
			size = len(items)
		}
		chunks := make([][]string, 0, (len(items)+size-1)/size)
		for start := 0; start < len(items); start += size {
			end := start + size
			if end > len(items) {
				end = len(items)
			}
			chunks = append(chunks, items[start:end])
		}
		return chunks
	}

	log.Printf("[INFO] Removing all enterprise cost center resources: %s/%s", enterpriseSlug, costCenterID)

	for _, batch := range chunk(users, maxResourcesPerRequest) {
		if err := retryRemove(enterpriseCostCenterResourcesRequest{Users: batch}); err != nil {
			return diag.FromErr(err)
		}
	}
	for _, batch := range chunk(orgs, maxResourcesPerRequest) {
		if err := retryRemove(enterpriseCostCenterResourcesRequest{Organizations: batch}); err != nil {
			return diag.FromErr(err)
		}
	}
	for _, batch := range chunk(repos, maxResourcesPerRequest) {
		if err := retryRemove(enterpriseCostCenterResourcesRequest{Repositories: batch}); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGithubEnterpriseCostCenterResourcesImport(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
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

func isRetryableGithubResponseError(err error) bool {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil {
		switch ghErr.Response.StatusCode {
		case 404, 409, 500, 502, 503, 504:
			return true
		default:
			return false
		}
	}
	return false
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
