package jmespath_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/jmespath"
	"github.com/JSYoo5B/gophercloudsdk/internal/jsonfilter"
)

// The primary-source fixture distinguishes Python 1.0.1 runtime goldens from
// explicitly declared Go representation, ordering, arithmetic and grammar
// boundaries. Decode every input with UseNumber; never pass float64 to Search.
//
//go:embed engine_reference_cases.json
var engineReferenceCases []byte

type engineCase struct {
	Name           string          `json:"name"`
	Expression     string          `json:"expression"`
	InputJSON      string          `json:"input_json"`
	Expected       json.RawMessage `json:"expected"`
	ExpectedJSON   string          `json:"expected_json"`
	Error          bool            `json:"error"`
	Policy         string          `json:"policy"`
	ReferenceError string          `json:"reference_error"`
}

func engineInput(t *testing.T, text string) any {
	t.Helper()
	if !json.Valid([]byte(text)) {
		t.Fatalf("invalid test input JSON: %q", text)
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func engineJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func engineWantJSON(t *testing.T, value any, want json.RawMessage) {
	t.Helper()
	actual := engineJSON(t, value)
	equal, err := jsonfilter.EqualJSON(actual, want)
	if err != nil {
		t.Fatalf("independent exact-JSON comparison: %v", err)
	}
	if !equal {
		t.Fatalf("got %s, want %s", actual, want)
	}
}

func TestSearchReferenceAndDeclaredBoundaries(t *testing.T) {
	var cases []engineCase
	if err := json.Unmarshal(engineReferenceCases, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 146 {
		t.Fatalf("fixture cases = %d, want 146", len(cases))
	}
	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			input := engineInput(t, test.InputJSON)
			before := engineJSON(t, input)
			value, err := jmespath.Search(test.Expression, input)
			after := engineJSON(t, input)
			if !bytes.Equal(before, after) {
				t.Fatalf("Search mutated its input: before %s, after %s", before, after)
			}
			if test.Error {
				if err == nil {
					t.Fatalf("expected error (%s; %s), got %s", test.Policy, test.ReferenceError, engineJSON(t, value))
				}
				return
			}
			if err != nil {
				t.Fatalf("Search: %v (%s)", err, test.Policy)
			}
			want := test.Expected
			if test.ExpectedJSON != "" {
				want = json.RawMessage(test.ExpectedJSON)
			}
			engineWantJSON(t, value, want)
		})
	}
}

func TestSearchLiteralNumberRepresentation(t *testing.T) {
	value, err := jmespath.Search("`9007199254740993`", nil)
	if err != nil {
		t.Fatal(err)
	}
	number, ok := value.(json.Number)
	if !ok || number.String() != "9007199254740993" {
		t.Fatalf("literal number = %T(%v), want exact json.Number", value, value)
	}
}

func TestCompiledLiteralResultOwnership(t *testing.T) {
	const expression = "`{\"items\":[{\"n\":9007199254740993}],\"flag\":true}`"
	compiled, err := jmespath.Compile(expression)
	if err != nil {
		t.Fatal(err)
	}
	first, err := compiled.Search(nil)
	if err != nil {
		t.Fatal(err)
	}
	object := first.(map[string]any)
	object["items"].([]any)[0].(map[string]any)["n"] = json.Number("1")
	object["flag"] = false
	object["extra"] = "changed"
	second, err := compiled.Search(nil)
	if err != nil {
		t.Fatal(err)
	}
	engineWantJSON(t, second, json.RawMessage(`{"items":[{"n":9007199254740993}],"flag":true}`))
}

func TestCompiledSearchConcurrent(t *testing.T) {
	compiled, err := jmespath.Compile("sort_by(items, &n)[].id")
	if err != nil {
		t.Fatal(err)
	}
	input := engineInput(t, `{"items":[{"id":"higher","n":9007199254740993},{"id":"lower","n":9007199254740992}]}`)
	before := engineJSON(t, input)
	const workers = 24
	var group sync.WaitGroup
	failures := make(chan error, workers)
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for n := 0; n < 20; n++ {
				value, err := compiled.Search(input)
				if err != nil {
					failures <- err
					return
				}
				raw, err := json.Marshal(value)
				if err != nil || string(raw) != `["lower","higher"]` {
					failures <- fmt.Errorf("concurrent Search = %s, error %v", raw, err)
					return
				}
			}
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if after := engineJSON(t, input); !bytes.Equal(before, after) {
		t.Fatalf("compiled concurrent Search mutated shared input: %s", after)
	}
}

func TestCompiledLiteralConcurrentOwnership(t *testing.T) {
	compiled, err := jmespath.Compile("`{\"rows\":[{\"n\":9007199254740993}]}`")
	if err != nil {
		t.Fatal(err)
	}
	const workers = 24
	var group sync.WaitGroup
	failures := make(chan error, workers)
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for n := 0; n < 20; n++ {
				value, err := compiled.Search(nil)
				if err != nil {
					failures <- err
					return
				}
				row := value.(map[string]any)["rows"].([]any)[0].(map[string]any)
				if number, ok := row["n"].(json.Number); !ok || number.String() != "9007199254740993" {
					failures <- fmt.Errorf("compiled literal inherited result mutation: %v", row)
					return
				}
				row["n"] = json.Number("0")
			}
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}

func TestLexerSyntaxBoundaries(t *testing.T) {
	for _, expression := range []string{"a\u0080", string([]byte{0xff}), "'unterminated", "[:::]"} {
		t.Run(fmt.Sprintf("%q", expression), func(t *testing.T) {
			_, err := jmespath.Compile(expression)
			if err == nil {
				t.Fatal("malformed expression compiled")
			}
			var syntax jmespath.SyntaxError
			if !errors.As(err, &syntax) {
				t.Fatalf("expected SyntaxError, got %T: %v", err, err)
			}
		})
	}
}

func TestMustCompileContract(t *testing.T) {
	value, err := jmespath.MustCompile("length(@)").Search("한😀")
	if err != nil {
		t.Fatal(err)
	}
	engineWantJSON(t, value, json.RawMessage("2"))
	t.Run("syntax_panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("MustCompile did not panic on syntax error")
			}
		}()
		jmespath.MustCompile("[:::]")
	})
}
