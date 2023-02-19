# Create a script
resource "opensearch_script" "test_script" {
  script_id = "my_script"
  lang      = "painless"
  source    = "Math.log(_score * 2) + params.my_modifier"
}
