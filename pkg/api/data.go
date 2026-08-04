package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type Endpoint struct {
	Path   string
	Method string
}

func (e Endpoint) Validate() error {
	if e.Path == "" || !strings.HasPrefix(e.Path, "/") {
		return fmt.Errorf("endpoint path must start with /: %q", e.Path)
	}
	if e.Method == "" || e.Method != strings.ToUpper(e.Method) {
		return fmt.Errorf("endpoint method must be explicit and uppercase: %q", e.Method)
	}
	for _, character := range e.Method {
		if character < 'A' || character > 'Z' {
			return fmt.Errorf("endpoint method must contain only A-Z: %q", e.Method)
		}
	}
	return nil
}

func (e Endpoint) WithPathSegment(segment string) Endpoint {
	e.Path = strings.TrimRight(e.Path, "/") + "/" + url.PathEscape(segment)
	return e
}

type ReqOpts struct {
	Create      Endpoint
	Read        Endpoint
	Update      Endpoint
	Delete      Endpoint
	Search      Endpoint
	Reconfigure Endpoint

	Monad string
}

// Response structs
type addResp struct {
	Result      string                 `json:"result"`
	UUID        string                 `json:"uuid"`
	Validations map[string]interface{} `json:"validations,omitempty"`
}

type deleteResp struct {
	Result string `json:"result"`
}

// RCP Options
type RPCOpts struct {
	Endpoint        Endpoint
	PathParameters  []string
	QueryParameters map[string]string
	BodyParameters  map[string]interface{}
}

func (p *RPCOpts) EndpointURL() string {
	currentPath := p.Endpoint.Path
	for _, param := range p.PathParameters {
		escapedParam := url.PathEscape(param)

		if currentPath == "" {
			currentPath = escapedParam
		} else if strings.HasSuffix(currentPath, "/") {
			currentPath += escapedParam
		} else {
			currentPath += "/" + escapedParam
		}
	}

	if len(p.QueryParameters) > 0 {
		keys := make([]string, 0, len(p.QueryParameters))
		for k := range p.QueryParameters {
			keys = append(keys, k)
		}
		// Sort so the URL is deterministic — important for any test that
		// asserts on the request URL and for caches/loggers that key on it.
		sort.Strings(keys)

		values := url.Values{}
		for _, k := range keys {
			values.Set(k, p.QueryParameters[k])
		}

		separator := "?"
		if strings.Contains(currentPath, "?") {
			separator = "&"
		}
		currentPath += separator + values.Encode()
	}
	return currentPath
}

func (p *RPCOpts) Body() (string, error) {
	if len(p.BodyParameters) == 0 {
		return "{}", nil
	}
	jsonBytes, err := json.Marshal(p.BodyParameters)
	if err != nil {
		return "", fmt.Errorf("failed to marshal BodyParameters to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

type ActionResult struct {
	Result string `json:"result"`
}

type ReconfigureStatusResult struct {
	Status string `json:"status"`
}
