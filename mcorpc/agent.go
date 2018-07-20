package mcorpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/audit"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/sirupsen/logrus"
)

// Action is a function that implements a RPC Action
type Action func(context.Context, *Request, *Reply, *Agent, choria.ConnectorInfo)

// Agent is an instance of the MCollective compatible RPC agents
type Agent struct {
	Log              *logrus.Entry
	Config           *config.Config
	Choria           *choria.Framework
	ServerInfoSource agents.ServerInfoSource

	meta    *agents.Metadata
	actions map[string]Action
}

// New creates a new MCollective SimpleRPC compatible agent
func New(name string, metadata *agents.Metadata, fw *choria.Framework, log *logrus.Entry) *Agent {
	a := &Agent{
		meta:    metadata,
		Log:     log.WithFields(logrus.Fields{"agent": name}),
		actions: make(map[string]Action),
		Choria:  fw,
		Config:  fw.Config,
	}

	return a
}

// SetServerInfo stores the server info source that owns this agent
func (a *Agent) SetServerInfo(si agents.ServerInfoSource) {
	a.ServerInfoSource = si
}

// RegisterAction registers an action into the agent
func (a *Agent) RegisterAction(name string, f Action) error {
	if _, ok := a.actions[name]; ok {
		return fmt.Errorf("Cannot register action %s, it already exist", name)
	}

	a.actions[name] = f

	return nil
}

// MustRegisterAction registers an action and panics if it fails
func (a *Agent) MustRegisterAction(name string, f Action) {
	if _, ok := a.actions[name]; ok {
		panic(fmt.Errorf("Cannot register action %s, it already exist", name))
	}

	a.actions[name] = f
}

// HandleMessage attempts to parse a choria.Message as a MCollective SimpleRPC request and calls
// the agents and actions associated with it
func (a *Agent) HandleMessage(ctx context.Context, msg *choria.Message, request protocol.Request, conn choria.ConnectorInfo, outbox chan *agents.AgentReply) {
	var err error

	reply := a.newReply()
	defer a.publish(reply, msg, request, outbox)

	rpcrequest, err := a.parseIncomingMessage(msg.Payload, request)
	if err != nil {
		reply.Statuscode = InvalidData
		reply.Statusmsg = fmt.Sprintf("Could not process request: %s", err)
		return
	}

	action, found := a.actions[rpcrequest.Action]
	if !found {
		reply.Statuscode = UnknownAction
		reply.Statusmsg = fmt.Sprintf("Unknown action %s for agent %s", rpcrequest.Action, a.Name())
		return
	}

	// TODO:
	//  authorize
	//  timeouts

	if a.Config.RPCAudit {
		audit.Request(request, rpcrequest.Agent, rpcrequest.Action, rpcrequest.Data, a.Config)
	}

	a.Log.Infof("Handling message %s for %s#%s from %s", msg.RequestID, a.Name(), rpcrequest.Action, request.CallerID())

	action(ctx, rpcrequest, reply, a, conn)
}

// Name retrieves the name of the agent
func (a *Agent) Name() string {
	return a.meta.Name
}

// ActionNames returns a list of known actions in the agent
func (a *Agent) ActionNames() []string {
	actions := []string{}

	for k := range a.actions {
		actions = append(actions, k)
	}

	sort.Strings(actions)

	return actions
}

// Metadata retrieves the agent metadata
func (a *Agent) Metadata() *agents.Metadata {
	return a.meta
}

func (a *Agent) publish(rpcreply *Reply, msg *choria.Message, request protocol.Request, outbox chan *agents.AgentReply) {
	if rpcreply.DisableResponse {
		return
	}

	reply := &agents.AgentReply{
		Message: msg,
		Request: request,
	}

	if rpcreply.Data == nil {
		rpcreply.Data = "{}"
	}

	j, err := json.Marshal(rpcreply)
	if err != nil {
		logrus.Errorf("Could not JSON encode reply: %s", err)
		reply.Error = err
	}

	reply.Body = j

	outbox <- reply
}

func (a *Agent) newReply() *Reply {
	reply := &Reply{
		Statuscode: OK,
		Statusmsg:  "OK",
		Data:       json.RawMessage(`{}`),
	}

	return reply
}

func (a *Agent) parseIncomingMessage(msg string, request protocol.Request) (*Request, error) {
	r := &Request{}

	err := json.Unmarshal([]byte(msg), r)
	if err != nil {
		return nil, fmt.Errorf("Could not parse incoming message as a MCollective SimpleRPC Request: %s", err)
	}

	r.CallerID = request.CallerID()
	r.RequestID = request.RequestID()
	r.SenderID = request.SenderID()
	r.Collective = request.Collective()
	r.TTL = request.TTL()
	r.Time = request.Time()
	r.Filter, _ = request.Filter()

	if r.Data == nil {
		r.Data = json.RawMessage(`{}`)
	}

	return r, nil
}
