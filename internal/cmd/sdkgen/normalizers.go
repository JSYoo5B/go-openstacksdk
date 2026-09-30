package main

import "go/types"

// These responses need library-owned handling rather than choosing one of
// several upstream extractors or guessing defaults for extractor parameters.
type resultNormalizer struct {
	resultType string
	valueType  string
	zero       string
	call       string
	options    string
	prepare    string
}

var resultNormalizers = map[string]resultNormalizer{
	upstreamModule + "/openstack/identity/v2/tokens.Create": {
		resultType: "CreateResult", valueType: "*Authentication", zero: "nil",
		call: "extractAuthentication(result.Result)",
	},
	upstreamModule + "/openstack/identity/v2/tokens.Get": {
		resultType: "GetResult", valueType: "*Authentication", zero: "nil",
		call: "extractAuthentication(result.Result)",
	},
	upstreamModule + "/openstack/compute/v2/servers.GetPassword": {
		resultType: "GetPasswordResult", valueType: "string", zero: `""`,
		call: "result.ExtractPassword(password.privateKey)", options: "GetPasswordOption",
		prepare: "password,err:=applyGetPasswordOptions(options...)",
	},
}

func normalizerFor(fn *types.Func) *resultNormalizer {
	if fn.Pkg() == nil {
		return nil
	}
	normalizer, ok := resultNormalizers[fn.Pkg().Path()+"."+fn.Name()]
	if !ok {
		return nil
	}
	sig := fn.Type().(*types.Signature)
	if sig.Results().Len() != 1 {
		return nil
	}
	named, ok := sig.Results().At(0).Type().(*types.Named)
	if !ok || named.Obj().Pkg() != fn.Pkg() || named.Obj().Name() != normalizer.resultType {
		return nil
	}
	return &normalizer
}
