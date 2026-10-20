package main

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"
)

// Glance Task waiting owns a stateful recreate path; it is not the generic image
// wait policy and does not infer capabilities for tasks in other services.
func glanceTaskWaitCollectionBinding(pkg *types.Package, plan *collectionPlan) (string, error) {
	if sdkPath(pkg.Path()) != "image/v2/tasks" {
		return "", nil
	}
	if plan == nil || plan.model == nil || plan.modelName != "Task" || plan.id != "ID" || plan.status != "Status" || plan.getter == nil || plan.lister == nil || plan.name != "" {
		return "", fmt.Errorf("Glance Task wait requires its ID-only Task status collection")
	}
	create, ok := pkg.Scope().Lookup("Create").(*types.Func)
	if !ok || create == nil {
		return "", fmt.Errorf("Glance Task wait requires native Task Create")
	}
	model := plan.model
	if pointer, ok := model.(*types.Pointer); ok {
		model = pointer.Elem()
	}
	structure, ok := model.Underlying().(*types.Struct)
	if !ok {
		return "", fmt.Errorf("Glance Task wait requires a struct Task model")
	}
	keys := map[string]string{"ID": "id", "Status": "status", "Type": "type", "Message": "message", "Input": "input", "Result": "result"}
	for i := 0; i < structure.NumFields(); i++ {
		name := structure.Field(i).Name()
		if key, known := keys[name]; known {
			actual, _, _ := strings.Cut(reflect.StructTag(structure.Tag(i)).Get("json"), ",")
			if actual != key {
				return "", fmt.Errorf("Glance Task wait requires canonical Task.%s JSON key %q", name, key)
			}
		}
	}
	for _, name := range []string{"ID", "Status", "Type", "Message"} {
		obj, _, _ := types.LookupFieldOrMethod(plan.model, true, nil, name)
		v, ok := obj.(*types.Var)
		if !ok || !v.IsField() {
			return "", fmt.Errorf("Glance Task wait requires Task.%s", name)
		}
		b, ok := v.Type().Underlying().(*types.Basic)
		if !ok || b.Info()&types.IsString == 0 {
			return "", fmt.Errorf("Glance Task wait requires string Task.%s", name)
		}
	}
	for _, name := range []string{"Input", "Result"} {
		obj, _, _ := types.LookupFieldOrMethod(plan.model, true, nil, name)
		v, ok := obj.(*types.Var)
		if !ok || !v.IsField() {
			return "", fmt.Errorf("Glance Task wait requires Task.%s", name)
		}
		m, ok := v.Type().Underlying().(*types.Map)
		if !ok || !isString(m.Key()) {
			return "", fmt.Errorf("Glance Task wait requires map[string] Task.%s", name)
		}
		element, ok := m.Elem().Underlying().(*types.Interface)
		if !ok || element.NumMethods() != 0 {
			return "", fmt.Errorf("Glance Task wait requires arbitrary JSON Task.%s values", name)
		}
	}
	return "WaitForTask", nil
}
