package mapper

import (
	"fmt"
	"reflect"
	"strings"

	"zonekit/pkg/dnsrecord"
)

// Mappings defines field mappings between our format and provider format
type Mappings struct {
	Request  FieldMapping
	Response FieldMapping
	ListPath string // JSON path to records array (e.g., "result" or "data.records")
}

// FieldMapping defines how to map fields
type FieldMapping struct {
	HostName   string
	RecordType string
	Address    string
	TTL        string
	MXPref     string
	ID         string
}

// DefaultMappings returns default mappings (no transformation needed)
func DefaultMappings() Mappings {
	return Mappings{
		Request: FieldMapping{
			HostName:   "hostname",
			RecordType: "record_type",
			Address:    "address",
			TTL:        "ttl",
			MXPref:     "mx_pref",
			ID:         "",
		},
		Response: FieldMapping{
			HostName:   "hostname",
			RecordType: "record_type",
			Address:    "address",
			TTL:        "ttl",
			MXPref:     "mx_pref",
			ID:         "",
		},
		ListPath: "records",
	}
}

// ToProviderFormat converts a dnsrecord.Record to provider's format
func ToProviderFormat(record dnsrecord.Record, mapping FieldMapping) map[string]interface{} {
	result := make(map[string]interface{})

	if mapping.HostName != "" {
		result[mapping.HostName] = record.HostName
	}
	if mapping.RecordType != "" {
		result[mapping.RecordType] = record.RecordType
	}
	if mapping.Address != "" {
		result[mapping.Address] = record.Address
	}
	if mapping.TTL != "" && record.TTL > 0 {
		result[mapping.TTL] = record.TTL
	}
	if mapping.MXPref != "" && record.MXPref > 0 {
		result[mapping.MXPref] = record.MXPref
	}
	if mapping.ID != "" && record.ID != "" {
		result[mapping.ID] = record.ID
	}

	return result
}

// FromProviderFormat converts provider's format to dnsrecord.Record
func FromProviderFormat(data map[string]interface{}, mapping FieldMapping) (dnsrecord.Record, error) {
	record := dnsrecord.Record{}

	// Helper to get string value
	getString := func(key string) string {
		if val, ok := data[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
			return fmt.Sprintf("%v", val)
		}
		return ""
	}

	// Helper to get int value
	getInt := func(key string) int {
		if val, ok := data[key]; ok {
			switch v := val.(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			}
		}
		return 0
	}

	if mapping.HostName != "" {
		record.HostName = getString(mapping.HostName)
	}
	if mapping.RecordType != "" {
		record.RecordType = getString(mapping.RecordType)
	}
	if mapping.Address != "" {
		record.Address = getString(mapping.Address)
	}
	if mapping.TTL != "" {
		record.TTL = getInt(mapping.TTL)
	}
	if mapping.MXPref != "" {
		record.MXPref = getInt(mapping.MXPref)
	}
	if mapping.ID != "" {
		record.ID = getString(mapping.ID)
	}

	return record, nil
}

// ExtractRecords extracts records from a JSON response using the list path
func ExtractRecords(data interface{}, listPath string) ([]map[string]interface{}, error) {
	if listPath == "" {
		// Default: assume data is an array
		if arr, ok := data.([]interface{}); ok {
			return convertArrayToMaps(arr)
		}
		return nil, fmt.Errorf("no list path specified and data is not an array")
	}

	// Navigate through the path (e.g., "result" or "data.records")
	parts := strings.Split(listPath, ".")
	current := reflect.ValueOf(data)

	for _, part := range parts {
		if current.Kind() == reflect.Interface {
			current = current.Elem()
		}

		switch current.Kind() {
		case reflect.Map:
			key := reflect.ValueOf(part)
			current = current.MapIndex(key)
			if !current.IsValid() {
				return nil, fmt.Errorf("path '%s' not found in response", listPath)
			}
		case reflect.Slice, reflect.Array:
			// If we hit an array/slice, we're done navigating
			break
		default:
			return nil, fmt.Errorf("invalid path '%s': cannot navigate through %v", listPath, current.Kind())
		}
	}

	// Convert to array of maps
	if current.Kind() == reflect.Interface {
		current = current.Elem()
	}

	if current.Kind() != reflect.Slice && current.Kind() != reflect.Array {
		return nil, fmt.Errorf("path '%s' does not point to an array", listPath)
	}

	arr := make([]interface{}, current.Len())
	for i := 0; i < current.Len(); i++ {
		arr[i] = current.Index(i).Interface()
	}

	return convertArrayToMaps(arr)
}

// convertArrayToMaps converts an array of interfaces to array of maps
func convertArrayToMaps(arr []interface{}) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0, len(arr))

	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		} else {
			// Try to convert using reflection
			val := reflect.ValueOf(item)
			if val.Kind() == reflect.Interface {
				val = val.Elem()
			}

			if val.Kind() == reflect.Map {
				m := make(map[string]interface{})
				for _, key := range val.MapKeys() {
					m[key.String()] = val.MapIndex(key).Interface()
				}
				result = append(result, m)
			} else {
				return nil, fmt.Errorf("cannot convert item to map: %v", item)
			}
		}
	}

	return result, nil
}
