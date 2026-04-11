package provider

import (
	"encoding/json"
	"reflect"
)

// RemoveCommonAPIMetadata removes fields commonly added by OpenSearch API
func RemoveCommonAPIMetadata(data map[string]interface{}) {
	delete(data, "last_update_time")
	delete(data, "last_updated_time")
	delete(data, "created_time")
	delete(data, "enabled_time")
	delete(data, "schema_version")
}

// RemoveIDFields removes auto-generated ID fields from nested structures
func RemoveIDFields(data []interface{}) {
	for _, item := range data {
		if m, ok := item.(map[string]interface{}); ok {
			delete(m, "id")
		}
	}
}

// NormalizeJSONBody wraps common JSON normalization logic
// It unmarshals both old and new JSON, applies the normalizer function to both,
// and compares them using reflect.DeepEqual
func NormalizeJSONBody(old, new string, normalizer func(map[string]interface{})) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	if om, ok := oo.(map[string]interface{}); ok {
		normalizer(om)
	}
	if nm, ok := no.(map[string]interface{}); ok {
		normalizer(nm)
	}

	return reflect.DeepEqual(oo, no)
}

// NormalizeQueryDefaults removes default query values added by OpenSearch
// These include: adjust_pure_negative: true, boost: 1
func NormalizeQueryDefaults(query map[string]interface{}) {
	if query == nil {
		return
	}

	// Remove default values that OpenSearch adds
	if adjustPureNegative, ok := query["adjust_pure_negative"]; ok {
		if adjustPureNegative == true {
			delete(query, "adjust_pure_negative")
		}
	}

	if boost, ok := query["boost"]; ok {
		// Remove boost if it's the default value of 1
		if boostFloat, ok := boost.(float64); ok && boostFloat == 1 {
			delete(query, "boost")
		} else if boostInt, ok := boost.(int); ok && boostInt == 1 {
			delete(query, "boost")
		}
	}

	// Recursively process nested structures
	for _, key := range []string{"bool", "must", "should", "must_not", "filter"} {
		if nested, ok := query[key]; ok {
			switch v := nested.(type) {
			case map[string]interface{}:
				NormalizeQueryDefaults(v)
			case []interface{}:
				for _, item := range v {
					if itemMap, ok := item.(map[string]interface{}); ok {
						NormalizeQueryDefaults(itemMap)
					}
				}
			}
		}
	}

	// Process range queries which also get boost defaults
	if rangeQuery, ok := query["range"].(map[string]interface{}); ok {
		for _, fieldValue := range rangeQuery {
			if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
				if boost, ok := fieldMap["boost"]; ok {
					if boostFloat, ok := boost.(float64); ok && boostFloat == 1 {
						delete(fieldMap, "boost")
					} else if boostInt, ok := boost.(int); ok && boostInt == 1 {
						delete(fieldMap, "boost")
					}
				}
			}
		}
	}
}

// NormalizeMonitorInputs normalizes the inputs section of a monitor
// It removes default values added by OpenSearch to queries
func NormalizeMonitorInputs(inputs []interface{}) {
	for _, input := range inputs {
		if inputMap, ok := input.(map[string]interface{}); ok {
			if search, ok := inputMap["search"].(map[string]interface{}); ok {
				if query, ok := search["query"].(map[string]interface{}); ok {
					NormalizeQueryDefaults(query)
					// Remove empty aggregations object added by API
					if aggregations, ok := query["aggregations"]; ok {
						if aggMap, ok := aggregations.(map[string]interface{}); ok && len(aggMap) == 0 {
							delete(query, "aggregations")
						}
					}
				}
			}
		}
	}
}
