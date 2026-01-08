package github

import (
	"context"

	"github.com/google/go-github/v81/github"
)

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
