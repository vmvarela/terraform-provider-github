package github

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceGithubEnterpriseTeamMembership() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages membership of a user in a GitHub enterprise team.",
		CreateContext: resourceGithubEnterpriseTeamMembershipCreate,
		ReadContext:   resourceGithubEnterpriseTeamMembershipRead,
		DeleteContext: resourceGithubEnterpriseTeamMembershipDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"enterprise_slug": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "The slug of the enterprise.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.All(validation.StringIsNotWhiteSpace, validation.StringIsNotEmpty)),
			},
			"enterprise_team": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "The slug or ID of the enterprise team.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.All(validation.StringIsNotWhiteSpace, validation.StringIsNotEmpty)),
			},
			"username": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "The username of the user to add to the team.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.All(validation.StringIsNotWhiteSpace, validation.StringIsNotEmpty)),
			},
			"user_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The ID of the user.",
			},
		},
	}
}

func resourceGithubEnterpriseTeamMembershipCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug := strings.TrimSpace(d.Get("enterprise_slug").(string))
	enterpriseTeam := strings.TrimSpace(d.Get("enterprise_team").(string))
	username := strings.TrimSpace(d.Get("username").(string))

	// Find the team to ensure it exists and get its slug
	team, err := findEnterpriseTeamBySlugOrID(ctx, client, enterpriseSlug, enterpriseTeam)
	if err != nil {
		return diag.FromErr(err)
	}

	// Add the user to the team using the SDK
	user, _, err := client.Enterprise.AddTeamMember(ctx, enterpriseSlug, team.Slug, username)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildThreePartID(enterpriseSlug, team.Slug, username))
	if user != nil && user.ID != nil {
		if err := d.Set("user_id", int(*user.ID)); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceGithubEnterpriseTeamMembershipRead(ctx, d, meta)
}

func resourceGithubEnterpriseTeamMembershipRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug, teamSlug, username, err := parseThreePartID(d.Id(), "enterprise_slug", "enterprise_team", "username")
	if err != nil {
		return diag.FromErr(err)
	}

	// Get the membership using the SDK
	user, resp, err := client.Enterprise.GetTeamMembership(ctx, enterpriseSlug, teamSlug, username)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if err := d.Set("enterprise_slug", enterpriseSlug); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("enterprise_team", teamSlug); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("username", username); err != nil {
		return diag.FromErr(err)
	}
	if user != nil && user.ID != nil {
		if err := d.Set("user_id", int(*user.ID)); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGithubEnterpriseTeamMembershipDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*Owner).v3client
	enterpriseSlug, teamSlug, username, err := parseThreePartID(d.Id(), "enterprise_slug", "enterprise_team", "username")
	if err != nil {
		return diag.FromErr(err)
	}

	// Remove the user from the team using the SDK
	_, err = client.Enterprise.RemoveTeamMember(ctx, enterpriseSlug, teamSlug, username)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
