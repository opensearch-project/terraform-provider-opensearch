package provider

import (
	"fmt"
	"html"
)

// normalizeAnomalyDetection drops the fields the anomaly detection plugin adds
// or rewrites on save, so a detector read back compares equal to its config:
//   - last_update_time, detector_state, schema_version, user: server managed
//   - feature_attributes[].feature_id: generated per feature on every save
//   - description: HTML-escaped on save ("->" is stored as "-&amp;gt;")
//   - filter_query: parsed and re-serialized in long form, see normalizeQueryDSL
func normalizeAnomalyDetection(tpl map[string]any) {
	delete(tpl, "last_update_time")
	delete(tpl, "detector_state")
	delete(tpl, "schema_version")
	delete(tpl, "user")

	if features, ok := tpl["feature_attributes"].([]any); ok {
		for _, f := range features {
			if fm, ok := f.(map[string]any); ok {
				delete(fm, "feature_id")
			}
		}
	}
	if desc, ok := tpl["description"].(string); ok {
		tpl["description"] = unescapeHTMLFully(desc)
	}
	if fq, ok := tpl["filter_query"]; ok {
		tpl["filter_query"] = normalizeQueryDSL(fq)
	}
}

func unescapeHTMLFully(s string) string {
	for i := 0; i < 5; i++ {
		u := html.UnescapeString(s)
		if u == s {
			return s
		}
		s = u
	}
	return s
}

// normalizeQueryDSL rewrites a query to a short canonical form so the long
// form OpenSearch returns (explicit boost 1.0, term/match_phrase as objects,
// range as from/to/include_lower/include_upper, adjust_pure_negative,
// minimum_should_match as a string) equals the form usually written in config.
func normalizeQueryDSL(v any) any {
	switch t := v.(type) {
	case map[string]any:
		if b, ok := t["boost"].(float64); ok && b == 1 {
			delete(t, "boost")
		}
		if apn, ok := t["adjust_pure_negative"].(bool); ok && apn {
			delete(t, "adjust_pure_negative")
		}
		if msm, ok := t["minimum_should_match"].(float64); ok {
			t["minimum_should_match"] = fmt.Sprintf("%v", msm)
		}
		for k, child := range t {
			switch k {
			case "term":
				t[k] = shortFieldForm(normalizeQueryDSL(child), "value", nil)
			case "match_phrase", "match":
				t[k] = shortFieldForm(normalizeQueryDSL(child), "query", map[string]any{"slop": float64(0), "zero_terms_query": "NONE"})
			case "range":
				t[k] = normalizeRange(normalizeQueryDSL(child))
			default:
				t[k] = normalizeQueryDSL(child)
			}
		}
		return t
	case []any:
		for i, child := range t {
			t[i] = normalizeQueryDSL(child)
		}
		return t
	default:
		return v
	}
}

// shortFieldForm turns {"field": {"<key>": x}} into {"field": x} once the
// default-valued options are dropped.
func shortFieldForm(v any, key string, defaults map[string]any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	for field, opts := range m {
		om, ok := opts.(map[string]any)
		if !ok {
			continue
		}
		for dk, dv := range defaults {
			if om[dk] == dv {
				delete(om, dk)
			}
		}
		if inner, ok := om[key]; ok && len(om) == 1 {
			m[field] = inner
		}
	}
	return m
}

// normalizeRange rewrites from/to/include_lower/include_upper into gte/gt/lte/lt.
func normalizeRange(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	for _, opts := range m {
		om, ok := opts.(map[string]any)
		if !ok {
			continue
		}
		if from, ok := om["from"]; ok {
			inc, _ := om["include_lower"].(bool)
			if _, set := om["include_lower"]; !set {
				inc = true
			}
			if from != nil {
				if inc {
					om["gte"] = from
				} else {
					om["gt"] = from
				}
			}
			delete(om, "from")
			delete(om, "include_lower")
		}
		if to, ok := om["to"]; ok {
			inc, _ := om["include_upper"].(bool)
			if _, set := om["include_upper"]; !set {
				inc = true
			}
			if to != nil {
				if inc {
					om["lte"] = to
				} else {
					om["lt"] = to
				}
			}
			delete(om, "to")
			delete(om, "include_upper")
		}
	}
	return m
}
