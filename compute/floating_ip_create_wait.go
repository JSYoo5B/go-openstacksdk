package compute

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var floatingIPCreateWaitDeadline = errors.New("SDK floating IP create wait deadline")

func (p *floatingIPQueryState) finishNeutronCreate(result *CreateFloatingIPResult, policy FloatingIPCreateOpts) (*CreateFloatingIPResult, error) {
	if result.Selection.PortID == "" {
		return result, p.state.check(p.ctx)
	}
	completionCtx, waitID := p.ctx, ""
	if policy.Wait {
		id, err := floatingIPExecutableID(result.Allocation)
		if err != nil {
			return result, createResponseError(result.AllocationResponse, err)
		}
		waitCtx, cancel := context.WithTimeoutCause(p.ctx, policy.WaitTimeout, floatingIPCreateWaitDeadline)
		defer cancel()
		completionCtx, waitID = waitCtx, id
		child := *p
		child.ctx = waitCtx
		interval := policy.WaitInterval
		if interval == 0 {
			interval = min(5*time.Second, policy.WaitTimeout)
		}
		var waitErr error
		for {
			if waitCtx.Err() != nil {
				waitErr = errors.Join(waitCtx.Err(), context.Cause(waitCtx))
				break
			}
			observation, err := child.get(id)
			if observation != nil {
				result.Observations = append(result.Observations, observation)
			}
			if err != nil {
				waitErr = err
				if observation != nil {
					result.Failure = observation.Failure
				}
				break
			}
			if observation != nil && observation.FloatingIP != nil {
				var status string
				_ = json.Unmarshal(observation.FloatingIP.Resource.Body["status"], &status)
				if status == "ACTIVE" {
					if err := child.state.check(waitCtx); err != nil {
						waitErr = err
						break
					}
					result.FloatingIP = observation.FloatingIP
					break
				}
			}
			timer := time.NewTimer(interval)
			select {
			case <-waitCtx.Done():
				waitErr = errors.Join(waitCtx.Err(), context.Cause(waitCtx))
			case <-timer.C:
			}
			timer.Stop()
			if waitErr != nil {
				break
			}
		}
		if waitErr != nil {
			return p.failedCreateWait(result, policy, id, waitCtx, waitErr)
		}
	}
	var port string
	if result.FloatingIP == nil || result.FloatingIP.Resource == nil || json.Unmarshal(result.FloatingIP.Resource.Body["port_id"], &port) != nil || port != result.Selection.PortID {
		if policy.Wait {
			return result, errors.Join(invalid("observed floating IP is not on selected port %q", result.Selection.PortID), p.state.check(p.ctx))
		}
		return result, createResponseError(result.AllocationResponse, invalid("created floating IP is not on selected port %q", result.Selection.PortID))
	}
	if err := p.state.check(completionCtx); err != nil {
		if policy.Wait {
			return p.failedCreateWait(result, policy, waitID, completionCtx, err)
		}
		return result, err
	}
	result.Waited = policy.Wait
	return result, nil
}

func (p *floatingIPQueryState) failedCreateWait(result *CreateFloatingIPResult, policy FloatingIPCreateOpts, id string, waitCtx context.Context, waitErr error) (*CreateFloatingIPResult, error) {
	if errors.Is(context.Cause(waitCtx), floatingIPCreateWaitDeadline) {
		var last *FloatingIPRecord
		if len(result.Observations) > 0 {
			last = result.Observations[len(result.Observations)-1].FloatingIP
		}
		waitErr = errors.Join(&FloatingIPCreateTimeoutError{ID: id, Timeout: policy.WaitTimeout, Last: last}, waitErr)
		if err := p.state.check(p.ctx); err == nil {
			result.Cleanup, result.CleanupError = p.deleteWithRetries(id, 1)
		} else {
			result.CleanupError = err
		}
	}
	return result, errors.Join(waitErr, result.CleanupError, p.state.check(p.ctx))
}
