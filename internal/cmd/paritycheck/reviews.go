package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type contractEvidence struct {
	Behavior string   `json:"behavior"`
	Tests    []string `json:"tests"`
}

type supportReview struct {
	ID          string             `json:"operation"`
	Fingerprint string             `json:"source_fingerprint"`
	Status      string             `json:"status"`
	GoAPI       []string           `json:"go_api,omitempty"`
	Contracts   []contractEvidence `json:"contracts,omitempty"`
	Differences []string           `json:"differences,omitempty"`
	Docs        []string           `json:"documentation,omitempty"`
	Remaining   []string           `json:"remaining,omitempty"`
}

type reviewLedger struct {
	Schema  int             `json:"schema_version"`
	Pins    sourcePins      `json:"source_pins"`
	Reviews []supportReview `json:"reviews"`
}

type supportCounts struct {
	total  int
	status map[string]int
}

func check(root string, sync bool) (supportCounts, error) {
	var counts supportCounts
	current, err := loadInventory(root)
	if err != nil {
		return counts, err
	}
	var ledger reviewLedger
	if err := readJSON(filepath.Join(root, "api/sdk_reviews.json"), &ledger); err != nil {
		return counts, err
	}
	if ledger.Schema != 1 || ledger.Pins != pins {
		return counts, fmt.Errorf("review schema or source pins do not match the current inventory")
	}
	if !sync {
		var catalog operationCatalog
		if err := readJSON(filepath.Join(root, "api/sdk_support_catalog.json"), &catalog); err != nil {
			return counts, err
		}
		if catalog.Schema != 1 || catalog.Pins != pins || !reflect.DeepEqual(catalog.Operations, current.Operations) {
			return counts, fmt.Errorf("operation catalog is stale or incomplete; review source changes and run paritycheck -sync")
		}
	}
	symbols, err := goSymbols(root)
	if err != nil {
		return counts, err
	}
	counts = supportCounts{total: len(current.Operations), status: map[string]int{"unresolved": len(current.Operations)}}
	seen := map[string]bool{}
	for _, review := range ledger.Reviews {
		if seen[review.ID] {
			return counts, fmt.Errorf("duplicate review operation %q", review.ID)
		}
		seen[review.ID] = true
		if err := validateReview(root, current, symbols, review); err != nil {
			return counts, fmt.Errorf("review %s: %w", review.ID, err)
		}
		counts.status["unresolved"]--
		counts.status[review.Status]++
	}
	if sync {
		data, err := json.MarshalIndent(current, "", "  ")
		if err != nil {
			return counts, err
		}
		// Reviews are deliberately not rewritten. A changed source fingerprint
		// fails above even during sync, leaving both files available for review.
		if err := os.WriteFile(filepath.Join(root, "api/sdk_support_catalog.json"), append(data, '\n'), 0644); err != nil {
			return counts, err
		}
	}
	return counts, nil
}

func validateReview(root string, catalog operationCatalog, symbols symbolIndex, review supportReview) error {
	fingerprint, ok := catalog.Operations[review.ID]
	if !ok {
		return fmt.Errorf("operation does not exist in the pinned declared inventory")
	}
	if review.Fingerprint != fingerprint {
		return fmt.Errorf("source fingerprint changed or is missing; reassess the contract before keeping its status")
	}
	switch review.Status {
	case "supported":
		if len(review.Remaining) != 0 || len(review.Differences) != 0 {
			return fmt.Errorf("supported cannot have remaining contracts or behavioral differences")
		}
	case "go_mapping":
		if !nonempty(review.Differences) || len(review.Remaining) != 0 {
			return fmt.Errorf("go_mapping requires differences and no remaining contracts")
		}
	case "unsupported", "unresolved":
		if !nonempty(review.Remaining) {
			return fmt.Errorf("incomplete review requires specific remaining contracts")
		}
	default:
		return fmt.Errorf("unknown status %q", review.Status)
	}
	complete := review.Status == "supported" || review.Status == "go_mapping"
	if complete && (!nonempty(review.GoAPI) || len(review.Contracts) == 0 || !nonempty(review.Docs)) {
		return fmt.Errorf("complete review requires Go APIs, tested contracts and usage documentation")
	}
	for _, api := range review.GoAPI {
		if !symbols.api[api] {
			return fmt.Errorf("Go API %q is not an exported declaration", api)
		}
	}
	for _, doc := range review.Docs {
		if _, err := localFile(root, doc); err != nil {
			return err
		}
	}
	for _, contract := range review.Contracts {
		if strings.TrimSpace(contract.Behavior) == "" || !nonempty(contract.Tests) {
			return fmt.Errorf("each claimed contract requires a behavior and test evidence")
		}
		for _, test := range contract.Tests {
			if !symbols.tests[test] {
				return fmt.Errorf("test evidence %q is not a declared Test function", test)
			}
		}
	}
	return nil
}

func nonempty(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}
