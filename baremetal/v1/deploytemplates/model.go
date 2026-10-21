package deploytemplates

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicrest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Step is one deploy template step. Args is sent exactly as given: a nil map becomes
// JSON null and an empty map becomes {}. Extra carries step members that this
// package does not model; it may not replace a typed member. Priority is
// always sent.
type Step struct {
	Interface string
	Step      string
	Args      map[string]json.RawMessage
	Priority  int
	Extra     map[string]json.RawMessage
}

func (s Step) MarshalJSON() ([]byte, error) {
	core := map[string]any{"interface": s.Interface, "step": s.Step, "args": s.Args, "priority": s.Priority}
	return ironicrest.MarshalFlat(core, s.Extra)
}

func (s *Step) UnmarshalJSON(data []byte) error {
	var v struct {
		Interface string                     `json:"interface"`
		Step      string                     `json:"step"`
		Args      map[string]json.RawMessage `json:"args"`
		Priority  int                        `json:"priority"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	extra, err := ironicrest.SplitExtra(data, "interface", "step", "args", "priority")
	if err != nil {
		return err
	}
	*s = Step{Interface: v.Interface, Step: v.Step, Args: v.Args, Priority: v.Priority, Extra: extra}
	return nil
}

// DeployTemplate is a bare metal deploy template. Metadata.Body keeps every member the service
// returned, including those this struct does not model, and tells omitted from
// null. A list row or a fields-limited read carries only the requested members.
type DeployTemplate struct {
	resource.Metadata
	UUID  string                     `json:"uuid"`
	Name  string                     `json:"name"`
	Extra map[string]json.RawMessage `json:"extra"`
	Steps []Step                     `json:"steps"`
}

func (r *DeployTemplate) UnmarshalJSON(data []byte) error {
	type plain DeployTemplate
	var v plain
	if err := resource.DecodeObject(data, &v, &v.Metadata); err != nil {
		return err
	}
	*r = DeployTemplate(v)
	return nil
}

func decode(raw json.RawMessage, resp *ironicrest.Response) (*DeployTemplate, error) {
	var v DeployTemplate
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	v.Header, v.StatusCode = resp.Header, resp.Status
	return &v, nil
}
