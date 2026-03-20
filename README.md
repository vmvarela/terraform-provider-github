# Terraform Provider GitHub

<img src="https://cloud.githubusercontent.com/assets/98681/24211275/c4ebd04e-0ee8-11e7-8606-061d656a42df.png" width="72" height="">

<img src="https://raw.githubusercontent.com/hashicorp/terraform-website/d841a1e5fca574416b5ca24306f85a0f4f41b36d/content/source/assets/images/logo-terraform-main.svg" width="300px">

This is a fork of the [official Terraform GitHub provider](https://github.com/integrations/terraform-provider-github) maintained by [@vmvarela](https://github.com/vmvarela), published under `vmvarela/github` with additional enterprise features pending upstream merge.

- **Terraform Registry**: [registry.terraform.io/providers/vmvarela/github](https://registry.terraform.io/providers/vmvarela/github)
- **OpenTofu Registry**: [search.opentofu.org/provider/vmvarela/github](https://search.opentofu.org/provider/vmvarela/github)
- **Upstream**: [integrations/terraform-provider-github](https://github.com/integrations/terraform-provider-github)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 1.x or [OpenTofu](https://opentofu.org/docs/intro/install/) 1.x
- [Go](https://golang.org/doc/install) 1.26.x (to build the provider plugin)

## Usage

```hcl
terraform {
  required_providers {
    github = {
      source  = "vmvarela/github"
      version = "~> 26.0"
    }
  }
}
```

Full provider documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/vmvarela/github/latest/docs).

## Additional Features (vs upstream)

This fork includes the following features that are open PRs against upstream:

- **Enterprise Billing Usage** — `github_enterprise_billing_usage` data source
- **Enterprise Cost Centers** — `github_enterprise_cost_center` resource and data sources
- **Enterprise SCIM** — `github_enterprise_scim_*` resources for user and group provisioning
- **Enterprise Teams** — enhanced enterprise team management resources

## Contributing

Detailed documentation for contributing to this provider can be found in [CONTRIBUTING.md](CONTRIBUTING.md).

For release and publishing instructions, see [RELEASE.md](RELEASE.md).

## Roadmap

This project uses [Milestones](https://github.com/vmvarela/terraform-provider-github/milestones) to track upcoming features and fixes.

## Support

This is a community-maintained fork. For issues specific to the additional enterprise features, open an issue in this repository. For upstream provider issues, refer to the [integrations/terraform-provider-github](https://github.com/integrations/terraform-provider-github) project.
