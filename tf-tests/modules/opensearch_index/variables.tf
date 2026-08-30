variable "name" {
  description = "Name of the index"
  type        = string
}

variable "number_of_shards" {
  description = "Number of primary shards"
  type        = number
  default     = 1
}

variable "number_of_replicas" {
  description = "Number of replica shards"
  type        = number
  default     = 0
}

variable "mappings" {
  description = "Index mappings in JSON format"
  type        = string
  default     = "{ \"properties\": { } }"
}

variable "aliases" {
  description = "Index aliases"
  type        = map(any)
  default     = {}
}
