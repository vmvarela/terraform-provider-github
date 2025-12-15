---
layout: "github"
page_title: "Github: github_enterprise_cost_center_resources"
description: |-
  Manage resource assignments for a GitHub enterprise cost center.
---

# github_enterprise_cost_center_resources

This resource allows you to manage which users, organizations, and repositories are assigned to a GitHub enterprise cost center.

The `users`, `organizations`, and `repositories` arguments are authoritative: on every apply, Terraform will add and remove assignments to match exactly what is configured.

## Example Usage

```
resource "github_enterprise_cost_center" "example" {
  enterprise_slug = "example-enterprise"
  name            = "platform-cost-center"
}

resource "github_enterprise_cost_center_resources" "example" {
  enterprise_slug = "example-enterprise"
  cost_center_id  = github_enterprise_cost_center.example.id

  users         = ["octocat"]
  organizations = ["my-org"]
  repositories  = ["my-org/my-repo"]
}
```

## Argument Reference

* `enterprise_slug` - (Required) The slug of the enterprise.
* `cost_center_id` - (Required) The cost center ID.
* `users` - (Required) The usernames assigned to this cost center.
* `organizations` - (Required) The organization logins assigned to this cost center.
* `repositories` - (Required) The repositories (full name) assigned to this cost center.

## Import

GitHub Enterprise Cost Center Resources can be imported using the `enterprise_slug` and the `cost_center_id`, separated by a `/` character.

```
$ terraform import github_enterprise_cost_center_resources.example example-enterprise/cc_123456
```

