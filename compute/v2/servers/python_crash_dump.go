package servers

import (
	"context"
	"net/http"
)

// TriggerServerCrashDump requests a guest crash dump through Nova's action
// endpoint. Successful submission does not wait for the guest dump or reboot.
// The action owns its literal null payload; only transport header options apply.
// The pinned Source action does not impose a local 2.17 microversion gate, so
// the SDK leaves action availability to the server. The existing Go action
// profile validates the explicitly selected Nova version without discovery.
func (a *API) TriggerServerCrashDump(ctx context.Context, serverID string, options ...UserActionOption) error {
	const op = "TriggerServerCrashDump"
	prepared, err := a.prepareUserAction(ctx, serverID)
	if err != nil {
		return userActionWrap(ctx, op, err)
	}
	_, err = applyUserActionOptions(prepared, UserActionOpts{}, func(value UserActionOpts) UserActionOpts { return value }, false, options)
	if err == nil {
		err = prepared.version("2.1", "")
	}
	if err == nil {
		err = prepared.sendWithFaultPolicy(http.MethodPost, "action", map[string]any{"trigger_crash_dump": nil}, true)
	}
	return userActionWrap(ctx, op, err)
}
