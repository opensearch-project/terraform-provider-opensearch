package provider

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-homedir"
)

func normalizeChannelConfiguration(tpl map[string]interface{}) {
	delete(tpl, "last_updated_time_ms")
	delete(tpl, "created_time_ms")

	// Remove API-generated config_id
	delete(tpl, "config_id")

	// Normalize config section
	if config, ok := tpl["config"].(map[string]interface{}); ok {
		// Remove default fields added by API to webhook config
		if webhook, ok := config["webhook"].(map[string]interface{}); ok {
			// Remove empty header_params
			if headerParams, ok := webhook["header_params"]; ok {
				if headerMap, ok := headerParams.(map[string]interface{}); ok && len(headerMap) == 0 {
					delete(webhook, "header_params")
				}
			}
			// Remove default method
			if method, ok := webhook["method"]; ok && method == "POST" {
				delete(webhook, "method")
			}
		}
	}
}

func normalizeMonitor(tpl map[string]interface{}) {
	// Normalize inputs to remove default query values added by OpenSearch
	if inputs, ok := tpl["inputs"].([]interface{}); ok {
		NormalizeMonitorInputs(inputs)
	}

	if triggers, ok := tpl["triggers"].([]interface{}); ok {
		normalizeMonitorTriggers(triggers)
	}

	// Use shared utility for common API metadata
	RemoveCommonAPIMetadata(tpl)
	delete(tpl, "id")
	delete(tpl, "user")

	// Remove alerting-specific fields added by OpenSearch
	delete(tpl, "data_sources")
	delete(tpl, "delete_query_index_in_every_run")
	delete(tpl, "owner")
	delete(tpl, "should_create_single_alert_for_findings")
}

func normalizeMonitorTriggers(triggers []interface{}) {
	for _, t := range triggers {
		if trigger, ok := t.(map[string]interface{}); ok {
			delete(trigger, "id")

			if actions, ok := trigger["actions"].([]interface{}); ok {
				normalizeMonitorTriggerActions(actions)
			}
		}
	}
}

func normalizeMonitorTriggerActions(actions []interface{}) {
	for _, a := range actions {
		action := a.(map[string]interface{})
		delete(action, "id")
	}
}

func normalizePolicy(tpl map[string]interface{}) {
	delete(tpl, "last_updated_time")
	delete(tpl, "policy_id")
	delete(tpl, "schema_version")
	delete(tpl, "user")
	if ism_template, ok := tpl["ism_template"]; ok {
		if ism_template == nil {
			delete(tpl, "ism_template")
		}

		switch templates := ism_template.(type) {
		case map[string]interface{}:
			delete(templates, "last_updated_time")
		case []interface{}:
			for _, t := range templates {
				if template, ok := t.(map[string]interface{}); ok {
					delete(template, "last_updated_time")
				}
			}
		default:
			log.Printf("[INFO] normalizePolicy unknown type: %T", ism_template)
		}
	}
	// Normalize states to remove API-added fields
	if states, ok := tpl["states"].([]interface{}); ok {
		for _, s := range states {
			if state, ok := s.(map[string]interface{}); ok {
				normalizeISMState(state)
			}
		}
	}
	// ignore if set to null in response (ie not specified)
	if error_notification, ok := tpl["error_notification"]; ok {
		if error_notification == nil {
			delete(tpl, "error_notification")
		}
	}
}

