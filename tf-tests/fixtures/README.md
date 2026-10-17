# Sample Index Lifecycle Management (ISM) Policies

This directory contains sample policies and configurations for testing.

## ISM Policy Examples

### Hot-Cold-Delete Lifecycle
```json
{
  "policy": {
    "description": "Hot to Cold to Delete lifecycle",
    "default_state": "hot",
    "states": [
      {
        "name": "hot",
        "actions": [],
        "transitions": [
          {
            "state_name": "cold",
            "conditions": {
              "min_index_age": "30d"
            }
          }
        ]
      },
      {
        "name": "cold",
        "actions": [
          {
            "replica_count": {
              "number_of_replicas": 0
            }
          }
        ],
        "transitions": [
          {
            "state_name": "delete",
            "conditions": {
              "min_index_age": "90d"
            }
          }
        ]
      },
      {
        "name": "delete",
        "actions": [
          {
            "delete": {}
          }
        ],
        "transitions": []
      }
    ]
  }
}
```

### Log Rollover Policy
```json
{
  "policy": {
    "description": "Log rollover policy",
    "default_state": "hot",
    "states": [
      {
        "name": "hot",
        "actions": [
          {
            "rollover": {
              "min_size": "50gb",
              "min_index_age": "1d"
            }
          }
        ],
        "transitions": []
      }
    ]
  }
}
```

## Component Template Examples

### Common Mappings
```json
{
  "template": {
    "mappings": {
      "properties": {
        "@timestamp": {
          "type": "date"
        },
        "host_name": {
          "type": "keyword"
        },
        "log_level": {
          "type": "keyword"
        },
        "message": {
          "type": "text"
        }
      }
    }
  }
}
```

## Ingest Pipeline Examples

### Log Enrichment
```json
{
  "description": "Enrich log entries",
  "processors": [
    {
      "set": {
        "field": "ingested_at",
        "value": "{{_ingest.timestamp}}"
      }
    },
    {
      "script": {
        "description": "Extract log level from message",
        "source": "ctx.log_level = ctx.message =~ /^\\[ERROR\\]/ ? 'ERROR' : ctx.message =~ /^\\[WARN\\]/ ? 'WARN' : 'INFO'"
      }
    }
  ]
}
```

## Role Templates

### Read-Only Role
```json
{
  "cluster_permissions": ["cluster:monitor/health"],
  "index_permissions": [
    {
      "index_patterns": ["logs-*"],
      "allowed_actions": ["read", "view_index_metadata"]
    }
  ]
}
```

### Log Ingestion Role
```json
{
  "cluster_permissions": [
    "cluster:monitor/health",
    "cluster:monitor/state"
  ],
  "index_permissions": [
    {
      "index_patterns": ["logs-*"],
      "allowed_actions": ["indices:admin/create", "indices:data/write/*"]
    }
  ]
}
```
