package runbooks

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicrest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Step is one runbook step. Args is sent exactly as given: a nil map becomes
// JSON null and an empty map becomes {}. Extra carries step members that this
// package does not model; it may not replace a typed member. Order is omitted
// when nil.
type Step struct {
	Interface string
	Step      string
	Args      map[string]json.RawMessage
	Order     *int
	Extra     map[string]json.RawMessage
}

func (s Step) MarshalJSON() ([]byte, error) {
	core := map[string]any{"interface": s.Interface, "step": s.Step, "args": s.Args}
	if s.Order != nil {
		core["order"] = *s.Order
	}
	return ironicrest.MarshalFlat(core, s.Extra)
}

func (s *Step) UnmarshalJSON(data []byte) error {
	var v struct {
		Interface string                     `json:"interface"`
		Step      string                     `json:"step"`
		Args      map[string]json.RawMessage `json:"args"`
		Order     *int                       `json:"order"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	extra, err := ironicrest.SplitExtra(data, "interface", "step", "args", "order")
	if err != nil {
		return err
	}
	*s = Step{Interface: v.Interface, Step: v.Step, Args: v.Args, Order: v.Order, Extra: extra}
	return nil
}

// Runbook is a bare metal runbook. Metadata.Body keeps every member the service
// returned, including those this struct does not model, and tells omitted from
// null. A list row or a fields-limited read carries only the requested members.
type Runbook struct {
	resource.Metadata
	UUID        string                     `json:"uuid"`
	Name        string                     `json:"name"`
	Description *string                    `json:"description"`
	Extra       map[string]json.RawMessage `json:"extra"`
	Steps       []Step                     `json:"steps"`
	Public      *bool                      `json:"public"`
	Owner       *string                    `json:"owner"`
	Traits      []string                   `json:"traits"`
}

func (r *Runbook) UnmarshalJSON(data []byte) error {
	type plain Runbook
	var v plain
	if err := resource.DecodeObject(data, &v, &v.Metadata); err != nil {
		return err
	}
	*r = Runbook(v)
	return nil
}

func decode(raw json.RawMessage, resp *ironicrest.Response) (*Runbook, error) {
	var v Runbook
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	v.Header, v.StatusCode = resp.Header, resp.Status
	return &v, nil
}
