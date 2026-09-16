package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func enterpriseTeamTestOwner(t *testing.T, handler http.HandlerFunc) *Owner {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &Owner{v3client: mustCreateTestGitHubClient(t, server.URL+"/"), maxPerPage: 100}
}

func TestEnterpriseTeamReadRejectsReusedSlug(t *testing.T) {
	owner := enterpriseTeamTestOwner(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/enterprises/ent/teams/ent:old":
			fmt.Fprint(w, `{"id":99,"slug":"ent:old","name":"Other team"}`)
		case "/enterprises/ent/teams":
			fmt.Fprint(w, `[{"id":42,"slug":"ent:renamed","name":"Renamed"}]`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			http.NotFound(w, r)
		}
	})
	d := schema.TestResourceDataRaw(t, resourceGithubEnterpriseTeam().Schema, map[string]any{
		"enterprise_slug": "ent", "name": "Old", "slug": "ent:old", "team_id": 42,
	})
	d.SetId("42")
	if diags := resourceGithubEnterpriseTeamRead(t.Context(), d, owner); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Id() != "42" || d.Get("team_id") != 42 || d.Get("slug") != "ent:renamed" {
		t.Fatalf("adopted wrong team: id=%s team_id=%v slug=%v", d.Id(), d.Get("team_id"), d.Get("slug"))
	}
}

func TestEnterpriseTeamOrganizationsReadFollowsNumericID(t *testing.T) {
	owner := enterpriseTeamTestOwner(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/enterprises/ent/teams/ent:old":
			http.NotFound(w, r)
		case "/enterprises/ent/teams":
			fmt.Fprint(w, `[{"id":42,"slug":"ent:renamed","name":"Renamed"}]`)
		case "/enterprises/ent/teams/ent:renamed/organizations":
			fmt.Fprint(w, `[{"login":"org-a"}]`)
		default:
			http.NotFound(w, r)
		}
	})
	d := schema.TestResourceDataRaw(t, resourceGithubEnterpriseTeamOrganizations().Schema, map[string]any{
		"enterprise_slug": "ent", "team_id": 42, "organization_slugs": []any{"org-a"},
	})
	d.SetId("ent/ent:old")
	if diags := resourceGithubEnterpriseTeamOrganizationsRead(t.Context(), d, owner); diags.HasError() {
		t.Fatal(diags)
	}
	if d.Id() != "ent/ent:renamed" {
		t.Fatalf("lost assignment identity after rename: %q", d.Id())
	}
}

