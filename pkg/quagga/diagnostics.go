package quagga

import (
	"context"

	"github.com/biptec/opnsense-go/pkg/api"
)

type OSPFDiagnosticsResponse struct {
	Response map[string]any `json:"response"`
}

func callOSPFDiagnostic[T any](c *Controller, ctx context.Context, path, method string, result *T) (*T, error) {
	return api.Call(c.Client(), ctx, api.RPCOpts{
		Endpoint: api.Endpoint{Path: path, Method: method},
	}, result)
}

func (c *Controller) OSPFOverview(ctx context.Context) (*OSPFDiagnosticsResponse, error) {
	return callOSPFDiagnostic(c, ctx, "/quagga/diagnostics/ospfoverview", "GET", &OSPFDiagnosticsResponse{})
}

func (c *Controller) SearchOSPFNeighbors(ctx context.Context) (*api.SearchResponse[map[string]any], error) {
	return callOSPFDiagnostic(c, ctx, "/quagga/diagnostics/searchOspfneighbor", "POST", &api.SearchResponse[map[string]any]{})
}

func (c *Controller) SearchOSPFRoutes(ctx context.Context) (*api.SearchResponse[map[string]any], error) {
	return callOSPFDiagnostic(c, ctx, "/quagga/diagnostics/searchOspfroute", "POST", &api.SearchResponse[map[string]any]{})
}

func (c *Controller) OSPFDatabase(ctx context.Context) (*OSPFDiagnosticsResponse, error) {
	return callOSPFDiagnostic(c, ctx, "/quagga/diagnostics/ospfdatabase", "GET", &OSPFDiagnosticsResponse{})
}

func (c *Controller) OSPFInterface(ctx context.Context) (*OSPFDiagnosticsResponse, error) {
	return callOSPFDiagnostic(c, ctx, "/quagga/diagnostics/ospfinterface", "GET", &OSPFDiagnosticsResponse{})
}

func (c *Controller) OSPFv3Overview(ctx context.Context) (*OSPFDiagnosticsResponse, error) {
	return callOSPFDiagnostic(c, ctx, "/quagga/diagnostics/ospfv3overview", "GET", &OSPFDiagnosticsResponse{})
}

func (c *Controller) SearchOSPFv3Routes(ctx context.Context) (*api.SearchResponse[map[string]any], error) {
	return callOSPFDiagnostic(c, ctx, "/quagga/diagnostics/searchOspfv3route", "POST", &api.SearchResponse[map[string]any]{})
}

func (c *Controller) OSPFv3Database(ctx context.Context) (*api.SearchResponse[map[string]any], error) {
	return callOSPFDiagnostic(c, ctx, "/quagga/diagnostics/searchOspfv3database", "POST", &api.SearchResponse[map[string]any]{})
}

func (c *Controller) OSPFv3Interface(ctx context.Context) (*OSPFDiagnosticsResponse, error) {
	return callOSPFDiagnostic(c, ctx, "/quagga/diagnostics/ospfv3interface", "GET", &OSPFDiagnosticsResponse{})
}
