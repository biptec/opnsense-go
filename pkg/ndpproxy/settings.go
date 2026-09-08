package ndpproxy

import (
	"context"
	"fmt"

	"github.com/biptec/opnsense-go/pkg/api"
)

// GeneralSettings models the global os-ndp-proxy-go configuration used by routed endpoints.
type GeneralSettings struct {
	Enabled      string              `json:"enabled"`
	Upstream     api.SelectedMap     `json:"upstream"`
	Downstream   api.SelectedMapList `json:"downstream"`
	RA           string              `json:"ra"`
	Routes       string              `json:"routes"`
	CacheTTL     string              `json:"cache_ttl"`
	CacheMax     string              `json:"cache_max"`
	CacheFile    string              `json:"cache_file"`
	RouteQPS     string              `json:"route_qps"`
	PFQPS        string              `json:"pf_qps"`
	PcapTimeout  string              `json:"pcap_timeout"`
	Debug        string              `json:"debug"`
	CarpDependOn string              `json:"carp_depend_on"`
}

type Settings struct {
	General GeneralSettings `json:"general"`
}

type SettingsResponse struct {
	NdpProxy Settings `json:"ndpproxy"`
}

func (c *Controller) SettingsGet(ctx context.Context) (*SettingsResponse, error) {
	callOpts := api.RPCOpts{
		Endpoint:        api.Endpoint{Path: "/ndpproxy/general/get", Method: "GET"},
		PathParameters:  []string{},
		QueryParameters: map[string]string{},
		BodyParameters:  map[string]interface{}{},
	}
	resultData := &SettingsResponse{}
	result, err := api.Call(c.Client(), ctx, callOpts, resultData)
	if err != nil {
		return nil, fmt.Errorf("get call failed: %w", err)
	}
	return result, nil
}

func (c *Controller) SettingsSet(ctx context.Context, settings *Settings) (*api.ActionResult, error) {
	bodyParams := map[string]interface{}{"ndpproxy": settings}
	callOpts := api.RPCOpts{
		Endpoint:        api.Endpoint{Path: "/ndpproxy/general/set", Method: "POST"},
		PathParameters:  []string{},
		QueryParameters: map[string]string{},
		BodyParameters:  bodyParams,
	}
	resultData := &api.ActionResult{}
	result, err := api.Call(c.Client(), ctx, callOpts, resultData)
	if err != nil {
		return nil, fmt.Errorf("set call failed: %w", err)
	}
	return result, nil
}
