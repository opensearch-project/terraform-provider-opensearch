provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_user_minimal" {
  state_key = "test-user-minimal"

  module {
    source = "./modules/opensearch_user"
  }

  variables {
    username = "test-user-minimal"
    password = "TestPassword123!"
  }

  assert {
    condition     = opensearch_user.this.username == "test-user-minimal"
    error_message = "Username should be test-user-minimal"
  }

  assert {
    condition     = opensearch_user.this.id != ""
    error_message = "User ID should not be empty"
  }
}

run "create_user_full" {
  state_key = "test-user-full"

  module {
    source = "./modules/opensearch_user"
  }

  variables {
    username      = "test-user-full"
    password      = "TestPassword123!"
    description   = "Full config test user"
    attributes    = { department = "engineering", team = "platform" }
    backend_roles = ["kibanauser", "readall"]
  }

  assert {
    condition     = opensearch_user.this.username == "test-user-full"
    error_message = "Username should be test-user-full"
  }

  assert {
    condition     = opensearch_user.this.description == "Full config test user"
    error_message = "Description should match the provided value"
  }

  assert {
    condition     = opensearch_user.this.attributes["department"] == "engineering"
    error_message = "Attribute department should be engineering"
  }

  assert {
    condition     = opensearch_user.this.attributes["team"] == "platform"
    error_message = "Attribute team should be platform"
  }

  assert {
    condition     = opensearch_user.this.backend_roles == toset(["kibanauser", "readall"])
    error_message = "Backend roles should contain kibanauser and readall"
  }

  assert {
    condition     = opensearch_user.this.id != ""
    error_message = "User ID should not be empty"
  }
}

run "update_user" {
  state_key = "test-user-minimal"

  module {
    source = "./modules/opensearch_user"
  }

  variables {
    username      = run.create_user_minimal.username
    password      = "UpdatedPassword456!"
    description   = "Updated test user"
    attributes    = { department = "security" }
    backend_roles = ["log_viewer"]
  }

  assert {
    condition     = opensearch_user.this.username == "test-user-minimal"
    error_message = "Username should remain test-user-minimal"
  }

  assert {
    condition     = opensearch_user.this.description == "Updated test user"
    error_message = "Description should be updated"
  }

  assert {
    condition     = opensearch_user.this.attributes["department"] == "security"
    error_message = "Attribute department should be updated to security"
  }

  assert {
    condition     = opensearch_user.this.backend_roles == toset(["log_viewer"])
    error_message = "Backend roles should be updated to log_viewer"
  }
}