func TestEnterpriseTeamUpdateClearsDescription(t *testing.T) {
	resource := resourceGithubEnterpriseTeam()
	old := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{
		"enterprise_slug": "ent", "name": "Team", "slug": "ent:team", "team_id": 42,
		"description": "Remove this", "group_id": "keep-group",
	})
	old.SetId("42")
	config := terraform.NewResourceConfigRaw(map[string]any{
		"enterprise_slug": "ent", "name": "Team", "group_id": "keep-group",
	})
	diff, err := resource.Diff(t.Context(), old.State(), config, nil)
	if err != nil {
		t.Fatal(err)
	}
	d := resource.Data(old.State())
	// Apply through the SDK so HasChange sees the real old/new values.
	owner := enterpriseTeamTestOwner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"id":42,"slug":"ent:team","name":"Team"}`)
			return
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if string(body["description"]) != `""` {
			t.Errorf("description not explicitly cleared: %s", body["description"])
		}
		if _, present := body["group_id"]; present {
			t.Error("unchanged IdP mapping should not be resubmitted")
		}
		fmt.Fprint(w, `{"id":42,"slug":"ent:team","name":"Team","description":"","group_id":"keep-group"}`)
	})
	_, diags := resource.Apply(t.Context(), d.State(), diff, owner)
	if diags.HasError() {
		t.Fatal(diags)
	}
}

func TestEnterpriseTeamInputValidation(t *testing.T) {
	for name, resource := range map[string]*schema.Resource{
		"membership":    resourceGithubEnterpriseTeamMembership(),
		"organizations": resourceGithubEnterpriseTeamOrganizations(),
	} {
		t.Run(name, func(t *testing.T) {
			for _, id := range []int{0, -1} {
				config := map[string]any{"enterprise_slug": "ent", "team_id": id}
				if name == "membership" {
					config["username"] = "user"
				} else {
					config["organization_slugs"] = []any{"org"}
				}
				if diags := resource.Validate(terraform.NewResourceConfigRaw(config)); !diags.HasError() {
					t.Errorf("accepted invalid team_id %d", id)
				}
			}
		})
	}
	for _, slug := range []string{"", "   "} {
		config := terraform.NewResourceConfigRaw(map[string]any{
			"enterprise_slug": "ent", "team_id": 42, "organization_slugs": []any{slug},
		})
		if diags := resourceGithubEnterpriseTeamOrganizations().Validate(config); !diags.HasError() {
			t.Errorf("accepted invalid organization slug %q", slug)
		}
	}
}

func TestEnterpriseTeamGroupRemovalPlansReplacement(t *testing.T) {
	resource := resourceGithubEnterpriseTeam()
	for _, tc := range []struct {
		name, old, next string
		replace         bool
	}{
		{"remove", "group-a", "", true},
		{"change", "group-a", "group-b", false},
		{"add", "", "group-a", false},
		{"unchanged", "group-a", "group-a", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{
				"enterprise_slug": "ent", "name": "Team", "group_id": tc.old,
				"slug": "ent:team", "team_id": 42,
			})
			old.SetId("42")
			config := map[string]any{"enterprise_slug": "ent", "name": "Team"}
			if tc.next != "" {
				config["group_id"] = tc.next
			}
			diff, err := resource.Diff(t.Context(), old.State(), terraform.NewResourceConfigRaw(config), nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := diff != nil && diff.RequiresNew(); got != tc.replace {
				t.Fatalf("replacement = %v, want %v", got, tc.replace)
			}
		})
	}
}

func TestEnterpriseTeamDependentsRetainIdentity(t *testing.T) {
	for name, resource := range map[string]*schema.Resource{
		"membership":    resourceGithubEnterpriseTeamMembership(),
		"organizations": resourceGithubEnterpriseTeamOrganizations(),
	} {
		for _, selector := range []string{"team_slug", "team_id"} {
			for _, operation := range []string{"read", "delete"} {
				t.Run(name+"/"+selector+"/"+operation, func(t *testing.T) {
					mutations := 0
					owner := enterpriseTeamTestOwner(t, func(w http.ResponseWriter, r *http.Request) {
						switch r.URL.Path {
						case "/enterprises/ent/teams/ent:old":
							fmt.Fprint(w, `{"id":99,"slug":"ent:old","name":"Unrelated"}`)
						case "/enterprises/ent/teams":
							fmt.Fprint(w, `[{"id":42,"slug":"ent:renamed","name":"Renamed"}]`)
						case "/enterprises/ent/teams/ent:renamed/memberships/user":
							if r.Method == http.MethodDelete {
								mutations++
								w.WriteHeader(http.StatusNoContent)
							} else {
								fmt.Fprint(w, `{"id":7,"login":"user"}`)
							}
						case "/enterprises/ent/teams/ent:renamed/organizations":
							fmt.Fprint(w, `[{"login":"org-a"}]`)
						case "/enterprises/ent/teams/ent:renamed/organizations/remove":
							mutations++
							fmt.Fprint(w, `[]`)
						default:
							t.Errorf("request targets wrong team: %s %s", r.Method, r.URL)
							http.NotFound(w, r)
						}
					})
					config := map[string]any{"enterprise_slug": "ent", "resolved_team_id": 42}
					if selector == "team_id" {
						config[selector] = 42
					} else {
						config[selector] = "ent:old"
					}
					id, expected := "ent/ent:old", "ent/ent:renamed"
					if name == "membership" {
						config["username"] = "user"
						id += "/user"
						expected += "/user"
					} else {
						config["organization_slugs"] = []any{"org-a"}
					}
					d := schema.TestResourceDataRaw(t, resource.Schema, config)
					d.SetId(id)
					if operation == "read" {
						if diags := resource.ReadContext(t.Context(), d, owner); diags.HasError() {
							t.Fatal(diags)
						}
						if d.Id() != expected || d.Get("resolved_team_id") != 42 {
							t.Fatalf("lost identity: %s, %v", d.Id(), d.Get("resolved_team_id"))
						}
						if selector == "team_slug" && d.Get("team_slug") != "ent:renamed" {
							t.Fatal("slug not refreshed")
						}
						if selector == "team_id" && d.Get("team_slug") != "" {
							t.Fatal("populated conflicting selector")
						}
					} else {
						if diags := resource.DeleteContext(t.Context(), d, owner); diags.HasError() {
							t.Fatal(diags)
						}
						if mutations != 1 {
							t.Fatalf("expected one deletion, got %d", mutations)
						}
					}
				})
			}
		}
	}
}

func TestEnterpriseTeamDependentsImportAndLegacyRefresh(t *testing.T) {
	for name, resource := range map[string]*schema.Resource{
		"membership":    resourceGithubEnterpriseTeamMembership(),
		"organizations": resourceGithubEnterpriseTeamOrganizations(),
	} {
		for _, legacy := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/legacy=%v", name, legacy), func(t *testing.T) {
				owner := enterpriseTeamTestOwner(t, func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/enterprises/ent/teams/ent:team":
						fmt.Fprint(w, `{"id":42,"slug":"ent:team","name":"Team"}`)
					case "/enterprises/ent/teams/ent:team/memberships/user":
						fmt.Fprint(w, `{"id":7,"login":"user"}`)
					case "/enterprises/ent/teams/ent:team/organizations":
						fmt.Fprint(w, `[{"login":"org-a"}]`)
					default:
						t.Errorf("unexpected request: %s", r.URL)
						http.NotFound(w, r)
					}
				})
				config := map[string]any{}
				if legacy {
					config["enterprise_slug"] = "ent"
					config["team_slug"] = "ent:team"
				}
				d := schema.TestResourceDataRaw(t, resource.Schema, config)
				id := "ent/ent:team"
				if name == "membership" {
					id += "/user"
				}
				d.SetId(id)
				if !legacy {
					if _, err := resource.Importer.StateContext(t.Context(), d, owner); err != nil {
						t.Fatal(err)
					}
				}
				if diags := resource.ReadContext(t.Context(), d, owner); diags.HasError() {
					t.Fatal(diags)
				}
				if d.Id() != id || d.Get("resolved_team_id") != 42 || d.Get("team_slug") != "ent:team" || d.Get("enterprise_slug") != "ent" {
					t.Fatalf("incomplete imported state: %#v", d.State().Attributes)
				}
			})
		}
	}
}

func TestEnterpriseTeamOrganizationsUpdateDeltaAfterRename(t *testing.T) {
	resource := resourceGithubEnterpriseTeamOrganizations()
	old := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{
		"enterprise_slug": "ent", "team_id": 42, "resolved_team_id": 42, "organization_slugs": []any{"org-a", "org-b"},
	})
	old.SetId("ent/ent:old")
	config := terraform.NewResourceConfigRaw(map[string]any{
		"enterprise_slug": "ent", "team_id": 42, "organization_slugs": []any{"org-b", "org-c"},
	})
	diff, err := resource.Diff(t.Context(), old.State(), config, nil)
	if err != nil {
		t.Fatal(err)
	}
	mutations := 0
	owner := enterpriseTeamTestOwner(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/enterprises/ent/teams/ent:old":
			fmt.Fprint(w, `{"id":99,"slug":"ent:old"}`)
		case "/enterprises/ent/teams":
			fmt.Fprint(w, `[{"id":42,"slug":"ent:renamed"}]`)
		case "/enterprises/ent/teams/ent:renamed/organizations/add", "/enterprises/ent/teams/ent:renamed/organizations/remove":
			var body struct {
				Slugs []string `json:"organization_slugs"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			want := "org-c"
			if r.URL.Path == "/enterprises/ent/teams/ent:renamed/organizations/remove" {
				want = "org-a"
			}
			if len(body.Slugs) != 1 || body.Slugs[0] != want {
				t.Errorf("incorrect delta: %v, want %s", body.Slugs, want)
			}
			mutations++
			fmt.Fprint(w, `[]`)
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	})
	state, diags := resource.Apply(t.Context(), old.State(), diff, owner)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if mutations != 2 {
		t.Fatalf("got %d mutations, want 2", mutations)
	}
	if state.ID != "ent/ent:renamed" {
		t.Fatalf("update kept stale ID: %s", state.ID)
	}
}

