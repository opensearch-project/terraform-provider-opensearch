resource "opensearch_script" "this" {
  script_id = var.script_id
  source    = var.script_source
  lang      = var.lang
}
