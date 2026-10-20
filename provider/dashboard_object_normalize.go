package provider

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Saved-object attributes that hold JSON documents encoded as strings.
var dashboardObjectJSONStringFields = []string{"visState", "uiStateJSON", "panelsJSON", "optionsJSON"}

// refNameKeys maps the "*RefName" keys written by the Dashboards saved-object
// migration to the inline key the reference replaces.
var refNameKeys = map[string]string{
	"indexRefName":        "index",
	"indexPatternRefName": "indexPattern",
	"savedSearchRefName":  "savedSearchId",
}

// normalizeDashboardObjectReferences folds "references" back into the
// attributes of a saved object's _source. When Dashboards starts it migrates
// saved objects and moves inline ids (searchSourceJSON.index,
// filter[].meta.index, panelsJSON id/type, savedSearchId,
// controls[].indexPattern) into "references" plus "*RefName" keys, so the stored
// object never matches the inline form Terraform wrote. JSON-string attributes
// are re-encoded so key order and spacing do not matter either.
// "references" is only dropped when every entry was resolved.
func normalizeDashboardObjectReferences(source map[string]any) {
	objType, _ := source["type"].(string)
	attrs, _ := source[objType].(map[string]any)
	if attrs == nil {
		return
	}
	refs := dashboardObjectReferences(source["references"])
	used := map[string]bool{}

	if name, ok := attrs["savedSearchRefName"].(string); ok {
		if ref, ok := refs[name]; ok {
			attrs["savedSearchId"] = ref["id"]
			delete(attrs, "savedSearchRefName")
			used[name] = true
		}
	}
	if meta, ok := attrs["kibanaSavedObjectMeta"].(map[string]any); ok {
		if s, ok := meta["searchSourceJSON"].(string); ok {
			meta["searchSourceJSON"] = rewriteJSONString(s, refs, used)
		}
	}
	for _, f := range dashboardObjectJSONStringFields {
		if s, ok := attrs[f].(string); ok {
			attrs[f] = rewriteJSONString(s, refs, used)
		}
	}
	if _, hasRefs := source["references"]; hasRefs && len(used) == len(refs) {
		delete(source, "references")
	}
}

func dashboardObjectReferences(v any) map[string]map[string]any {
	refs := map[string]map[string]any{}
	list, _ := v.([]any)
	for _, r := range list {
		if ref, ok := r.(map[string]any); ok {
			if name, ok := ref["name"].(string); ok {
				refs[name] = ref
			}
		}
	}
	return refs
}

// rewriteJSONString decodes a JSON-string attribute, inlines references and
// re-encodes it canonically. Strings that are not JSON are returned unchanged.
func rewriteJSONString(s string, refs map[string]map[string]any, used map[string]bool) string {
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return s
	}
	v = inlineReferences(v, refs, used)
	out, err := marshalNoEscape(v)
	if err != nil {
		return s
	}
	return out
}

func inlineReferences(v any, refs map[string]map[string]any, used map[string]bool) any {
	switch t := v.(type) {
	case map[string]any:
		for refKey, inlineKey := range refNameKeys {
			if name, ok := t[refKey].(string); ok {
				if ref, ok := refs[name]; ok {
					t[inlineKey] = ref["id"]
					delete(t, refKey)
					used[name] = true
				}
			}
		}
		// dashboard panels: panelRefName -> id + type
		if name, ok := t["panelRefName"].(string); ok {
			if ref, ok := refs[name]; ok {
				t["id"] = ref["id"]
				t["type"] = ref["type"]
				delete(t, "panelRefName")
				used[name] = true
			}
		}
		for k, child := range t {
			t[k] = inlineReferences(child, refs, used)
		}
		return t
	case []any:
		for i, child := range t {
			t[i] = inlineReferences(child, refs, used)
		}
		return t
	default:
		return v
	}
}

func marshalNoEscape(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}
