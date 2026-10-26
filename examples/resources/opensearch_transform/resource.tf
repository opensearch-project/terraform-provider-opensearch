# Create a transform job that aggregates ecommerce sample data by customer
# gender and day of week into a summarized target index.
resource "opensearch_transform" "ecommerce" {
  transform_id = "ecommerce_transform"

  body = jsonencode({
    transform = {
      enabled     = true
      continuous  = true
      description = "Sample transform job"

      schedule = {
        interval = {
          period     = 1
          unit       = "Minutes"
          start_time = 1602100553
        }
      }

      source_index = "opensearch_dashboards_sample_data_ecommerce"
      target_index = "ecommerce_transform"
      page_size    = 1

      groups = [
        {
          terms = {
            source_field = "customer_gender"
            target_field = "gender"
          }
        },
        {
          terms = {
            source_field = "day_of_week"
            target_field = "day"
          }
        },
      ]

      aggregations = {
        quantity = {
          sum = {
            field = "total_quantity"
          }
        }
      }
    }
  })
}
