// These tests use source fixtures adapted from Gophercloud v2.15.0.
// Copyright 2012-2013 Rackspace, Inc.
// Copyright Gophercloud authors
// SPDX-License-Identifier: Apache-2.0
//
// Local modifications select upstream Get and URL declarations and create
// altered source variants to check generator behavior.
// Upstream scope (github.com/gophercloud/gophercloud/v2):
//   openstack/networking/v2/extensions/{taas,segments,vpnaas,fwaas_v2,layer3/floatingips,bgpvpns,bgp}
//   openstack/identity/v3/{policies,regions}
//   openstack/compute/v2/aggregates
// See the root THIRD_PARTY_NOTICES.md and licenses/gophercloud-v2.15.0-LICENSE.

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// identityExtendedSpec reports specs that use the name, list-query or
// literal-GET extensions instead of the original Name/name-query shape.
func identityExtendedSpec(spec identityCollectionSpec) bool {
	return spec.nameField != "" || spec.idOnly || spec.listNameQuery != "" || spec.noListQuery || spec.literalGet
}

// identityFixtureShape returns native-shaped model fields, member ID type,
// list arguments and ListOpts name tag for one audited spec.
func identityFixtureShape(spec identityCollectionSpec) (fields, idType, listArgs, nameTag string) {
	fields, idType, listArgs, nameTag = "ID string;Name string", "string", ",opts ListOptsBuilder", identityExpectedNameQuery(spec)
	if spec.parents != 0 {
		listArgs = ",parentID string,opts ListOptsBuilder"
	}
	if spec.rawListIterator != "" {
		listArgs = ",opts ListOpts"
	}
	switch {
	case spec.idOnly:
		fields = "ID string"
	case spec.nameField != "":
		fields = "ID string;" + spec.nameField + " string"
	}
	if spec.literalGet {
		fields = strings.Replace(fields, "ID string", "ID int", 1)
		idType = "int"
	}
	if spec.noListQuery {
		listArgs = ""
	}
	return fields, idType, listArgs, nameTag
}

func identityDefaultCollectionFixture(t *testing.T, spec identityCollectionSpec) (*types.Package, *collectionPlan) {
	t.Helper()
	fields, idType, _, nameTag := identityFixtureShape(spec)
	return identityCollectionFixture(t, spec, fields, idType, nameTag)
}

func identityExtendedSpecs(t *testing.T) []identityCollectionSpec {
	t.Helper()
	var specs []identityCollectionSpec
	for _, spec := range identityCollectionSpecs {
		if identityExtendedSpec(spec) {
			specs = append(specs, spec)
		}
	}
	if len(specs) != 7 {
		t.Fatal("extended identity inventory must remain explicit", len(specs))
	}
	return specs
}