func normalizeIndexTemplate(tpl map[string]interface{}) {
	delete(tpl, "version")
	// Remove order if it's the default (0)
	if order, ok := tpl["order"]; ok {
		if orderNum, ok := order.(float64); ok && orderNum == 0 {
			delete(tpl, "order")
		}
	}
	// Handle settings at top level (legacy format)
	if settings, ok := tpl["settings"]; ok {
		if settingsMap, ok := settings.(map[string]interface{}); ok {
			if len(settingsMap) == 0 {
				delete(tpl, "settings")
			} else {
				tpl["settings"] = normalizedIndexSettings(settingsMap)
			}
		} else {
			delete(tpl, "settings")
		}
	}
	// Remove empty mappings
	if mappings, ok := tpl["mappings"]; ok {
		if mappingsMap, ok := mappings.(map[string]interface{}); ok && len(mappingsMap) == 0 {
			delete(tpl, "mappings")
		}
	}
	// Remove empty aliases
	if aliases, ok := tpl["aliases"]; ok {
		if aliasesMap, ok := aliases.(map[string]interface{}); ok && len(aliasesMap) == 0 {
			delete(tpl, "aliases")
		}
	}
	// Handle settings nested under template (ES 7.8+ format)
	if innerTpl, ok := tpl["template"].(map[string]interface{}); ok {
		if settings, ok := innerTpl["settings"]; ok {
			if settingsMap, ok := settings.(map[string]interface{}); ok {
				innerTpl["settings"] = normalizedIndexSettings(settingsMap)
			}
		}
	}
}

/*
normalizeComposableIndexTemplate normalizes an index_template (ES >= 7.8) Index template definition.
For legacy index templates (ES < 7.8) or /_template endpoint on ES >= 7.8 see normalizeIndexTemplate.
*/
func normalizeComposableIndexTemplate(tpl map[string]interface{}) {
	delete(tpl, "version")

	// data_stream accepts only the attribute "hidden", but can return additional attributes, so
	// remove them
	if dataStream, ok := tpl["data_stream"].(map[string]interface{}); ok {
		for k := range dataStream {
			if k != "hidden" {
				delete(dataStream, k)
			}
		}
	}

	if innerTpl, ok := tpl["template"]; ok {
		if innerTplMap, ok := innerTpl.(map[string]interface{}); ok {
			if settings, ok := innerTplMap["settings"]; ok {
				if settingsMap, ok := settings.(map[string]interface{}); ok {
					innerTplMap["settings"] = normalizedIndexSettings(settingsMap)
				}
			}
		}
	}
}

func normalizeComponentTemplate(tpl map[string]interface{}) {
	delete(tpl, "version")
	if innerTpl, ok := tpl["template"]; ok {
		if innerTplMap, ok := innerTpl.(map[string]interface{}); ok {
			if settings, ok := innerTplMap["settings"]; ok {
				if settingsMap, ok := settings.(map[string]interface{}); ok {
					innerTplMap["settings"] = normalizedIndexSettings(settingsMap)
				}
			}
		}
	}
}

func normalizedIndexSettings(settings map[string]interface{}) map[string]interface{} {
	f := flattenMap(settings)
	for k, v := range f {
		f[k] = fmt.Sprintf("%v", v)
		if !strings.HasPrefix(k, "index.") {
			f["index."+k] = fmt.Sprintf("%v", v)
			delete(f, k)
		}
	}

	return f
}

func normalizeISMState(state map[string]interface{}) {
	// Normalize actions to remove retry defaults
	if actions, ok := state["actions"].([]interface{}); ok {
		for _, a := range actions {
			if action, ok := a.(map[string]interface{}); ok {
				// Remove default retry configuration if present
				if retry, ok := action["retry"].(map[string]interface{}); ok {
					// Check if retry has default values
					isDefault := true
					if count, ok := retry["count"]; !ok || fmt.Sprintf("%v", count) != "3" {
						isDefault = false
					}
					if backoff, ok := retry["backoff"]; !ok || fmt.Sprintf("%v", backoff) != "exponential" {
						isDefault = false
					}
					if delay, ok := retry["delay"]; !ok || fmt.Sprintf("%v", delay) != "1m" {
						isDefault = false
					}
					if isDefault {
						delete(action, "retry")
					}
				}
				// Remove default rollover fields
				if rollover, ok := action["rollover"].(map[string]interface{}); ok {
					delete(rollover, "copy_alias")
				}
				// Remove default delete fields
				if _, ok := action["delete"]; ok {
					// delete action doesn't need normalization
				}
			}
		}
	}
}

