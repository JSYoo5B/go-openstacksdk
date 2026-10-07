package cloudfilter_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/jmespath"
)

//go:embed testdata/python_cloud_filter_reference.json
var cloudFilterContractReferenceJSON []byte

type cloudFilterContractReferenceCase struct {
	Name       string            `json:"name"`
	Entry      string            `json:"entry"`
	Input      []json.RawMessage `json:"input"`
	NameOrID   *string           `json:"name_or_id"`
	Filters    json.RawMessage   `json:"filters"`
	FiltersRaw string            `json:"filters_raw"`
	Result     json.RawMessage   `json:"result"`
	ErrorType  string            `json:"error_type"`
}

func cloudFilterContractReferenceRun(c cloudFilterContractReferenceCase) (json.RawMessage, string, error) {
	pattern := ""
	if c.NameOrID != nil {
		pattern = *c.NameOrID
	}
	filter := c.Filters
	if c.FiltersRaw != "" {
		filter = json.RawMessage(c.FiltersRaw)
	}
	value, err := cloudfilter.Select(c.Input, pattern, &filter, nil)
	if err != nil {
		var shape any
		decoder := json.NewDecoder(bytes.NewReader(filter))
		decoder.UseNumber()
		_ = decoder.Decode(&shape)
		if text, ok := shape.(string); ok && text != "" {
			var syntax jmespath.SyntaxError
			if errors.As(err, &syntax) {
				return nil, "expression-syntax", err
			}
			return nil, "expression-runtime", err
		}
		return nil, "mapping", err
	}
	if c.Entry != "get_volume" {
		return value.Value, "", nil
	}
	selected, err := cloudfilter.First(value.Value)
	if err != nil {
		var ambiguity *cloudfilter.MultipleError
		if errors.As(err, &ambiguity) {
			return nil, "multiple", err
		}
		var shape any
		decoder := json.NewDecoder(bytes.NewReader(value.Value))
		decoder.UseNumber()
		_ = decoder.Decode(&shape)
		if _, ok := shape.(map[string]any); ok {
			return nil, "index", err
		}
		return nil, "length", err
	}
	if selected == nil {
		selected = json.RawMessage("null")
	}
	return selected, "", nil
}
func cloudFilterContractExpectedCategory(c cloudFilterContractReferenceCase) string {
	if c.ErrorType == "" {
		return ""
	}
	switch c.ErrorType {
	case "SDKException":
		return "multiple"
	case "KeyError":
		return "index"
	case "AttributeError":
		return "mapping"
	case "TypeError":
		if c.Entry == "get_volume" {
			return "length"
		}
		return "mapping"
	case "IncompleteExpressionError", "ParseError", "LexerError":
		return "expression-syntax"
	default:
		return "expression-runtime"
	}
}

func TestCloudFilterRepairedPinnedReferenceSnapshotsRetainValuesAndErrorPhases(t *testing.T) {
	var fixture struct {
		SourceSHA string                             `json:"source_sha256"`
		Cases     []cloudFilterContractReferenceCase `json:"cases"`
	}
	decoder := json.NewDecoder(bytes.NewReader(cloudFilterContractReferenceJSON))
	decoder.UseNumber()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SourceSHA != "aa2ad7d6375d70247183d807a2826a04a042a5ce169b1f22dbf6051a18a3e044" || len(fixture.Cases) != 30 {
		t.Fatal("reference snapshot provenance/case count changed", fixture.SourceSHA, len(fixture.Cases))
	}
	for _, c := range fixture.Cases {
		t.Run(c.Name, func(t *testing.T) {
			value, category, err := cloudFilterContractReferenceRun(c)
			expected := cloudFilterContractExpectedCategory(c)
			if expected != "" {
				if err == nil || category != expected || value != nil {
					t.Fatal("reference error phase changed", category, err, expected, string(value))
				}
				return
			}
			if err != nil || category != "" {
				t.Fatal(category, err)
			}
			cloudFilterContractEqual(t, value, string(c.Result))
		})
	}
}
