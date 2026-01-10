# Copilot Instructions for terraform-provider-github

## Project Overview

This is the official Terraform provider for GitHub, written in Go. It uses the Terraform Plugin SDK v2 and the go-github client library.

## Tech Stack

- **Language**: Go 1.24+
- **SDK**: `hashicorp/terraform-plugin-sdk/v2`
- **GitHub API Client**: `google/go-github/v81` (REST API)
- **GraphQL Client**: `shurcooL/githubv4`
- **Testing**: Terraform acceptance testing + testify

## File Naming Conventions

| Type | Pattern | Example |
|------|---------|---------|
| Resources | `resource_github_<name>.go` | `resource_github_repository.go` |
| Data Sources | `data_source_github_<name>.go` | `data_source_github_repository.go` |
| Tests | `<filename>_test.go` | `resource_github_repository_test.go` |
| Utilities | `util_<domain>.go` | `util_enterprise_teams.go` |

## Function Naming Conventions

```go
// Resources
func resourceGithub<Name>() *schema.Resource
func resourceGithub<Name>Create(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics
func resourceGithub<Name>Read(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics
func resourceGithub<Name>Update(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics
func resourceGithub<Name>Delete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics

// Data Sources
func dataSourceGithub<Name>() *schema.Resource
func dataSourceGithub<Name>Read(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics
```

## Resource Structure Template

```go
func resourceGithub<Name>() *schema.Resource {
    return &schema.Resource{
        Description:   "Creates and manages a GitHub <resource>.", // Always include Description
        CreateContext: resourceGithub<Name>Create,
        ReadContext:   resourceGithub<Name>Read,
        UpdateContext: resourceGithub<Name>Update, // Omit if resource doesn't support updates
        DeleteContext: resourceGithub<Name>Delete,
        Importer: &schema.ResourceImporter{
            StateContext: schema.ImportStatePassthroughContext,
        },
        Schema: map[string]*schema.Schema{
            // ... fields
        },
    }
}
```

## Data Source Structure Template

```go
func dataSourceGithub<Name>() *schema.Resource {
    return &schema.Resource{
        Description: "Gets information about a GitHub <resource>.", // Always include Description
        ReadContext: dataSourceGithub<Name>Read,
        Schema: map[string]*schema.Schema{
            // ... fields
        },
    }
}
```

## Schema Field Template

```go
"field_name": {
    Type:             schema.TypeString,
    Required:         true,           // or Optional: true, or Computed: true
    ForceNew:         true,           // if changing requires recreation
    Sensitive:        true,           // for secrets/tokens
    Description:      "Description of the field.", // Always include, end with period
    ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
},
```

## Available Utility Functions (github/util.go)

```go
// ID helpers
buildTwoPartID(a, b string) string           // Returns "a:b"
parseTwoPartID(id, left, right string) (string, string, error)
buildThreePartID(a, b, c string) string      // Returns "a:b:c"
parseThreePartID(id, left, middle, right string) (string, string, string, error)

// Type conversions
flattenStringList(s []*string) []string
expandStringList(d any) []*string

// Validation
toDiagFunc(oldFunc schema.SchemaValidateFunc, keyName string) schema.SchemaValidateDiagFunc
validateValueFunc(values []string) schema.SchemaValidateDiagFunc
wrapErrors(errs []error) diag.Diagnostics

// Diff suppression
caseInsensitive() schema.SchemaDiffSuppressFunc

// Context checks
checkOrganization(meta any) error            // Verify org context
```

## Testing Patterns

```go
func TestAccGithub<Resource>(t *testing.T) {
    randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
    
    config := fmt.Sprintf(`
        resource "github_<resource>" "test" {
            name = "%s%s"
        }
    `, testResourcePrefix, randomID)
    
    check := resource.ComposeTestCheckFunc(
        resource.TestCheckResourceAttr("github_<resource>.test", "name", testResourcePrefix+randomID),
    )
    
    resource.Test(t, resource.TestCase{
        PreCheck:          func() { skipUnauthenticated(t) },
        ProviderFactories: providerFactories,
        Steps: []resource.TestStep{
            {
                Config: config,
                Check:  check,
            },
        },
    })
}
```

## Make Commands

```bash
make build          # Build with linting
make test           # Unit tests
make testacc        # Acceptance tests (requires GITHUB_TOKEN)
make lint           # Run linter with auto-fix
make fmt            # Format code

# Run specific tests
TESTARGS="-run ^TestAccGithubRepository" make testacc

# With debug output
TF_LOG=DEBUG TF_ACC=1 go test -v ./... -run ^TestAccGithubRepository
```

## Environment Variables for Testing

```bash
export GITHUB_TOKEN="<your-pat>"
export GITHUB_OWNER="<org-name>"
export GITHUB_USERNAME="<username>"
export TF_ACC=1
export TF_LOG=DEBUG  # Optional: for debug output
```

---

# Common Mistakes to Avoid

## 1. Capitalization: "Github" vs "GitHub"

❌ **Wrong**:
```markdown
page_title: "Github: github_repository"
```

✅ **Correct**:
```markdown
page_title: "GitHub: github_repository"
```

Always use "GitHub" (capital H) in documentation and descriptions.

## 2. Grammar in Descriptions

❌ **Wrong**:
```go
Description: "Create and manages a GitHub repository."
```

✅ **Correct**:
```go
Description: "Creates and manages a GitHub repository."
```

Use consistent verb forms (third person singular: "Creates", "Gets", "Manages").

## 3. Error Handling: Use `diag.Errorf` Instead of Wrapping

❌ **Wrong**:
```go
return diag.FromErr(fmt.Errorf("could not find team %s", slug))
```

✅ **Correct**:
```go
return diag.Errorf("could not find team %s", slug)
```

