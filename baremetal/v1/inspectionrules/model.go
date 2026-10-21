package inspectionrules

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicrest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Condition is one inspection rule condition. Args and Loop are sent as given
// (omitted when empty) so JSON arrays, objects and numbers stay exact. Extra
// holds members this package does not model and may not replace a typed one.
type Condition struct {
	Op       string
	Args     json.RawMessage
	Loop     json.RawMessage
	Multiple *string
	Extra    map[string]json.RawMessage
}

func (c Condition) MarshalJSON() ([]byte, error) {
	core := map[string]any{"op": c.Op}
	if len(c.Args) > 0 {
		core["args"] = c.Args
	}
	if len(c.Loop) > 0 {
		core["loop"] = c.Loop
	}
	if c.Multiple != nil {
		core["multiple"] = *c.Multiple
	}
	return ironicrest.MarshalFlat(core, c.Extra)
}

func (c *Condition) UnmarshalJSON(data []byte) error {
	var v struct {
		Op       string          `json:"op"`
		Args     json.RawMessage `json:"args"`
		Loop     json.RawMessage `json:"loop"`
		Multiple *string         `json:"multiple"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	extra, err := ironicrest.SplitExtra(data, "op", "args", "loop", "multiple")
	if err != nil {
		return err
	}
	*c = Condition{Op: v.Op, Args: v.Args, Loop: v.Loop, Multiple: v.Multiple, Extra: extra}
	return nil
}

// Action is one inspection rule action; see Condition for Args, Loop and Extra.
type Action struct {
	Op    string
	Args  json.RawMessage
	Loop  json.RawMessage
	Extra map[string]json.RawMessage
}

func (a Action) MarshalJSON() ([]byte, error) {
	core := map[string]any{"op": a.Op}
	if len(a.Args) > 0 {
		core["args"] = a.Args
	}
	if len(a.Loop) > 0 {
		core["loop"] = a.Loop
	}
	return ironicrest.MarshalFlat(core, a.Extra)
}

func (a *Action) UnmarshalJSON(data []byte) error {
	var v struct {
		Op   string          `json:"op"`
		Args json.RawMessage `json:"args"`
		Loop json.RawMessage `json:"loop"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	extra, err := ironicrest.SplitExtra(data, "op", "args", "loop")
	if err != nil {
		return err
	}
	*a = Action{Op: v.Op, Args: v.Args, Loop: v.Loop, Extra: extra}
	return nil
}

// InspectionRule is a bare metal inspection rule. Metadata.Body keeps every
// returned member and tells omitted from null; a list row or fields-limited
// read carries only the requested members.
type InspectionRule struct {
	resource.Metadata
	UUID        string      `json:"uuid"`
	Description *string     `json:"description"`
	Phase       *string     `json:"phase"`
	Priority    *int        `json:"priority"`
	Sensitive   *bool       `json:"sensitive"`
	Conditions  []Condition `json:"conditions"`
	Actions     []Action    `json:"actions"`
}

func (r *InspectionRule) UnmarshalJSON(data []byte) error {
	type plain InspectionRule
	var v plain
	if err := resource.DecodeObject(data, &v, &v.Metadata); err != nil {
		return err
	}
	*r = InspectionRule(v)
	return nil
}

func decode(raw json.RawMessage, resp *ironicrest.Response) (*InspectionRule, error) {
	var v InspectionRule
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	v.Header, v.StatusCode = resp.Header, resp.Status
	return &v, nil
}
