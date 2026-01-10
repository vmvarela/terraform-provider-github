package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v81/github"
)

// buildEnterpriseTeamMembershipID creates an ID for enterprise team membership resources.
// Uses "/" as separator because team slugs contain ":" (e.g., "ent:team-name").
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
func findEnterpriseTeamByID(ctx context.Context, client *github.Client, enterpriseSlug string, id int64) (*github.EnterpriseTeam, error) {
	opt := &github.ListOptions{PerPage: maxPerPage}

	for {
		teams, resp, err := client.Enterprise.ListTeams(ctx, enterpriseSlug, opt)
		if err != nil {
			return nil, err
		}
		for _, t := range teams {
			if t.ID == id {
				return t, nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return nil, nil
}

// findEnterpriseTeamBySlugOrID finds a team by slug. If the slug looks like a numeric ID,
// it will search for a team with that ID. Otherwise, it uses GetTeam to fetch directly by slug.
func findEnterpriseTeamBySlugOrID(ctx context.Context, client *github.Client, enterpriseSlug, teamSlugOrID string) (*github.EnterpriseTeam, error) {
	// First, try to get the team directly by slug
	team, resp, err := client.Enterprise.GetTeam(ctx, enterpriseSlug, teamSlugOrID)
	if err == nil {
		return team, nil
	}
	// If we got a 404, try searching by ID in case it's a numeric ID
	if resp != nil && resp.StatusCode == 404 {
		// Try to parse as int64 and search by ID
		var id int64
		if _, scanErr := fmt.Sscanf(teamSlugOrID, "%d", &id); scanErr == nil {
			return findEnterpriseTeamByID(ctx, client, enterpriseSlug, id)
		}
	}
	return nil, err
}

// listAllEnterpriseTeamOrganizations returns all organizations assigned to an enterprise team with pagination handled.
func listAllEnterpriseTeamOrganizations(ctx context.Context, client *github.Client, enterpriseSlug, enterpriseTeam string) ([]*github.Organization, error) {
	var all []*github.Organization
	opt := &github.ListOptions{PerPage: maxPerPage}

	for {
		orgs, resp, err := client.Enterprise.ListAssignments(ctx, enterpriseSlug, enterpriseTeam, opt)
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