`diag.Errorf` is cleaner and more idiomatic.

## 4. Redundant Schema Constraints

❌ **Wrong** (redundant ConflictsWith when ExactlyOneOf exists):
```go
"slug": {
    Type:          schema.TypeString,
    Optional:      true,
    ExactlyOneOf:  []string{"slug", "team_id"},
},
"team_id": {
    Type:          schema.TypeInt,
    Optional:      true,
    ConflictsWith: []string{"slug"},  // Redundant!
},
```

✅ **Correct**:
```go
"slug": {
    Type:         schema.TypeString,
    Optional:     true,
    ExactlyOneOf: []string{"slug", "team_id"},
},
"team_id": {
    Type:     schema.TypeInt,
    Optional: true,
    // ExactlyOneOf on slug handles the constraint
},
```

## 5. Missing Description Field

❌ **Wrong** (no top-level Description):
```go
func dataSourceGithubTeam() *schema.Resource {
    return &schema.Resource{
        ReadContext: dataSourceGithubTeamRead,
        Schema: map[string]*schema.Schema{...},
    }
}
```

✅ **Correct**:
```go
func dataSourceGithubTeam() *schema.Resource {
    return &schema.Resource{
        Description: "Gets information about a GitHub team.",
        ReadContext: dataSourceGithubTeamRead,
        Schema: map[string]*schema.Schema{...},
    }
}
```

Always include `Description` at the resource level.

## 6. Naming Inconsistency Between Resources and Data Sources

❌ **Wrong** (inconsistent field names):
```go
// In resource
"team_slug": {...}

// In data source
"enterprise_team": {...}  // Different name for same concept!
```

✅ **Correct**:
```go
// In resource
"team_slug": {...}

// In data source
"team_slug": {...}  // Same name
```

Use consistent field names across resources and data sources for the same concept.

## 7. Outdated go-github Import Version

❌ **Wrong**:
```go
import "github.com/google/go-github/v67/github"
```

✅ **Correct**:
```go
import "github.com/google/go-github/v81/github"
```

Always use the latest version specified in `go.mod`.

## 8. Test Variables: Global Variables vs testAccConf

❌ **Wrong** (using deprecated global variables):
```go
config := fmt.Sprintf(`...`, testEnterprise, randomID)
```

✅ **Correct** (using testAccConf):
```go
config := fmt.Sprintf(`...`, testAccConf.enterpriseSlug, randomID)
```

## 9. Linting: Function Parameters of Same Type

❌ **Wrong**:
```go
func testCheck(resourceName string, dataSourceName string) resource.TestCheckFunc
```

✅ **Correct**:
```go
func testCheck(resourceName, dataSourceName string) resource.TestCheckFunc
```

Collapse consecutive parameters of the same type.

## 10. Read-after-Create/Update Anti-pattern

❌ **Wrong**:
```go
func resourceCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
    // ... create resource
    d.SetId(id)
    return resourceRead(ctx, d, meta)  // Unnecessary extra API call
}
```

✅ **Correct**:
```go
func resourceCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
    // ... create resource
    d.SetId(id)
    // Set computed fields directly from API response
    if err := d.Set("computed_field", response.ComputedField); err != nil {
        return diag.FromErr(err)
    }
    return nil
}
```

Set computed fields directly from the Create/Update API response instead of calling Read.

## 11. ID Separator When Field May Contain Separator

❌ **Wrong** (using ":" when slug can contain ":"):
```go
// team slugs like "org:team-name" contain ":"
d.SetId(buildTwoPartID(enterprise, teamSlug))  // Results in "enterprise:org:team-name"
```

✅ **Correct** (use "/" or custom function):
```go
func buildEnterpriseTeamID(enterprise, teamSlug string) string {
    return enterprise + "/" + teamSlug
}
```

Choose ID separators that won't appear in the component values.

## 12. Missing 404 Handling in Delete

❌ **Wrong**:
```go
func resourceDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
    _, err := client.Delete(ctx, id)
    if err != nil {
        return diag.FromErr(err)  // Fails if already deleted
    }
    return nil
}
```

✅ **Correct**:
```go
func resourceDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
    _, err := client.Delete(ctx, id)
    if err != nil {
        var ghErr *github.ErrorResponse
        if errors.As(err, &ghErr) && ghErr.Response.StatusCode == http.StatusNotFound {
            return nil  // Already deleted, that's fine
        }
        return diag.FromErr(err)
    }
    return nil
}
```

## 13. Provider Registration

When adding a new resource or data source, always register it in `provider.go`:

```go
// In provider.go
ResourcesMap: map[string]*schema.Resource{
    // ... existing resources
    "github_new_resource": resourceGithubNewResource(),
},

DataSourcesMap: map[string]*schema.Resource{
    // ... existing data sources
    "github_new_data_source": dataSourceGithubNewDataSource(),
},
```

## 14. Documentation Location

- Resources: `website/docs/r/<resource_name>.html.markdown`
- Data Sources: `website/docs/d/<data_source_name>.html.markdown`

Documentation must match the schema exactly.

---

# Checklist Before Committing

- [ ] Run `make lint` and fix all issues
- [ ] Run `make fmt` to format code
- [ ] Verify all new resources/data sources are registered in `provider.go`
- [ ] Add/update documentation in `website/docs/`
- [ ] Verify "GitHub" capitalization (not "Github")
- [ ] Verify descriptions use consistent verb forms
- [ ] Add acceptance tests for new functionality
- [ ] Use `testAccConf` instead of global test variables
- [ ] Handle 404 errors gracefully in Delete functions
- [ ] Use `diag.Errorf` instead of `diag.FromErr(fmt.Errorf(...))`
- [ ] Include `Description` field in all resources and data sources