func normalizeAnomalyDetection(tpl map[string]interface{}) {
	delete(tpl, "last_update_time")
	delete(tpl, "schema_version")
	delete(tpl, "user")

	// Normalize filter_query to remove default query values added by OpenSearch
	if filterQuery, ok := tpl["filter_query"].(map[string]interface{}); ok {
		NormalizeQueryDefaults(filterQuery)
	}

	// Remove API-generated fields
	delete(tpl, "detector_type")
	delete(tpl, "feature_id")
	delete(tpl, "flatten_custom_result_index")
	delete(tpl, "history")
	delete(tpl, "recency_emphasis")
	delete(tpl, "rules")
	delete(tpl, "shingle_size")

	// Normalize feature_attributes to remove feature_id
	if features, ok := tpl["feature_attributes"].([]interface{}); ok {
		for _, f := range features {
			if feature, ok := f.(map[string]interface{}); ok {
				delete(feature, "feature_id")
			}
		}
	}
}

func flattenMap(m map[string]interface{}) map[string]interface{} {
	f := make(map[string]interface{})
	for k, v := range m {
		if vm, ok := v.(map[string]interface{}); ok {
			fm := flattenMap(vm)
			for k2, v2 := range fm {
				f[k+"."+k2] = v2
			}
		} else {
			f[k] = v
		}
	}

	return f
}

func concatStringSlice(args ...[]string) []string {
	merged := make([]string, 0)
	for _, slice := range args {
		merged = append(merged, slice...)
	}
	return merged
}

func containsString(h []string, n string) bool {
	for _, e := range h {
		if e == n {
			return true
		}
	}
	return false
}

func functionallyEquivalentJSON(j1, j2 string) bool {
	var unmarshaled1, unmarshaled2 map[string]interface{}
	if err := json.Unmarshal([]byte(j1), &unmarshaled1); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(j2), &unmarshaled2); err != nil {
		return false
	}
	return reflect.DeepEqual(unmarshaled1, unmarshaled2)
}

// Takes the result of flatmap.Expand for an array of strings
// and returns a []string
func expandStringList(resourcesArray []interface{}) []string {
	vs := make([]string, 0, len(resourcesArray))
	for _, v := range resourcesArray {
		val, ok := v.(string)
		if ok && val != "" {
			vs = append(vs, v.(string))
		}
	}
	return vs
}

func flattenStringList(list []string) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, v)
	}
	return vs
}

func flattenStringSet(list []string) *schema.Set {
	return flattenStringAsInterfaceSet(flattenStringList(list))
}

func flattenStringAsInterfaceSet(list []interface{}) *schema.Set {
	return schema.NewSet(schema.HashString, list)
}

type resourceDataSetter struct {
	d   *schema.ResourceData
	err error
}

func (ds *resourceDataSetter) set(key string, value interface{}) {
	if ds.err != nil {
		return
	}
	ds.err = ds.d.Set(key, value)
}

func flattenIndexPermissions(permissions []IndexPermissions, d *schema.ResourceData) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(permissions))
	for _, permission := range permissions {
		p := make(map[string]interface{})

		if len(permission.IndexPatterns) > 0 {
			p["index_patterns"] = flattenStringSet(permission.IndexPatterns)
		}
		if len(permission.DocumentLevelSecurity) > 0 {
			p["document_level_security"] = permission.DocumentLevelSecurity
		}

		if len(permission.FieldLevelSecurity) > 0 {
			p["field_level_security"] = flattenStringSet(permission.FieldLevelSecurity)
		}

		if len(permission.MaskedFields) > 0 {
			p["masked_fields"] = flattenStringSet(permission.MaskedFields)
		}
		if len(permission.AllowedActions) > 0 {
			p["allowed_actions"] = flattenStringSet(permission.AllowedActions)
		}

		result = append(result, p)
	}

	return result
}

