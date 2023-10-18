package provider

import (
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func diffSuppressIndexTemplate(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	if om, ok := oo.(map[string]interface{}); ok {
		normalizeIndexTemplate(om)
	}

	if nm, ok := no.(map[string]interface{}); ok {
		normalizeIndexTemplate(nm)
	}

	return reflect.DeepEqual(oo, no)
}

/*
diffSuppressComposableIndexTemplate compares an index_template (ES >= 7.8) Index template definition
For legacy index templates (ES < 7.8) or /_template endpoint on ES >= 7.8 see diffSuppressIndexTemplate.
*/
func diffSuppressComposableIndexTemplate(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	if om, ok := oo.(map[string]interface{}); ok {
		normalizeComposableIndexTemplate(om)
	}

	if nm, ok := no.(map[string]interface{}); ok {
		normalizeComposableIndexTemplate(nm)
	}

	return reflect.DeepEqual(oo, no)
}

func diffSuppressComponentTemplate(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	if om, ok := oo.(map[string]interface{}); ok {
		normalizeComponentTemplate(om)
	}

	if nm, ok := no.(map[string]interface{}); ok {
		normalizeComponentTemplate(nm)
	}

	return reflect.DeepEqual(oo, no)
}

func diffSuppressMonitor(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	if om, ok := oo.(map[string]interface{}); ok {
		normalizeMonitor(om)
	}

	if nm, ok := no.(map[string]interface{}); ok {
		normalizeMonitor(nm)
	}

	return reflect.DeepEqual(oo, no)
}

func diffSuppressChannelConfiguration(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	if om, ok := oo.(map[string]interface{}); ok {
		normalizeChannelConfiguration(om)
	}

	if nm, ok := no.(map[string]interface{}); ok {
		normalizeChannelConfiguration(nm)
	}

	return reflect.DeepEqual(oo, no)
}

func diffSuppressIngestPipeline(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	return reflect.DeepEqual(oo, no)
}

func diffSuppressPolicy(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	if om, ok := oo.(map[string]interface{}); ok {
		normalizePolicy(om)
	}

	if nm, ok := no.(map[string]interface{}); ok {
		normalizePolicy(nm)
	}

	return reflect.DeepEqual(oo, no)
}

func diffSuppressAnomalyDetection(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	if om, ok := oo.(map[string]interface{}); ok {
		normalizeAnomalyDetection(om)
	}

	if nm, ok := no.(map[string]interface{}); ok {
		normalizeAnomalyDetection(nm)
	}

	return reflect.DeepEqual(oo, no)
}
