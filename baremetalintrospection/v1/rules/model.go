package rules

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicrest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Condition is one rule condition. Value is sent as given (omitted when empty)
// so strings, numbers, arrays and null stay exact. Extra holds members this
// package does not model and may not replace a typed member.
type Condition struct {
	Op       string
	Field    string
	Value    json.RawMessage
	Invert   *bool
	Multiple *string
	Extra    map[string]json.RawMessage
}

func (c Condition) MarshalJSON() ([]byte, error) {
	core := map[string]any{"op": c.Op, "field": c.Field}
	if len(c.Value) > 0 {
		core["value"] = c.Value
	}
	if c.Invert != nil {
		core["invert"] = *c.Invert
	}
	if c.Multiple != nil {
		core["multiple"] = *c.Multiple
	}
	return ironicrest.MarshalFlat(core, c.Extra)
}

func (c *Condition) UnmarshalJSON(data []byte) error {
	var v struct {
		Op       string          `json:"op"`
		Field    string          `json:"field"`
		Value    json.RawMessage `json:"value"`
		Invert   *bool           `json:"invert"`
		Multiple *string         `json:"multiple"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	extra, err := ironicrest.SplitExtra(data, "op", "field", "value", "invert", "multiple")
	if err != nil {
		return err
	}
	*c = Condition{Op: v.Op, Field: v.Field, Value: v.Value, Invert: v.Invert, Multiple: v.Multiple, Extra: extra}
	return nil
}

// Action is one rule action. Its parameters (path, value, name, ...) depend on
// the action and are carried in Params, which may not replace the action member.
type Action struct {
	Action string
	Params map[string]json.RawMessage
}

func (a Action) MarshalJSON() ([]byte, error) {
	return ironicrest.MarshalFlat(map[string]any{"action": a.Action}, a.Params)
}

func (a *Action) UnmarshalJSON(data []byte) error {
	var v struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	params, err := ironicrest.SplitExtra(data, "action")
	if err != nil {
		return err
	}
	*a = Action{Action: v.Action, Params: params}
	return nil
}

// Rule is an ironic-inspector introspection rule. Metadata.Body keeps every
// returned member. List rows carry only uuid, description, scope and links, so
// Conditions and Actions are nil for them.
type Rule struct {
	resource.Metadata
	UUID        string      `json:"uuid"`
	Description *string     `json:"description"`
	Scope       *string     `json:"scope"`
	Conditions  []Condition `json:"conditions"`
	Actions     []Action    `json:"actions"`
}

func (r *Rule) UnmarshalJSON(data []byte) error {
	type plain Rule
	var v plain
	if err := resource.DecodeObject(data, &v, &v.Metadata); err != nil {
		return err
	}
	*r = Rule(v)
	return nil
}

func decode(raw json.RawMessage, resp *ironicrest.Response) (*Rule, error) {
	var v Rule
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	v.Header, v.StatusCode = resp.Header, resp.Status
	return &v, nil
}
