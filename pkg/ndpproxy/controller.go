package ndpproxy

import "github.com/biptec/opnsense-go/pkg/api"

// Controller provides access to the OPNsense NDP Proxy Go API.
type Controller struct {
	Api *api.Client
}

func (c *Controller) Client() *api.Client {
	return c.Api
}