func expandIndexPermissionsSet(resourcesArray []interface{}) ([]IndexPermissions, error) {
	vperm := make([]IndexPermissions, 0, len(resourcesArray))
	for _, item := range resourcesArray {
		data, ok := item.(map[string]interface{})
		if !ok {
			return vperm, fmt.Errorf("Error asserting data as type []byte : %v", item)
		}

		fls := data["field_level_security"]
		flsList := fls.(*schema.Set).List()

		obj := IndexPermissions{
			IndexPatterns:         expandStringList(data["index_patterns"].(*schema.Set).List()),
			DocumentLevelSecurity: data["document_level_security"].(string),
			FieldLevelSecurity:    expandStringList(flsList),
			MaskedFields:          expandStringList(data["masked_fields"].(*schema.Set).List()),
			AllowedActions:        expandStringList(data["allowed_actions"].(*schema.Set).List()),
		}
		vperm = append(vperm, obj)
	}
	return vperm, nil
}

func flattenTenantPermissions(permissions []TenantPermissions) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(permissions))
	for _, permission := range permissions {
		p := make(map[string]interface{})

		if len(permission.TenantPatterns) > 0 {
			p["tenant_patterns"] = flattenStringSet(permission.TenantPatterns)
		}
		if len(permission.AllowedActions) > 0 {
			p["allowed_actions"] = flattenStringSet(permission.AllowedActions)
		}

		result = append(result, p)
	}

	return result
}

func expandTenantPermissionsSet(resourcesArray []interface{}) ([]TenantPermissions, error) {
	vperm := make([]TenantPermissions, 0, len(resourcesArray))
	for _, item := range resourcesArray {
		data, ok := item.(map[string]interface{})
		if !ok {
			return vperm, fmt.Errorf("Error asserting data as type []byte : %v", item)
		}
		obj := TenantPermissions{
			TenantPatterns: expandStringList(data["tenant_patterns"].(*schema.Set).List()),
			AllowedActions: expandStringList(data["allowed_actions"].(*schema.Set).List()),
		}
		vperm = append(vperm, obj)
	}
	return vperm, nil
}

func hashSum(contents interface{}) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(contents.(string))))
}

func indexPermissionsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	// We need to make sure to sort the strings below so that we always
	// generate the same hash code no matter what is in the set.
	if v, ok := m["index_patterns"]; ok {
		vs := v.(*schema.Set).List()
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	if v, ok := m["document_level_security"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["fls"]; ok {
		vs := v.(*schema.Set).List()
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	if v, ok := m["field_level_security"]; ok {
		vs := v.(*schema.Set).List()
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}
	if v, ok := m["masked_fields"]; ok {
		vs := v.(*schema.Set).List()
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}
	if v, ok := m["allowed_actions"]; ok {
		vs := v.(*schema.Set).List()
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	return hashcode(buf.String())
}

func tenantPermissionsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	// We need to make sure to sort the strings below so that we always
	// generate the same hash code no matter what is in the set.
	if v, ok := m["tenant_patterns"]; ok {
		vs := v.(*schema.Set).List()
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}
	if v, ok := m["allowed_actions"]; ok {
		vs := v.(*schema.Set).List()
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	return hashcode(buf.String())
}

// hashcode hashes a string to a unique hash code.
//
// crc32 returns a uint32, but for our use we need
// and non negative integer. Here we cast to an integer
// and invert it if the result is negative.
func hashcode(s string) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	if -v >= 0 {
		return -v
	}
	// v == MinInt
	return 0
}

// If the argument is a path, readPathOrContent loads it and returns the contents,
// otherwise the argument is assumed to be the desired contents and is simply
// returned.
//
// The boolean second return value can be called `wasPath` - it indicates if a
// path was detected and a file loaded.
func readPathOrContent(poc string) (string, bool, error) {
	if len(poc) == 0 {
		return poc, false, nil
	}

	path := poc
	if path[0] == '~' {
		var err error
		path, err = homedir.Expand(path)
		if err != nil {
			return path, true, err
		}
	}

	if _, err := os.Stat(path); err == nil {
		contents, err := os.ReadFile(path)
		if err != nil {
			return string(contents), true, err
		}
		return string(contents), true, nil
	}

	return poc, false, nil
}
