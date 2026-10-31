provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_model_group_minimal" {
  command   = apply
  state_key = "ml_model_group_minimal"

  module {
    source = "./modules/opensearch_ml_model_group"
  }

  variables {
    name = "tf_test_ml_model_group_minimal"
  }

  assert {
    condition     = output.id != ""
    error_message = "Model group ID should not be empty"
  }

  assert {
    condition     = opensearch_ml_model_group.this.access_mode == "private"
    error_message = "Model group should default to private access"
  }
}

run "create_model_group_full" {
  command   = apply
  state_key = "ml_model_group_full"

  module {
    source = "./modules/opensearch_ml_model_group"
  }

  variables {
    name          = "tf_test_ml_model_group_full"
    description   = "Full Terraform native ML model group test"
    access_mode   = "restricted"
    backend_roles = ["ml_full_access"]
  }

  assert {
    condition     = output.id != ""
    error_message = "Model group ID should not be empty"
  }

  assert {
    condition     = opensearch_ml_model_group.this.name == "tf_test_ml_model_group_full"
    error_message = "Model group name should match the full configuration"
  }

  assert {
    condition     = opensearch_ml_model_group.this.description == "Full Terraform native ML model group test"
    error_message = "Model group description should be retained"
  }

  assert {
    condition     = opensearch_ml_model_group.this.access_mode == "restricted"
    error_message = "Model group access mode should be restricted"
  }

  assert {
    condition     = opensearch_ml_model_group.this.backend_roles == tolist(["ml_full_access"])
    error_message = "Model group backend roles should be retained"
  }
}

run "update_model_group" {
  command   = apply
  state_key = "ml_model_group_full"

  module {
    source = "./modules/opensearch_ml_model_group"
  }

  variables {
    name          = "tf_test_ml_model_group_full_updated"
    description   = "Updated Terraform native ML model group test"
    access_mode   = "restricted"
    backend_roles = ["ml_full_access", "ml_readonly_access"]
  }

  assert {
    condition     = output.id == run.create_model_group_full.id
    error_message = "Updating the model group should preserve its ID"
  }

  assert {
    condition     = opensearch_ml_model_group.this.name == "tf_test_ml_model_group_full_updated"
    error_message = "Model group name should be updated"
  }

  assert {
    condition     = opensearch_ml_model_group.this.description == "Updated Terraform native ML model group test"
    error_message = "Model group description should be updated"
  }

  assert {
    condition     = opensearch_ml_model_group.this.backend_roles == tolist(["ml_full_access", "ml_readonly_access"])
    error_message = "Model group backend roles should be updated"
  }
}

run "create_model_group_public" {
  command   = apply
  state_key = "ml_model_group_public"

  module {
    source = "./modules/opensearch_ml_model_group"
  }

  variables {
    name        = "tf_test_ml_model_group_public"
    description = "Public Terraform native ML model group test"
    access_mode = "public"
  }

  assert {
    condition     = output.id != ""
    error_message = "Public model group ID should not be empty"
  }

  assert {
    condition     = opensearch_ml_model_group.this.access_mode == "public"
    error_message = "Model group access mode should be public"
  }
}
