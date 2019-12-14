package agents

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/golang/choriautil"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/golang/discovery"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/golang/rpcutil"
	"github.com/sirupsen/logrus"
)

// Provider is a Agent Provider capable of executing compiled mcollective compatible agents written in Go
type Provider struct {
	fw  *choria.Framework
	log *logrus.Entry
}

// Initialize configures the agent provider
func (p Provider) Initialize(fw *choria.Framework, log *logrus.Entry) {
	p.fw = fw
	p.log = fw.Logger("agents").WithFields(logrus.Fields{"provider": "golang"})
}

// Version reports the version for this provider
func (p *Provider) Version() string {
	return fmt.Sprintf("%s version %s", p.PluginName(), p.PluginVersion())
}

// RegisterAgents registers known ruby agents using a shimm agent
func (p *Provider) RegisterAgents(ctx context.Context, mgr server.AgentManager, connector choria.InstanceConnector, log *logrus.Entry) error {
	var agent agents.Agent

	agent, err := discovery.New(mgr)
	if err != nil {
		return err
	}

	err = mgr.RegisterAgent(ctx, "discovery", agent, connector)
	if err != nil {
		return err
	}

	agent, err = rpcutil.New(mgr)
	if err != nil {
		return err
	}

	err = mgr.RegisterAgent(ctx, "rpcutil", agent, connector)
	if err != nil {
		return err
	}

	agent, err = choriautil.New(mgr)
	if err != nil {
		return err
	}

	err = mgr.RegisterAgent(ctx, "choria_util", agent, connector)
	if err != nil {
		return err
	}

	return nil
}
