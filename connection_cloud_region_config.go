package openstack

import (
	"fmt"
	"maps"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvendors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type cloudRegionEntry struct {
	name   string
	values map[string]any
}

// applyRegionValues mirrors the region step of the pinned
// OpenStackConfig.get_one for SDK-owned settings. The known regions are the
// cloud's own regions list; a cloud with region_name but no regions has only
// value-less entries; otherwise the regions of the merged profile are used,
// as _get_known_regions does. Entries are names or {name, values} objects.
// An explicit, nonempty WithRegion must name an entry. Without it the entry
// named by region_name is selected when present, so that the values match the
// native endpoint region, and otherwise the first entry, which is Python's
// default. Every non-null value of the selected entry replaces the merged
// setting, including whole networks lists.
func applyRegionValues(cloudSettings, settings map[string]any, explicit *string) (map[string]any, error) {
	raw, present := cloudSettings["regions"]
	if !present {
		if _, named := cloudSettings["region_name"]; named {
			return settings, nil
		}
		raw, present = settings["regions"]
	}
	if !present || raw == nil {
		return settings, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, invalid("regions must be a list")
	}
	entries := make([]cloudRegionEntry, 0, len(list))
	for i, item := range list {
		entry, err := parseRegionEntry(i, item)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return settings, nil
	}
	var selected *cloudRegionEntry
	if explicit != nil && *explicit != "" {
		for i := range entries {
			if entries[i].name == *explicit {
				selected = &entries[i]
				break
			}
		}
		if selected == nil {
			names := make([]string, len(entries))
			for i, entry := range entries {
				names[i] = entry.name
			}
			return nil, invalid("region %s is not a valid region name for this cloud; valid choices are %s (region names are case sensitive)", *explicit, strings.Join(names, ","))
		}
	} else {
		selected = &entries[0]
		if name, ok := settings["region_name"].(string); ok && name != "" {
			for i := range entries {
				if entries[i].name == name {
					selected = &entries[i]
					break
				}
			}
		}
	}
	if len(selected.values) == 0 {
		return settings, nil
	}
	result := maps.Clone(settings)
	if result == nil {
		result = make(map[string]any)
	}
	for key, value := range selected.values {
		if value != nil {
			result[key] = value
		}
	}
	return result, nil
}

func parseRegionEntry(index int, item any) (cloudRegionEntry, error) {
	switch value := item.(type) {
	case string:
		return cloudRegionEntry{name: value}, nil
	case map[any]any:
		entry := cloudRegionEntry{}
		for key, field := range value {
			switch key {
			case "name":
				name, ok := field.(string)
				if !ok {
					return entry, invalid("region entry %d name must be a string", index)
				}
				entry.name = name
			case "values":
				values, ok := field.(map[any]any)
				if !ok {
					return entry, invalid("region entry %d values must be an object", index)
				}
				entry.values = make(map[string]any, len(values))
				for k, v := range values {
					name, ok := k.(string)
					if !ok {
						return entry, invalid("region entry %d value keys must be strings", index)
					}
					entry.values[name] = v
				}
			default:
				return entry, invalid("invalid region entry %d: only name and values are allowed", index)
			}
		}
		if _, named := value["name"]; !named {
			return entry, invalid("invalid region entry %d: name is required", index)
		}
		return entry, nil
	default:
		return cloudRegionEntry{}, invalid("invalid region entry %d", index)
	}
}

// builtinCloudProfile resolves a pinned built-in vendor profile. A shutdown
// or remote profile is a configuration error.
func builtinCloudProfile(name string) (map[string]any, bool, error) {
	profile, found, err := cloudvendors.Profile(name)
	if err != nil {
		return nil, found, fmt.Errorf("%w: %w", resource.ErrInvalidOption, err)
	}
	return profile, found, nil
}
