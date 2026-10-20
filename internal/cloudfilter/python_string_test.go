package cloudfilter

import (
	"encoding/json"
	"testing"
)

func TestPythonStringKeepsWholeDescriptorValue(t *testing.T) {
	for _, test := range []struct{ raw, want string }{
		{`null`, "None"}, {`true`, "True"}, {`false`, "False"},
		{`"true"`, "true"}, {`"空 白"`, "空 白"},
		{`9007199254740993`, "9007199254740993"}, {`-0`, "0"},
		{`1.0`, "1.0"}, {`[]`, "[]"},
		{`[true,null,"a"]`, "[True, None, 'a']"},
		{`{"first":1,"second":[false],"first":2}`, "{'first': 2, 'second': [False]}"},
	} {
		t.Run(test.raw, func(t *testing.T) {
			got, err := PythonString(json.RawMessage(test.raw))
			if err != nil || got != test.want {
				t.Fatalf("got=%q want=%q err=%v", got, test.want, err)
			}
		})
	}
	for _, raw := range []string{"", `true false`, `{"x":!}`, "\"\xff\""} {
		if got, err := PythonString(json.RawMessage(raw)); err == nil {
			t.Fatalf("invalid descriptor %q returned %q", raw, got)
		}
	}
}