func TestIdentityExtendedCollectionsRejectNativeShapeDrift(t *testing.T) {
	for _, spec := range identityExtendedSpecs(t) {
		t.Run(spec.path, func(t *testing.T) {
			fields, idType, _, nameTag := identityFixtureShape(spec)
			drifts := map[string]func() (identityCollectionSpec, string, string, string){
				"wrong-model": func() (identityCollectionSpec, string, string, string) {
					s := spec
					s.model = "Other"
					return s, fields, idType, nameTag
				},
				"unrelated-package": func() (identityCollectionSpec, string, string, string) {
					s := spec
					s.path = "identity/v3/credentials"
					return s, fields, idType, nameTag
				},
				"url-identity": func() (identityCollectionSpec, string, string, string) {
					return spec, strings.Replace(fields, "ID ", "SecretRef ", 1), idType, nameTag
				},
			}
			switch {
			case spec.idOnly:
				drifts["native-name-added"] = func() (identityCollectionSpec, string, string, string) {
					return spec, fields + ";Name string", idType, nameTag
				}
			case spec.nameField != "":
				drifts["native-name-added"] = func() (identityCollectionSpec, string, string, string) {
					return spec, fields + ";Name string", idType, nameTag
				}
				drifts["name-field-removed"] = func() (identityCollectionSpec, string, string, string) { return spec, "ID string", idType, nameTag }
				drifts["name-field-numeric"] = func() (identityCollectionSpec, string, string, string) {
					return spec, "ID string;" + spec.nameField + " int", idType, nameTag
				}
			default:
				drifts["missing-model-name"] = func() (identityCollectionSpec, string, string, string) {
					return spec, strings.Replace(fields, ";Name string", "", 1), idType, nameTag
				}
			}
			if spec.listNameQuery != "" {
				drifts["native-name-query-added"] = func() (identityCollectionSpec, string, string, string) { return spec, fields, idType, "name" }
			}
			if spec.literalGet {
				drifts["string-model-id"] = func() (identityCollectionSpec, string, string, string) {
					return spec, strings.Replace(fields, "ID int", "ID string", 1), idType, nameTag
				}
				drifts["string-request-id"] = func() (identityCollectionSpec, string, string, string) { return spec, fields, "string", nameTag }
			} else {
				drifts["numeric-request-id"] = func() (identityCollectionSpec, string, string, string) { return spec, fields, "int", nameTag }
			}
			for name, drift := range drifts {
				t.Run(name, func(t *testing.T) {
					altered, fields, idType, nameTag := drift()
					pkg, plan := identityCollectionFixture(t, altered, fields, idType, nameTag)
					if identityCollectionEnabled(pkg, plan, altered.parents) {
						t.Fatal("unaudited extended identity shape was enabled", name, altered, plan)
					}
				})
			}
		})
	}
	// A native list that gains options must be re-audited before forwarding.
	for _, spec := range identityExtendedSpecs(t) {
		if !spec.noListQuery {
			continue
		}
		withOptions := spec
		withOptions.noListQuery = false
		fields, idType, _, _ := identityFixtureShape(spec)
		pkg, plan := identityCollectionFixture(t, withOptions, fields, idType, "")
		withOptions.noListQuery = true
		if _, ok := identityCollectionContract(pkg, plan, 0); ok {
			t.Fatal("native list options bypassed the no-query audit", spec.path)
		}
	}
}

func TestIdentityExtendedMetadataRejectsOverlappingPolicies(t *testing.T) {
	base := identityCollectionSpec{path: "fixture/v1/things", model: "Thing", getter: "Get", lister: "List"}
	for name, mutate := range map[string]func(*identityCollectionSpec){
		"name-field-and-id-only":     func(s *identityCollectionSpec) { s.nameField, s.idOnly = "Address", true },
		"list-name-and-no-list":      func(s *identityCollectionSpec) { s.listNameQuery, s.noListQuery = "name", true },
		"list-name-and-no-name":      func(s *identityCollectionSpec) { s.listNameQuery, s.noNameQuery = "name", true },
		"list-name-and-id-only":      func(s *identityCollectionSpec) { s.listNameQuery, s.idOnly = "name", true },
		"literal-get-with-list":      func(s *identityCollectionSpec) { s.literalGet = true },
		"extended-with-parent":       func(s *identityCollectionSpec) { s.idOnly, s.parents = true, 1 },
		"extended-with-raw-iterator": func(s *identityCollectionSpec) { s.noListQuery, s.rawListIterator = true, "IterateThings" },
		"extended-with-extra-specs":  func(s *identityCollectionSpec) { s.idOnly, s.extraSpecs = true, true },
		"extended-with-missing-list": func(s *identityCollectionSpec) { s.idOnly, s.missingListKey = true, "hidden" },
	} {
		spec := base
		mutate(&spec)
		if identityModelMetadataValid(spec) {
			t.Fatal("overlapping identity policies were accepted", name, spec)
		}
	}
	for _, spec := range identityCollectionSpecs {
		if !identityModelMetadataValid(spec) {
			t.Fatal("audited spec rejected", spec)
		}
	}
}

