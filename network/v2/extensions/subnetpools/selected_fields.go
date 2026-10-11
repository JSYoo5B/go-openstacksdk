package subnetpools

import "maps"

// extractSelectedSubnetPool adapts a sparse selected-field response to the
// native SubnetPool decoder, which requires all three prefix length keys.
// Only omitted keys get the typed model's zero value. Explicit null and every
// present value keep native validation. The original body and nested resource
// maps remain unchanged, including unknown response attributes.
func extractSelectedSubnetPool(result GetResult) (*SubnetPool, error) {
	if result.Err != nil {
		return result.Extract()
	}
	envelope, ok := result.Body.(map[string]any)
	if !ok || envelope == nil {
		return result.Extract()
	}
	row, ok := envelope["subnetpool"].(map[string]any)
	if !ok || row == nil {
		return result.Extract()
	}
	prefixes := [...]string{"default_prefixlen", "min_prefixlen", "max_prefixlen"}
	var projected map[string]any
	for _, key := range prefixes {
		if _, present := row[key]; present {
			continue
		}
		if projected == nil {
			projected = maps.Clone(row)
		}
		projected[key] = 0
	}
	if projected == nil {
		return result.Extract()
	}
	body := maps.Clone(envelope)
	body["subnetpool"] = projected
	selected := result
	selected.Body = body
	return selected.Extract()
}
