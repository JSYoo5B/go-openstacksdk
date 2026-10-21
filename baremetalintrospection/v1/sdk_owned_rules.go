package v1

import "github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/rules"

// RulesAPI returns the SDK-owned ironic-inspector introspection rule API on the
// same authenticated client as the native introspection API.
func (s *Service) RulesAPI() *rules.API { return rules.New(s.client) }
