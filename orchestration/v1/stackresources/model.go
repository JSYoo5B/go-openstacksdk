package stackresources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ResourceIdentity retains the actual owning stack and the resource_name used
// by Heat's resource endpoints. Neither LogicalID nor PhysicalID is a route ID.
type ResourceIdentity struct {
	Stack stacks.StackIdentity
	Name  string
}

func (identity ResourceIdentity) Validate() error {
	if err := identity.Stack.Validate(); err != nil {
		return err
	}
	return validateName(identity.Name)
}

// ResourceView preserves the native resource fields and the owning stack from
// Heat's links. A nested list can contain resources from several owning stacks.
type ResourceView struct {
	Resource
	Owner      stacks.StackIdentity       `json:"-"`
	OwnerKnown bool                       `json:"-"`
	Detailed   bool                       `json:"-"`
	Body       map[string]json.RawMessage `json:"-"`
	Header     http.Header                `json:"-"`
	StatusCode int                        `json:"-"`
}

func decodeResource(raw json.RawMessage, header http.Header, status int, detailed bool) (*ResourceView, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("resource must be a JSON object")
	}
	var value Resource
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	view, err := viewResource(value, detailed)
	if err != nil {
		return nil, err
	}
	view.Body, view.Header, view.StatusCode = fields, header.Clone(), status
	return view, nil
}

func (value *ResourceView) Identity() (ResourceIdentity, error) {
	if value == nil {
		return ResourceIdentity{}, fmt.Errorf("%w: nil stack resource", resource.ErrInvalidOption)
	}
	if !value.OwnerKnown {
		return ResourceIdentity{}, fmt.Errorf("%w: resource owner is unknown; bind an explicit stack identity", resource.ErrUnsupported)
	}
	identity := ResourceIdentity{Stack: value.Owner, Name: value.Name}
	return identity, identity.Validate()
}

func validateName(name string) error {
	if err := resource.ID(name).Validate(); err != nil {
		return err
	}
	if strings.ContainsRune(name, '\x00') {
		return fmt.Errorf("%w: resource_name contains NUL", resource.ErrInvalidOption)
	}
	return nil
}

func viewResource(value Resource, detailed bool) (*ResourceView, error) {
	if err := validateName(value.Name); err != nil {
		return nil, fmt.Errorf("invalid resource response: %w", err)
	}
	view := &ResourceView{Resource: value, Detailed: detailed}
	for _, link := range value.Links {
		if link.Rel != "stack" && link.Rel != "self" {
			continue
		}
		owner, name, err := linkIdentity(link.Href, link.Rel)
		if err != nil {
			return nil, fmt.Errorf("invalid resource %s link: %w", link.Rel, err)
		}
		if link.Rel == "self" && name != value.Name {
			return nil, fmt.Errorf("resource self link name %q does not match resource_name %q", name, value.Name)
		}
		if view.OwnerKnown && view.Owner != owner {
			return nil, fmt.Errorf("resource links disagree about the owning stack")
		}
		view.Owner, view.OwnerKnown = owner, true
	}
	return view, nil
}

// Links are parsed only as identity evidence. The SDK never follows their host.
func linkIdentity(raw, relation string) (stacks.StackIdentity, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return stacks.StackIdentity{}, "", err
	}
	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	want := 3
	if relation == "self" {
		want = 5
	}
	start := len(parts) - want
	if start < 0 || parts[start] != "stacks" || (relation == "self" && parts[start+3] != "resources") {
		return stacks.StackIdentity{}, "", fmt.Errorf("link is not a Heat %s identity", relation)
	}
	name, err := url.PathUnescape(parts[start+1])
	if err != nil {
		return stacks.StackIdentity{}, "", err
	}
	id, err := url.PathUnescape(parts[start+2])
	if err != nil {
		return stacks.StackIdentity{}, "", err
	}
	owner := stacks.StackIdentity{Name: name, ID: id}
	if err := owner.Validate(); err != nil {
		return stacks.StackIdentity{}, "", err
	}
	resourceName := ""
	if relation == "self" {
		resourceName, err = url.PathUnescape(parts[start+4])
		if err == nil {
			err = validateName(resourceName)
		}
	}
	return owner, resourceName, err
}