func TestEnterpriseTeamMissingAndAPIError(t *testing.T) {
	for name, resource := range map[string]*schema.Resource{
		"team":          resourceGithubEnterpriseTeam(),
		"membership":    resourceGithubEnterpriseTeamMembership(),
		"organizations": resourceGithubEnterpriseTeamOrganizations(),
	} {
		for _, code := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
			t.Run(fmt.Sprintf("%s/%d", name, code), func(t *testing.T) {
				writes := 0
				owner := enterpriseTeamTestOwner(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodGet {
						writes++
						t.Errorf("unexpected mutation %s", r.URL)
					}
					if code == http.StatusNotFound && r.URL.Path == "/enterprises/ent/teams" {
						fmt.Fprint(w, `[]`)
						return
					}
					w.WriteHeader(code)
					fmt.Fprint(w, `{"message":"error"}`)
				})
				config := map[string]any{"enterprise_slug": "ent", "team_id": 42}
				id := "42"
				if name == "team" {
					config["slug"] = "ent:old"
					config["name"] = "Old"
				} else {
					config["resolved_team_id"] = 42
					id = "ent/ent:old"
					if name == "membership" {
						config["username"] = "user"
						id += "/user"
					} else {
						config["organization_slugs"] = []any{"org-a"}
					}
				}
				d := schema.TestResourceDataRaw(t, resource.Schema, config)
				d.SetId(id)
				diags := resource.ReadContext(t.Context(), d, owner)
				if code == http.StatusNotFound {
					if diags.HasError() || d.Id() != "" {
						t.Fatalf("missing resource retained: %s %v", d.Id(), diags)
					}
				} else if !diags.HasError() || d.Id() != id {
					t.Fatalf("API error lost or state cleared: %s %v", d.Id(), diags)
				}
				d.SetId(id)
				diags = resource.DeleteContext(t.Context(), d, owner)
				if diags.HasError() != (code != http.StatusNotFound) {
					t.Fatalf("wrong delete error: %v", diags)
				}
				if writes != 0 {
					t.Fatal("mutated unverified team")
				}
			})
		}
	}
}

