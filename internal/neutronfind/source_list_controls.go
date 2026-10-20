package neutronfind

import (
	"bytes"
	"context"
	"encoding/json"
	"iter"
	"math/big"
)

// ListWithSourceControls opts into raw Source max_items truthiness, deferred
// comparison and pre-constructor row limits. Existing List/Find stay unchanged.
func (e *Engine) ListWithSourceControls(ctx context.Context, policy Options) iter.Seq2[*Record, error] {
	selected := *e
	selected.profile.SourceListControls = true
	return selected.List(ctx, policy)
}

func (e *Engine) sourceCap(raw json.RawMessage) func(context.Context, int) (bool, error) {
	owned := bytes.Clone(raw)
	return func(ctx context.Context, count int) (bool, error) {
		if err := e.Check(ctx); err != nil {
			return false, err
		}
		truth, err := e.truthy(owned)
		if err != nil || !truth {
			return false, err
		}
		decoder := json.NewDecoder(bytes.NewReader(owned))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return false, err
		}
		var text string
		switch value := value.(type) {
		case bool:
			text = "1"
		case json.Number:
			text = value.String()
		default:
			return false, findInvalid("Source max_items comparison requires a numeric value on a nonempty page")
		}
		cap, ok := new(big.Rat).SetString(text)
		if !ok {
			return false, findInvalid("invalid finite Source max_items")
		}
		return new(big.Rat).SetInt64(int64(count)).Cmp(cap) >= 0, nil
	}
}
