package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGithubEnterpriseCostCenter() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages an enterprise cost center in GitHub.",
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
		},
	}
}

func resourceGithubEnterpriseCostCenterCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	name := d.Get("name").(string)

	tflog.Info(ctx, "Creating enterprise cost center", map[string]any{
		"enterprise_slug": enterpriseSlug,
		"name":            name,
	})

	cc, _, err := client.Enterprise.CreateCostCenter(ctx, enterpriseSlug, github.CostCenterRequest{Name: name})
	if err != nil {
		return diag.FromErr(err)
	}

	if cc == nil || cc.ID == "" {
		return diag.Errorf("failed to create cost center: missing id in response")
	}

	d.SetId(cc.ID)

	if hasCostCenterAssignmentsConfigured(d) {
		if diags := syncEnterpriseCostCenterAssignments(ctx, d, client, enterpriseSlug, cc.ID, nil); diags.HasError() {
			return diags
		}
	}

	// Set computed fields from the API response
	state := strings.ToLower(cc.GetState())
	if state == "" {
		state = "active"
	}
	_ = d.Set("state", state)
	_ = d.Set("azure_subscription", cc.GetAzureSubscription())

	return nil
}

func resourceGithubEnterpriseCostCenterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Id()

	cc, _, err := client.Enterprise.GetCostCenter(ctx, enterpriseSlug, costCenterID)
	if err != nil {
		if is404(err) {
			tflog.Warn(ctx, "Cost center not found, removing from state", map[string]any{
				"enterprise_slug": enterpriseSlug,
				"cost_center_id":  costCenterID,
			})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", cc.Name)

	state := strings.ToLower(cc.GetState())
	if state == "" {
		state = "active"
	}
	_ = d.Set("state", state)
	_ = d.Set("azure_subscription", cc.GetAzureSubscription())

	setCostCenterResourceFields(d, cc)

	return nil
}

func resourceGithubEnterpriseCostCenterUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Id()

	// Check current state to prevent updates on archived cost centers
	currentState := d.Get("state").(string)
	if strings.EqualFold(currentState, "deleted") {
		return diag.Errorf("cannot update cost center %q because it is archived", costCenterID)
	}

	var updatedCC *github.CostCenter

	if d.HasChange("name") {
		name := d.Get("name").(string)
		tflog.Info(ctx, "Updating enterprise cost center name", map[string]any{
			"enterprise_slug": enterpriseSlug,
			"cost_center_id":  costCenterID,
			"name":            name,
		})
		cc, _, err := client.Enterprise.UpdateCostCenter(ctx, enterpriseSlug, costCenterID, github.CostCenterRequest{Name: name})
		if err != nil {
			return diag.FromErr(err)
		}
		updatedCC = cc
	}

	if d.HasChange("users") || d.HasChange("organizations") || d.HasChange("repositories") {
		// Get current resources from API only if we need to sync assignments
		var currentResources []*github.CostCenterResource
		if updatedCC != nil {
			currentResources = updatedCC.Resources
		} else {
			cc, _, err := client.Enterprise.GetCostCenter(ctx, enterpriseSlug, costCenterID)
			if err != nil {
				return diag.FromErr(err)
			}
			currentResources = cc.Resources
		}
		if diags := syncEnterpriseCostCenterAssignments(ctx, d, client, enterpriseSlug, costCenterID, currentResources); diags.HasError() {
			return diags
		}
	}

	// Fetch final state to set computed fields
	final, _, err := client.Enterprise.GetCostCenter(ctx, enterpriseSlug, costCenterID)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("name", final.Name)
	state := strings.ToLower(final.GetState())
	if state == "" {
		state = "active"
	}
	_ = d.Set("state", state)
	_ = d.Set("azure_subscription", final.GetAzureSubscription())
	setCostCenterResourceFields(d, final)

	return nil
}

func resourceGithubEnterpriseCostCenterDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := d.Get("enterprise_slug").(string)
	costCenterID := d.Id()

	tflog.Info(ctx, "Archiving enterprise cost center", map[string]any{
		"enterprise_slug": enterpriseSlug,
		"cost_center_id":  costCenterID,
	})

	_, _, err := client.Enterprise.DeleteCostCenter(ctx, enterpriseSlug, costCenterID)
	if err != nil {
		if is404(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubEnterpriseCostCenterImport(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	enterpriseSlug, costCenterID, err := parseTwoPartID(d.Id(), "enterprise_slug", "cost_center_id")
	if err != nil {
		return nil, fmt.Errorf("invalid import ID %q: expected format <enterprise_slug>:<cost_center_id>", d.Id())
	}

	d.SetId(costCenterID)
	_ = d.Set("enterprise_slug", enterpriseSlug)

	return []*schema.ResourceData{d}, nil
}

func syncEnterpriseCostCenterAssignments(ctx context.Context, d *schema.ResourceData, client *github.Client, enterpriseSlug, costCenterID string, currentResources []*github.CostCenterResource) diag.Diagnostics {
	desiredUsers := expandStringSet(getStringSetOrEmpty(d, "users"))
	desiredOrgs := expandStringSet(getStringSetOrEmpty(d, "organizations"))
	desiredRepos := expandStringSet(getStringSetOrEmpty(d, "repositories"))

	currentUsers, currentOrgs, currentRepos := costCenterSplitResources(currentResources)

	toAddUsers, toRemoveUsers := diffStringSlices(currentUsers, desiredUsers)
	toAddOrgs, toRemoveOrgs := diffStringSlices(currentOrgs, desiredOrgs)
	toAddRepos, toRemoveRepos := diffStringSlices(currentRepos, desiredRepos)

	if len(toRemoveUsers)+len(toRemoveOrgs)+len(toRemoveRepos) > 0 {
		tflog.Info(ctx, "Removing enterprise cost center resources", map[string]any{
			"enterprise_slug": enterpriseSlug,
			"cost_center_id":  costCenterID,
		})

		for _, batch := range chunkStringSlice(toRemoveUsers) {
			if diags := retryCostCenterRemoveResources(ctx, client, enterpriseSlug, costCenterID, github.CostCenterResourceRequest{Users: batch}); diags.HasError() {
				return diags
			}
		}
		for _, batch := range chunkStringSlice(toRemoveOrgs) {
			if diags := retryCostCenterRemoveResources(ctx, client, enterpriseSlug, costCenterID, github.CostCenterResourceRequest{Organizations: batch}); diags.HasError() {
				return diags
			}
		}
		for _, batch := range chunkStringSlice(toRemoveRepos) {
			if diags := retryCostCenterRemoveResources(ctx, client, enterpriseSlug, costCenterID, github.CostCenterResourceRequest{Repositories: batch}); diags.HasError() {
				return diags
			}
		}
	}

	if len(toAddUsers)+len(toAddOrgs)+len(toAddRepos) > 0 {
		tflog.Info(ctx, "Assigning enterprise cost center resources", map[string]any{
			"enterprise_slug": enterpriseSlug,
			"cost_center_id":  costCenterID,
		})

		for _, batch := range chunkStringSlice(toAddUsers) {
			if diags := retryCostCenterAddResources(ctx, client, enterpriseSlug, costCenterID, github.CostCenterResourceRequest{Users: batch}); diags.HasError() {
				return diags
			}
		}
		for _, batch := range chunkStringSlice(toAddOrgs) {
			if diags := retryCostCenterAddResources(ctx, client, enterpriseSlug, costCenterID, github.CostCenterResourceRequest{Organizations: batch}); diags.HasError() {
				return diags
			}
		}
		for _, batch := range chunkStringSlice(toAddRepos) {
			if diags := retryCostCenterAddResources(ctx, client, enterpriseSlug, costCenterID, github.CostCenterResourceRequest{Repositories: batch}); diags.HasError() {
				return diags
			}
		}
	}

	return nil
}

func hasCostCenterAssignmentsConfigured(d *schema.ResourceData) bool {
	assignmentKeys := []string{"users", "organizations", "repositories"}
	for _, key := range assignmentKeys {
		if v, ok := d.GetOk(key); ok {
			if set, ok := v.(*schema.Set); ok && set != nil && set.Len() > 0 {
				return true
			}
		}
	}
	return false
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

func diffStringSlices(current, desired []string) (toAdd, toRemove []string) {
	cur := schema.NewSet(schema.HashString, flattenStringList(current))
	des := schema.NewSet(schema.HashString, flattenStringList(desired))

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

func isRetryableGithubResponseError(err error) bool {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil {
		switch ghErr.Response.StatusCode {
		case http.StatusConflict, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}
	return false
}

func costCenterSplitResources(resources []*github.CostCenterResource) (users, organizations, repositories []string) {
	for _, r := range resources {
		if r == nil {
			continue
		}
		switch strings.ToLower(r.Type) {
		case "user":
			users = append(users, r.Name)
		case "org", "organization":
			organizations = append(organizations, r.Name)
		case "repo", "repository":
			repositories = append(repositories, r.Name)
		}
	}
	return users, organizations, repositories
}

// setCostCenterResourceFields sets the resource-related fields on the schema.ResourceData.
func setCostCenterResourceFields(d *schema.ResourceData, cc *github.CostCenter) {
	users, organizations, repositories := costCenterSplitResources(cc.Resources)
	sort.Strings(users)
	sort.Strings(organizations)
	sort.Strings(repositories)
	_ = d.Set("users", flattenStringList(users))
	_ = d.Set("organizations", flattenStringList(organizations))
	_ = d.Set("repositories", flattenStringList(repositories))
}

// Cost center resource management constants and retry functions.
const (
	maxResourcesPerRequest          = 50
	costCenterResourcesRetryTimeout = 5 * time.Minute
)

// chunkStringSlice splits a slice into chunks of the max resources per request.
func chunkStringSlice(items []string) [][]string {
	if len(items) == 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(items)+maxResourcesPerRequest-1)/maxResourcesPerRequest)
	for start := 0; start < len(items); start += maxResourcesPerRequest {
		end := min(start+maxResourcesPerRequest, len(items))
		chunks = append(chunks, items[start:end])
	}
	return chunks
}

// retryCostCenterRemoveResources removes resources from a cost center with retry logic.
// Uses retry.RetryContext for exponential backoff on transient errors.
func retryCostCenterRemoveResources(ctx context.Context, client *github.Client, enterpriseSlug, costCenterID string, req github.CostCenterResourceRequest) diag.Diagnostics {
	err := retry.RetryContext(ctx, costCenterResourcesRetryTimeout, func() *retry.RetryError {
		_, _, err := client.Enterprise.RemoveResourcesFromCostCenter(ctx, enterpriseSlug, costCenterID, req)
		if err == nil {
			return nil
		}
		if isRetryableGithubResponseError(err) {
			return retry.RetryableError(err)
		}
		return retry.NonRetryableError(err)
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// retryCostCenterAddResources adds resources to a cost center with retry logic.
// Uses retry.RetryContext for exponential backoff on transient errors.
func retryCostCenterAddResources(ctx context.Context, client *github.Client, enterpriseSlug, costCenterID string, req github.CostCenterResourceRequest) diag.Diagnostics {
	err := retry.RetryContext(ctx, costCenterResourcesRetryTimeout, func() *retry.RetryError {
		_, _, err := client.Enterprise.AddResourcesToCostCenter(ctx, enterpriseSlug, costCenterID, req)
		if err == nil {
			return nil
		}
		if isRetryableGithubResponseError(err) {
			return retry.RetryableError(err)
		}
		return retry.NonRetryableError(err)
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
