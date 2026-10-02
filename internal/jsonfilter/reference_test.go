package jsonfilter

import (
	"encoding/json"
	"errors"
	"net/url"
	"testing"
)

func TestReferenceLastComponentPassiveSpelling(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{`null`, `null`},
		{`"https://foreign.invalid/v1/containers/raw%2Fid?ignored=one#fragment"`, `"raw%2Fid"`},
		{`"https://foreign.invalid/v1/containers/한 글"`, `"한 글"`},
		{`"https://foreign.invalid/v1/containers/"`, `""`},
		{`"https://foreign.invalid/v1/containers/%zz"`, `"%zz"`},
		{`"custom://user@host/path/final"`, `"final"`},
	} {
		got, err := ReferenceLastComponent(json.RawMessage(tc.raw))
		if err != nil || string(got) != tc.want {
			t.Fatal(tc.raw, string(got), err)
		}
		got[0] = 'x'
		again, err := ReferenceLastComponent(json.RawMessage(tc.raw))
		if err != nil || string(again) != tc.want {
			t.Fatal("returned value aliases subsequent calls", string(again), err)
		}
	}
}

func TestReferenceLastComponentErrors(t *testing.T) {
	for _, raw := range []string{`"relative/path"`, `"https:///path"`, `"https://host"`, `true`, `{}`, `"unterminated`} {
		if got, err := ReferenceLastComponent(json.RawMessage(raw)); got != nil || err == nil {
			t.Fatal(raw, string(got), err)
		}
	}
	_, err := ReferenceLastComponent(json.RawMessage(`"https://host:notport/path"`))
	var parse *url.Error
	if !errors.As(err, &parse) {
		t.Fatal("Go parser cause lost", err)
	}
}