func TestEnterpriseTeamMutationsRejectReusedSlug(t *testing.T) {
	for _, operation := range []string{"update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			mutations := 0
			owner := enterpriseTeamTestOwner(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/enterprises/ent/teams/ent:old":
					if r.Method != http.MethodGet {
						t.Errorf("mutated unrelated team: %s", r.Method)
					}
					fmt.Fprint(w, `{"id":99,"slug":"ent:old"}`)
				case "/enterprises/ent/teams":
					if r.URL.Query().Get("page") == "2" {
						fmt.Fprint(w, `[{"id":42,"slug":"ent:renamed"}]`)
						return
					}
					w.Header().Set("Link", fmt.Sprintf("<http://%s/enterprises/ent/teams?page=2>; rel=\"next\"", r.Host))
					fmt.Fprint(w, `[{"id":99,"slug":"ent:old"}]`)
				case "/enterprises/ent/teams/ent:renamed":
					mutations++
					if operation == "delete" {
						w.WriteHeader(http.StatusNoContent)
					} else {
						fmt.Fprint(w, `{"id":42,"slug":"ent:renamed"}`)
					}
				default:
					t.Errorf("unexpected request: %s", r.URL)
					http.NotFound(w, r)
				}
			})
			r := resourceGithubEnterpriseTeam()
			d := schema.TestResourceDataRaw(t, r.Schema, map[string]any{
				"enterprise_slug": "ent", "name": "Renamed", "slug": "ent:old", "team_id": 42,
			})
			d.SetId("42")
			if operation == "delete" {
				if diags := r.DeleteContext(t.Context(), d, owner); diags.HasError() {
					t.Fatal(diags)
				}
			} else {
				if diags := r.UpdateContext(t.Context(), d, owner); diags.HasError() {
					t.Fatal(diags)
				}
			}
			if mutations != 1 {
				t.Fatalf("got %d writes to intended team, want 1", mutations)
			}
		})
	}
}

