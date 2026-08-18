# Changelog
## Unreleased
### Changed

### Added

### Fixed
* `opensearch_dashboard_object` no longer reports a diff when an index pattern's per-field popularity counters (`fields[].count`) are the only thing that changed. Dashboards increments them whenever someone uses a field in Discover, which made every managed index pattern drift continuously.

## [1.0.0] - 2023-04-15
### Added
* Release 1.0 based on phillbaker/terraform-provider-elasticsearch
