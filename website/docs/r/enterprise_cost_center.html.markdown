---
layout: "github"
page_title: "Github: github_enterprise_cost_center"
description: |-
  Create and manage a GitHub enterprise cost center.
---

# github_enterprise_cost_center

This resource allows you to create and manage a GitHub enterprise cost center.

Deleting this resource archives the cost center (GitHub calls this state `deleted`).

## Example Usage

```
resource "github_enterprise_cost_center" "example" {
  enterprise_slug = "example-enterprise"
  name            = "platform-cost-center"
}
```

## Argument Reference

* `enterprise_slug` - (Required) The slug of the enterprise.
* `name` - (Required) The name of the cost center.

## Attributes Reference

The following additional attributes are exported:

* `id` - The cost center ID.
* `state` - The state of the cost center.
* `azure_subscription` - The Azure subscription associated with the cost center.
* `resources` - A list of assigned resources.
  * `type` - The resource type.
  * `name` - The resource identifier (username, organization login, or repository full name).

## Import

GitHub Enterprise Cost Center can be imported using the `enterprise_slug` and the `cost_center_id`, separated by a `/` character.

```
$ terraform import github_enterprise_cost_center.example example-enterprise/<cost_center_id>
```

