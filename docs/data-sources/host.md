---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "opensearch_host Data Source - terraform-provider-opensearch"
subcategory: ""
description: |-
  opensearch_host can be used to retrieve the host URL for the provider's current cluster.
---

# opensearch_host (Data Source)

`opensearch_host` can be used to retrieve the host URL for the provider's current cluster.

## Example Usage

```terraform
data "opensearch_host" "test" {
  active = true
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `active` (Boolean) should be set to `true`

### Read-Only

- `id` (String) The ID of this resource.
- `url` (String) the url of the active cluster
