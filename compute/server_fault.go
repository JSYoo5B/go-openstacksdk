package compute

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ServerFaultError reports a server in ERROR with the two message forms used
// by Python's cloud create_server and get_active_server. It unwraps to
// resource.FailedStateError, so errors.Is(err, resource.ErrFailedState)
// remains true. Server is the observed server, equivalent to Python's
// extra_data={"server": server}.
//
// HasFaultMessage follows Python's key-presence test: the fault value is a
// non-null object containing a "message" key. For raw Nova reads the SDK
// inspects the actual JSON; for a supplied typed Server, which cannot record
// key presence, a non-empty Fault.Message is treated as present.
type ServerFaultError struct {
	ServerID        string
	Status          string
	Server          *Server
	FaultMessage    string
	HasFaultMessage bool
}

func (e *ServerFaultError) Error() string {
	if e.HasFaultMessage {
		return fmt.Sprintf("Error in creating the server. Compute service reports fault: %s (server %q status %s)", e.FaultMessage, e.ServerID, e.Status)
	}
	return fmt.Sprintf("Error in creating the server (no further information available) (server %q status %s)", e.ServerID, e.Status)
}

func (e *ServerFaultError) Unwrap() error {
	return &resource.FailedStateError{Resource: "server", ID: e.ServerID, Status: e.Status}
}

type serverFaultInfo struct {
	message string
	present bool
}

// rawServerFault applies Python's presence test to one raw server object.
// A JSON string message is used verbatim and null formats as Python's None.
// Other message types already fail the typed Server decode before this check.
func rawServerFault(body json.RawMessage) serverFaultInfo {
	var envelope struct {
		Fault json.RawMessage `json:"fault"`
	}
	if json.Unmarshal(body, &envelope) != nil || len(envelope.Fault) == 0 || string(envelope.Fault) == "null" {
		return serverFaultInfo{}
	}
	var fault map[string]json.RawMessage
	if json.Unmarshal(envelope.Fault, &fault) != nil {
		return serverFaultInfo{}
	}
	raw, ok := fault["message"]
	if !ok {
		return serverFaultInfo{}
	}
	if string(raw) == "null" {
		return serverFaultInfo{message: "None", present: true}
	}
	var text string
	_ = json.Unmarshal(raw, &text)
	return serverFaultInfo{message: text, present: true}
}

func newServerFaultError(server *Server, info *serverFaultInfo) *ServerFaultError {
	err := &ServerFaultError{Server: server}
	if server != nil {
		err.ServerID, err.Status = server.ID, server.Status
	}
	switch {
	case info != nil:
		err.FaultMessage, err.HasFaultMessage = info.message, info.present
	case server != nil && server.Fault.Message != "":
		err.FaultMessage, err.HasFaultMessage = server.Fault.Message, true
	}
	return err
}
