// Package provider provides Terraform provider for OpenSearch.
// This file contains migration utilities for transitioning from olivere/elastic to opensearch-go/v4.
package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// handleError processes errors from the new opensearch-go/v4 client
// and returns appropriate errors for Terraform
func handleError(err error, resourceType string) error {
	if err == nil {
		return nil
	}

	// Check for HTTP status codes in error messages
	errStr := err.Error()
	if strings.Contains(errStr, "404") || strings.Contains(errStr, "not found") {
		return fmt.Errorf("%s not found: %w", resourceType, err)
	}
	if strings.Contains(errStr, "403") {
		return fmt.Errorf("permission denied accessing %s: %w", resourceType, err)
	}
	if strings.Contains(errStr, "401") {
		return fmt.Errorf("unauthorized accessing %s: %w", resourceType, err)
	}
	if strings.Contains(errStr, "409") {
		return fmt.Errorf("conflict with %s: %w", resourceType, err)
	}

	return fmt.Errorf("error with %s: %w", resourceType, err)
}

// isNotFound checks if an error represents a "not found" response
// This replaces elastic7.IsNotFound(err)
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "Not Found") ||
		// ISM policy mapping returns 400 with "no documents to get" when index doesn't exist
		(strings.Contains(errStr, "400") && strings.Contains(errStr, "no documents to get"))
}

// isConflict checks if an error represents a "conflict" response
// This replaces elastic7.IsConflict(err)
func isConflict(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "409") ||
		strings.Contains(errStr, "conflict") ||
		strings.Contains(errStr, "Conflict")
}

// mapToJSON converts a map to a JSON string
func mapToJSON(m map[string]interface{}) (string, error) {
	if m == nil || len(m) == 0 {
		return "", nil
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// jsonToMap converts a JSON string to a map
func jsonToMap(s string) (map[string]interface{}, error) {
	if s == "" {
		return nil, nil
	}
	var m map[string]interface{}
	err := json.Unmarshal([]byte(s), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// jsonToReader converts a map to an io.Reader for API requests
func jsonToReader(data interface{}) (io.Reader, error) {
	if data == nil {
		return strings.NewReader(""), nil
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(dataBytes), nil
}

// flattenSettings flattens nested settings maps into a flat map with dot notation
// This is needed because OpenSearch returns settings in a nested format
func flattenSettings(settings map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range settings {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Recursively flatten nested maps
			nested := flattenSettings(v, newKey)
			for nk, nv := range nested {
				result[nk] = nv
			}
		default:
			result[newKey] = v
		}
	}

	return result
}

// unflattenSettings converts a flat map with dot notation into a nested map
func unflattenSettings(settings map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range settings {
		parts := strings.Split(key, ".")
		current := result

		for i, part := range parts {
			if i == len(parts)-1 {
				// Last part - set the value
				current[part] = value
			} else {
				// Create nested map if needed
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
				}
				current = current[part].(map[string]interface{})
			}
		}
	}

	return result
}

// getStringOrDefault returns a string from ResourceData or a default value
func getStringOrDefault(d *schema.ResourceData, key, defaultValue string) string {
	if v, ok := d.GetOk(key); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return defaultValue
}

// getBoolOrDefault returns a bool from ResourceData or a default value
func getBoolOrDefault(d *schema.ResourceData, key string, defaultValue bool) bool {
	if v, ok := d.GetOkExists(key); ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// normalizeJSON normalizes a JSON string for comparison
// This removes whitespace differences
func normalizeJSON(s string) (string, error) {
	var data interface{}
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// jsonEqual compares two JSON strings for equality
func jsonEqual(a, b string) (bool, error) {
	if a == b {
		return true, nil
	}

	normalizedA, err := normalizeJSON(a)
	if err != nil {
		return false, err
	}

	normalizedB, err := normalizeJSON(b)
	if err != nil {
		return false, err
	}

	return normalizedA == normalizedB, nil
}

// httpStatusFromError extracts HTTP status code from error message
// This is a best-effort approach since the SDK may not always expose status codes directly
func httpStatusFromError(err error) int {
	if err == nil {
		return 0
	}

	errStr := err.Error()

	// Common HTTP status codes in error messages
	statusCodes := map[string]int{
		"400": http.StatusBadRequest,
		"401": http.StatusUnauthorized,
		"403": http.StatusForbidden,
		"404": http.StatusNotFound,
		"409": http.StatusConflict,
		"500": http.StatusInternalServerError,
		"502": http.StatusBadGateway,
		"503": http.StatusServiceUnavailable,
	}

	for codeStr, status := range statusCodes {
		if strings.Contains(errStr, codeStr) {
			return status
		}
	}

	return 0
}
