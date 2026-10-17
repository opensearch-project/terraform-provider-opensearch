provider "opensearch" {
  url      = var.opensearch_url
  username = var.opensearch_username
  password = var.opensearch_password
  insecure = true
}

run "create_script" {

  state_key = "test-script"

  module {
    source = "./modules/opensearch_script"
  }

  variables {
    script_id     = "test-script"
    script_source = "ctx._source.counter += params.increment"
    lang          = "painless"
  }

  assert {
    condition     = opensearch_script.this.script_id == "test-script"
    error_message = "Script ID should be test-script"
  }
}

run "update_script" {
  state_key = "test-script"

  module {
    source = "./modules/opensearch_script"
  }

  variables {
    script_id     = run.create_script.script_id
    script_source = "ctx._source.counter -= params.decrement"
    lang          = "painless"
  }

  assert {
    condition     = opensearch_script.this.script_id == run.create_script.script_id
    error_message = "Script ID should match the previous run"
  }
}

run "create_full_script" {

  state_key = "test-script-full"

  module {
    source = "./modules/opensearch_script"
  }

  variables {
    script_id     = "test-script-full"
    script_source = "Math.log(ctx.score)"
    lang          = "painless"
  }

  assert {
    condition     = opensearch_script.this.script_id == "test-script-full"
    error_message = "Script ID should be test-script-full"
  }

  assert {
    condition     = opensearch_script.this.lang == "painless"
    error_message = "Script language should be painless"
  }
}


