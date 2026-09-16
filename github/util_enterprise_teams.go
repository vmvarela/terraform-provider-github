package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-github/v89/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// importEnterpriseTeamSelector preserves the selector used in the import ID.
func importEnterpriseTeamSelector(d *schema.ResourceData, enterpriseSlug, selector string) error {
	if strings.TrimSpace(enterpriseSlug) == "" || strings.TrimSpace(selector) == "" {
		return fmt.Errorf("enterprise and team selector must not be empty")
	}
	if err := d.Set("enterprise_slug", enterpriseSlug); err != nil {
		return err
	}
	id, err := strconv.Atoi(selector)
	if err == nil {
		if id <= 0 {
			return fmt.Errorf("team ID must be positive: %q", selector)
		}
		return d.Set("team_id", id)
	}
	if strings.Trim(selector, "0123456789") == "" {
		return fmt.Errorf("invalid team ID %q: %w", selector, err)
	}
	return d.Set("team_slug", selector)
}

// buildEnterpriseTeamMembershipID creates an ID for enterprise team membership resources.
// Uses "/" as separator because team slugs contain ":" (e.g., "ent:team-name").
// Note: GitHub slugs only allow alphanumeric characters, hyphens, and colons - never "/".
func buildEnterpriseTeamMembershipID(enterpriseSlug, teamSlug, username string) string {
	return fmt.Sprintf("%s/%s/%s", enterpriseSlug, teamSlug, username)
}

// parseEnterpriseTeamMembershipID parses the ID for enterprise team membership resources.
func parseEnterpriseTeamMembershipID(id string) (enterpriseSlug, teamSlug, username string, err error) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected ID format (%q); expected enterprise_slug/team_slug/username", id)
	}
	return parts[0], parts[1], parts[2], nil
}

// buildEnterpriseTeamOrganizationsID creates an ID for enterprise team organizations resources.
// Uses "/" as separator because team slugs contain ":" (e.g., "ent:team-name").
// Note: GitHub slugs only allow alphanumeric characters, hyphens, and colons - never "/".
func buildEnterpriseTeamOrganizationsID(enterpriseSlug, teamSlug string) string {
	return fmt.Sprintf("%s/%s", enterpriseSlug, teamSlug)
}

// parseEnterpriseTeamOrganizationsID parses the ID for enterprise team organizations resources.
func parseEnterpriseTeamOrganizationsID(id string) (enterpriseSlug, teamSlug string, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected ID format (%q); expected enterprise_slug/team_slug", id)
	}
	return parts[0], parts[1], nil
}

// findEnterpriseTeamByID lists all enterprise teams and returns the one matching the given ID.
// This is needed because the API doesn't provide a direct lookup by numeric ID.
func findEnterpriseTeamByID(meta *Owner, ctx context.Context, enterpriseSlug string, id int64) (*github.EnterpriseTeam, error) {
	teams, err := listAllEnterpriseTeams(meta, ctx, enterpriseSlug)
	if err != nil {
		return nil, err
	}
	for _, team := range teams {
		if team.ID == id {
			return team, nil
		}
	}
	return nil, nil
}

// resolveEnterpriseTeam fetches the enterprise team referenced by team_slug,
// falling back to a numeric team_id lookup when no slug is set (the schema
// enforces ExactlyOneOf between the two).
func resolveEnterpriseTeam(meta *Owner, ctx context.Context, enterpriseSlug string, d *schema.ResourceData) (*github.EnterpriseTeam, error) {
	if v, ok := d.GetOk("team_slug"); ok {
		team, _, err := meta.v3client.Enterprise.GetTeam(ctx, enterpriseSlug, v.(string))
		return team, err
	}
	return findEnterpriseTeamByID(meta, ctx, enterpriseSlug, int64(d.Get("team_id").(int)))
}

// findEnterpriseTeamByIdentity treats the slug as a hint, never as proof of identity.
func findEnterpriseTeamByIdentity(meta *Owner, ctx context.Context, enterpriseSlug, slug string, id int64) (*github.EnterpriseTeam, error) {
	if slug != "" {
		team, _, err := meta.v3client.Enterprise.GetTeam(ctx, enterpriseSlug, slug)
		if err != nil {
			var ghErr *github.ErrorResponse
			if !errors.As(err, &ghErr) || ghErr.Response == nil || ghErr.Response.StatusCode != http.StatusNotFound {
				return nil, err
			}
		} else if team != nil && team.ID == id {
			return team, nil
		}
	}
	return findEnterpriseTeamByID(meta, ctx, enterpriseSlug, id)
}

func resolveStoredEnterpriseTeam(meta *Owner, ctx context.Context, enterpriseSlug, slug string, d *schema.ResourceData) (*github.EnterpriseTeam, error) {
	if id, ok := d.GetOk("resolved_team_id"); ok {
		teamID, _ := id.(int)
		return findEnterpriseTeamByIdentity(meta, ctx, enterpriseSlug, slug, int64(teamID))
	}
	if id, ok := d.GetOk("team_id"); ok {
		teamID, _ := id.(int)
		// A numeric import ID has no slug hint until the first refresh.
		if slug == strconv.Itoa(teamID) {
			slug = ""
		}
		return findEnterpriseTeamByIdentity(meta, ctx, enterpriseSlug, slug, int64(teamID))
	}
	// Import and legacy state do not yet contain a numeric identity.
	team, _, err := meta.v3client.Enterprise.GetTeam(ctx, enterpriseSlug, slug)
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	return team, err
}

// organizationSlugs extracts the non-empty logins of the given organizations.
func organizationSlugs(orgs []*github.Organization) []string {
	slugs := make([]string, 0, len(orgs))
	for _, org := range orgs {
		if org.Login != nil && *org.Login != "" {
			slugs = append(slugs, *org.Login)
		}
	}
	return slugs
}

// listAllEnterpriseTeamOrganizations returns all organizations assigned to an enterprise team with pagination handled.
func listAllEnterpriseTeamOrganizations(meta *Owner, ctx context.Context, enterpriseSlug, enterpriseTeam string) ([]*github.Organization, error) {
	var all []*github.Organization
	opt := &github.ListOptions{PerPage: meta.maxPerPage}

	for {
		orgs, resp, err := meta.v3client.Enterprise.ListAssignments(ctx, enterpriseSlug, enterpriseTeam, opt)
		if err != nil {
			return nil, err
		}
		all = append(all, orgs...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return all, nil
}

// listAllEnterpriseTeams returns all enterprise teams with pagination handled.
func listAllEnterpriseTeams(meta *Owner, ctx context.Context, enterpriseSlug string) ([]*github.EnterpriseTeam, error) {
	var all []*github.EnterpriseTeam
	opt := &github.ListOptions{PerPage: meta.maxPerPage}

	for {
		teams, resp, err := meta.v3client.Enterprise.ListTeams(ctx, enterpriseSlug, opt)
		if err != nil {
			return nil, err
		}
		all = append(all, teams...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return all, nil
}
