provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_repository" {
  state_key = "create_repository"

  module {
    source = "./modules/opensearch_snapshot_repository"
  }

  variables {
    name = "test-repo"
    type = "fs"
    settings = {
      location = "/tmp/snapshots"
    }
  }

  assert {
    condition     = opensearch_snapshot_repository.this.name == "test-repo"
    error_message = "Repository name should be test-repo"
  }

  assert {
    condition     = opensearch_snapshot_repository.this.type == "fs"
    error_message = "Repository type should be fs"
  }
}

run "create_repository_full" {
  state_key = "create_repository_full"

  module {
    source = "./modules/opensearch_snapshot_repository"
  }

  variables {
    name = "test-repo-full"
    type = "fs"
    settings = {
      location                   = "/tmp/snapshots"
      compress                   = "true"
      chunk_size                 = "100mb"
      max_snapshot_bytes_per_sec = "100mb"
      max_restore_bytes_per_sec  = "100mb"
      readonly                   = "false"
    }
  }

  assert {
    condition     = opensearch_snapshot_repository.this.name == "test-repo-full"
    error_message = "Repository name should be test-repo-full"
  }

  assert {
    condition     = opensearch_snapshot_repository.this.type == "fs"
    error_message = "Repository type should be fs"
  }
}

run "update_repository" {
  state_key = "create_repository"

  module {
    source = "./modules/opensearch_snapshot_repository"
  }

  variables {
    name = run.create_repository.name
    type = "fs"
    settings = {
      location = "/tmp/snapshots"
      compress = "true"
    }
  }

  assert {
    condition     = opensearch_snapshot_repository.this.name == run.create_repository.name
    error_message = "Repository name should match the previous run"
  }
}
