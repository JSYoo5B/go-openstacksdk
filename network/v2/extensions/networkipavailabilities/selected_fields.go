package networkipavailabilities

import "maps"

// extractSelectedNetworkIPAvailability adapts a sparse selected-field response
// to the native NetworkIPAvailability decoder, which rejects a row without
// total_ips or used_ips. Omitted counters decode as zero and are then reported
// as the typed model's zero value (""). Explicit null and every present value
// keep native validation. The original body and nested maps remain unchanged.
func extractSelectedNetworkIPAvailability(result GetResult) (*NetworkIPAvailability, error) {
	if result.Err != nil {
		return result.Extract()
	}
	envelope, ok := result.Body.(map[string]any)
	if !ok || envelope == nil {
		return result.Extract()
	}
	row, ok := envelope["network_ip_availability"].(map[string]any)
	if !ok || row == nil {
		return result.Extract()
	}
	_, hasTotal := row["total_ips"]
	_, hasUsed := row["used_ips"]
	if hasTotal && hasUsed {
		return result.Extract()
	}
	projected := maps.Clone(row)
	if !hasTotal {
		projected["total_ips"] = 0
	}
	if !hasUsed {
		projected["used_ips"] = 0
	}
	body := maps.Clone(envelope)
	body["network_ip_availability"] = projected
	selected := result
	selected.Body = body
	value, err := selected.Extract()
	if err != nil || value == nil {
		return value, err
	}
	if !hasTotal {
		value.TotalIPs = ""
	}
	if !hasUsed {
		value.UsedIPs = ""
	}
	return value, nil
}