func TestIdentityExtendedEmissionCarriesNamePolicyAndLiteralGet(t *testing.T) {
	for _, spec := range identityCollectionSpecs {
		t.Run(spec.path, func(t *testing.T) {
			pkg, plan := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
			if !identityCollectionEnabled(pkg, plan, spec.parents) {
				t.Fatal("native-shaped fixture unexpectedly disabled", spec)
			}
			receiver, parents := "a", []string(nil)
			if spec.parents != 0 {
				receiver, parents = "s.api", []string{"s.parentID"}
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", spec.model)
			emitCollectionAdapter(&e, plan, receiver, parents)
			e.printf("}\n")
			source, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			body := strings.Join(strings.Fields(string(source)), " ")
			wants, forbids := []string{}, []string{}
			switch {
			case spec.nameField != "":
				wants = append(wants, "IdentityName: func(v *"+spec.model+") string { return fmt.Sprint(v."+spec.nameField+") }")
				forbids = append(forbids, " Name: func", "NameQuery:")
			case spec.idOnly:
				wants = append(wants, "IdentityName: func(*"+spec.model+") string { return \"\" }")
				forbids = append(forbids, " Name: func", "NameQuery:")
			default:
				forbids = append(forbids, "IdentityName:")
			}
			if spec.listNameQuery != "" {
				wants = append(wants, "NameQuery: func(name string) string { return name }", "NameQueryKey: "+fmt.Sprintf("%q", spec.listNameQuery))
			} else {
				forbids = append(forbids, "NameQueryKey:")
			}
			if spec.noListQuery {
				wants = append(wants, "if len(q) != 0 {", "yield(nil, resource.ErrUnsupported)")
				forbids = append(forbids, "config.Query[key]", "NameQuery:")
			}
			if spec.literalGet {
				wants = append(wants, "IdentityLiteralGet: true", "request.NumericID[int](id)", "[]string{\"os-aggregates\", id}, q, []int{200}")
			} else {
				forbids = append(forbids, "IdentityLiteralGet:")
			}
			for _, want := range wants {
				if !strings.Contains(body, want) {
					t.Fatalf("missing %q:\n%s", want, body)
				}
			}
			for _, forbid := range forbids {
				if strings.Contains(body, forbid) {
					t.Fatalf("unexpected %q:\n%s", forbid, body)
				}
			}
		})
	}
}

// pinnedExtendedIdentityDeclarations reproduces the pinned upstream member
// GET and URL helpers for the extended identity bindings.
func pinnedExtendedIdentityDeclarations(t *testing.T, spec identityCollectionSpec) map[string]*ast.FuncDecl {
	t.Helper()
	neutronGet := `func Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := c.Get(ctx, resourceURL(c, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`
	baseGet := strings.Replace(neutronGet, "resourceURL(c, id)", "getURL(c, id)", 1)
	var source string
	switch spec.path {
	case "network/v2/extensions/taas/tapmirrors", "network/v2/extensions/vpnaas/services", "network/v2/extensions/vpnaas/endpointgroups", "network/v2/extensions/vpnaas/ikepolicies", "network/v2/extensions/vpnaas/ipsecpolicies", "network/v2/extensions/vpnaas/siteconnections", "network/v2/extensions/fwaas_v2/rules", "network/v2/extensions/fwaas_v2/policies", "network/v2/extensions/fwaas_v2/groups":
		source = neutronGet + `
func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(rootPath, resourcePath, id)
}`
	case "network/v2/extensions/segments":
		source = neutronGet + `
func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(urlBaase, id)
}`
	case "network/v2/extensions/layer3/floatingips":
		source = neutronGet + `
func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(resourcePath, id)
}`
	case "network/v2/extensions/bgpvpns", "network/v2/extensions/bgp/peers", "network/v2/extensions/bgp/speakers":
		source = baseGet + `
func getURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}
func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(urlBase, id)
}`
	case "identity/v3/policies":
		source = `func Get(ctx context.Context, client *gophercloud.ServiceClient, policyID string) (r GetResult) {
	resp, err := client.Get(ctx, getURL(client, policyID), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
func getURL(client *gophercloud.ServiceClient, policyID string) string {
	return client.ServiceURL(policyPath, policyID)
}`
	case "identity/v3/regions":
		source = `func Get(ctx context.Context, client *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := client.Get(ctx, getURL(client, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
func getURL(client *gophercloud.ServiceClient, regionID string) string {
	return client.ServiceURL("regions", regionID)
}`
	case "compute/v2/aggregates":
		source = `func Get(ctx context.Context, client *gophercloud.ServiceClient, aggregateID int) (r GetResult) {
	v := strconv.Itoa(aggregateID)
	resp, err := client.Get(ctx, aggregatesGetURL(client, v), &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200},
	})
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
func aggregatesGetURL(c *gophercloud.ServiceClient, aggregateID string) string {
	return c.ServiceURL("os-aggregates", aggregateID)
}`
	default:
		return nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), "native.go", "package fixture\n"+source, 0)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			decls[fn.Name.Name] = fn
		}
	}
	return decls
}
