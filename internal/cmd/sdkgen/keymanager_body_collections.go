package main

import (
	"go/types"
	"reflect"
)

func rawBodyFieldsMatch(value types.Type, tag string, wanted map[string][2]string) bool {
	if value == nil {
		return false
	}
	fields, ok := value.Underlying().(*types.Struct)
	if !ok || fields.NumFields() != len(wanted) {
		return false
	}
	for i := 0; i < fields.NumFields(); i++ {
		field := fields.Field(i)
		want, known := wanted[field.Name()]
		if !known || field.Embedded() || types.TypeString(field.Type(), func(p *types.Package) string { return p.Path() }) != want[0] || reflect.StructTag(fields.Tag(i)).Get(tag) != want[1] {
			return false
		}
	}
	return true
}

func rawBodyNativeDecoder(model *types.Named) bool {
	if model == nil || model.NumMethods() != 1 {
		return false
	}
	decoder, _, _ := types.LookupFieldOrMethod(types.NewPointer(model), true, nil, "UnmarshalJSON")
	decode, ok := decoder.(*types.Func)
	if !ok {
		return false
	}
	sig := decode.Type().(*types.Signature)
	return !sig.Variadic() && sig.Params().Len() == 1 && types.Identical(sig.Params().At(0).Type(), types.NewSlice(types.Typ[types.Uint8])) && sig.Results().Len() == 1 && isError(sig.Results().At(0).Type())
}

func rawBodyNativePage(pkg *types.Package, model types.Type, pageName, extractorName string) bool {
	object := pkg.Scope().Lookup(pageName)
	if object == nil {
		return false
	}
	page, ok := object.Type().(*types.Named)
	if !ok || page.NumMethods() != 2 {
		return false
	}
	fields, ok := page.Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 1 || !fields.Field(0).Embedded() || types.TypeString(fields.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.LinkedPageBase" {
		return false
	}
	for name, result := range map[string]types.Type{"IsEmpty": types.Typ[types.Bool], "NextPageURL": types.Typ[types.String]} {
		method := extractionMethod(page, name)
		own := false
		for i := 0; i < page.NumMethods(); i++ {
			own = own || page.Method(i).Name() == name
		}
		if !own || method == nil || !types.Identical(method.Results().At(0).Type(), result) {
			return false
		}
	}
	extractor, ok := pkg.Scope().Lookup(extractorName).(*types.Func)
	if !ok {
		return false
	}
	sig := extractor.Type().(*types.Signature)
	return !sig.Variadic() && sig.Params().Len() == 1 && types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) == upstreamModule+"/pagination.Page" && sig.Results().Len() == 2 && types.Identical(sig.Results().At(0).Type(), types.NewSlice(model)) && isError(sig.Results().At(1).Type())
}

// Secret and Container use the same native builder and raw-record bridge.
// Their strict schemas, source hashes and selectors remain resource-specific.
func emitKeyManagerBodyRecordAdapter(e *emitter, plan *collectionPlan, selector string) {
	e.use("github.com/JSYoo5B/gophercloudsdk/request")
	e.use("maps")
	e.printf("},\nBodyFilterRecordValue:func(record *resource.BodyRecord[%s],key string)(json.RawMessage,error){return %s(record,key)},\n", plan.modelName, selector)
	e.printf("IterateBodyControlled:func(ctx context.Context,q url.Values,control resource.ListControl)iter.Seq2[*resource.BodyRecord[%s],error]{\n", plan.modelName)
	e.printf("q=maps.Clone(q)\nq.Del(\"status\")\n")
	e.printf("options:=[]ListOption{func(config *request.Config[ListOpts])error{config.Query=make(url.Values,len(q));for key,values:=range q{config.Query[key]=append([]string(nil),values...)};return nil}}\nreturn a.listBodyWithControl(ctx,control,options...)\n},\n")
}

func emitKeyManagerBodyFilterList(e *emitter, plan *collectionPlan, envelope, extractor string) {
	e.use("github.com/JSYoo5B/gophercloudsdk/request")
	e.use(upstreamModule + "/pagination")
	e.use(e.pkg.Path())
	e.printf("func(a *API)listBodyWithControl(ctx context.Context,control resource.ListControl,options ...ListOption)iter.Seq2[*resource.BodyRecord[%s],error]{\nvar opts ListOpts\ncfg,err:=request.Apply(opts,options...)\n", plan.modelName)
	e.printf("if err==nil{err=request.ValidateCapabilities(cfg,false,true,false)}\nif err!=nil{err=request.Wrap(\"List\",%q,err);return func(yield func(*resource.BodyRecord[%s],error)bool){yield(nil,err)}}\n", envelope, plan.modelName)
	e.printf("_opts:=listOptsBuilder{base:cfg.Options,config:cfg}\nreturn resource.BodyStreamWithControl(ctx,upstream.List(a.client,_opts),func(page pagination.Page)([]%s,error){values,err:=upstream.%s(page);return []%s(values),err},%q,control)\n}\n", plan.modelName, extractor, plan.modelName, envelope)
}