func TestEnterpriseTeamDependentsCreateStoresIdentity(t *testing.T) {
	for name, resource := range map[string]*schema.Resource{
		"membership":    resourceGithubEnterpriseTeamMembership(),
		"organizations": resourceGithubEnterpriseTeamOrganizations(),
	} {
		t.Run(name, func(t *testing.T) {
			owner := enterpriseTeamTestOwner(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/enterprises/ent/teams/ent:team":
					fmt.Fprint(w, `{"id":42,"slug":"ent:team"}`)
				case "/enterprises/ent/teams/ent:team/memberships/user":
					fmt.Fprint(w, `{"id":7,"login":"user"}`)
				case "/enterprises/ent/teams/ent:team/organizations", "/enterprises/ent/teams/ent:team/organizations/add":
					fmt.Fprint(w, `[]`)
				default:
					t.Errorf("unexpected request: %s", r.URL)
					http.NotFound(w, r)
				}
			})
			config := map[string]any{"enterprise_slug": "ent", "team_slug": "ent:team"}
			if name == "membership" {
				config["username"] = "user"
			} else {
				config["organization_slugs"] = []any{"org-a"}
			}
			d := schema.TestResourceDataRaw(t, resource.Schema, config)
			if diags := resource.CreateContext(t.Context(), d, owner); diags.HasError() {
				t.Fatal(diags)
			}
			if d.Id() == "" || d.Get("resolved_team_id") != 42 {
				t.Fatalf("missing persisted identity: %#v", d.State().Attributes)
			}
		})
	}
}

func TestEnterpriseTeamOrganizationsCaseInsensitive(t *testing.T) {
	for _, tc := range []struct {
		name          string
		before, after []any
		writes        int
	}{
		{"case only", []any{"org-a"}, []any{"Org-A"}, 0},
		{"duplicates", []any{"org-a"}, []any{"org-a", "Org-A"}, 0},
		{"legacy casing", []any{"Org-A"}, []any{"org-a"}, 0},
		{"real delta", []any{"Org-A", "org-b"}, []any{"org-a", "Org-C"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := resourceGithubEnterpriseTeamOrganizations()
			legacy := resourceGithubEnterpriseTeamOrganizations()
			legacy.Schema["organization_slugs"].Set = schema.HashString
			element, _ := legacy.Schema["organization_slugs"].Elem.(*schema.Schema)
			element.StateFunc = nil
			old := schema.TestResourceDataRaw(t, legacy.Schema, map[string]any{
				"enterprise_slug": "ent", "team_id": 42, "resolved_team_id": 42, "organization_slugs": tc.before,
			})
			old.SetId("ent/ent:team")
			config := terraform.NewResourceConfigRaw(map[string]any{
				"enterprise_slug": "ent", "team_id": 42, "organization_slugs": tc.after,
			})
			diff, err := r.Diff(t.Context(), old.State(), config, nil)
			if err != nil {
				t.Fatal(err)
			}
			writes := 0
			owner := enterpriseTeamTestOwner(t, func(w http.ResponseWriter, req *http.Request) {
				switch req.URL.Path {
				case "/enterprises/ent/teams/ent:team":
					fmt.Fprint(w, `{"id":42,"slug":"ent:team"}`)
				case "/enterprises/ent/teams/ent:team/organizations":
					if tc.writes == 2 {
						fmt.Fprint(w, `[{"login":"Org-A"},{"login":"Org-C"}]`)
					} else {
						fmt.Fprint(w, `[{"login":"Org-A"}]`)
					}
				case "/enterprises/ent/teams/ent:team/organizations/add", "/enterprises/ent/teams/ent:team/organizations/remove":
					writes++
					var body struct {
						Slugs []string `json:"organization_slugs"`
					}
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					want := "org-c"
					if req.URL.Path == "/enterprises/ent/teams/ent:team/organizations/remove" {
						want = "org-b"
					}
					if tc.writes == 0 || len(body.Slugs) != 1 || body.Slugs[0] != want {
						t.Errorf("unexpected mutation: %s %v", req.URL.Path, body.Slugs)
					}
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request %s", req.URL)
					http.NotFound(w, req)
				}
			})
			state := old.State()
			if diff != nil && !diff.Empty() {
				var diags diag.Diagnostics
				state, diags = r.Apply(t.Context(), state, diff, owner)
				if diags.HasError() {
					t.Fatal(diags)
				}
			}
			if writes != tc.writes {
				t.Fatalf("writes = %d, want %d", writes, tc.writes)
			}
			refreshed := r.Data(state)
			if diags := r.ReadContext(t.Context(), refreshed, owner); diags.HasError() {
				t.Fatal(diags)
			}
			diff, err = r.Diff(t.Context(), refreshed.State(), config, owner)
			if err != nil {
				t.Fatal(err)
			}
			if diff != nil && !diff.Empty() {
				t.Fatalf("nonempty plan after refresh: %#v", diff.Attributes)
			}
		})
	}
}
