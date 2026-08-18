# Changelog
## Unreleased
### Changed

### Added

### Fixed
* `opensearch_dashboard_object` no longer reports a diff when the only changes are bookkeeping Dashboards maintains itself: `updated_at`, and an index pattern's per-field popularity counters (`fields[].count`). Using Dashboards is enough to change both — the saved objects API restamps `updated_at` on every write, and Discover increments a counter whenever someone uses a field — which made managed objects drift continuously.

## [1.0.0] - 2023-04-15
### Added
* Release 1.0 based on phillbaker/terraform-provider-elasticsearch
