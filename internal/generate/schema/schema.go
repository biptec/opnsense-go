package schema

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type AttrData struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	Key       string `yaml:"key"`
	OmitEmpty bool   `yaml:"omitEmpty"`
}

func (a *AttrData) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Unmarshal to map first
	d := make(map[string]string)
	err := unmarshal(&d)
	if err != nil {
		panic(err)
	}

	// Build AttrData from map
	a.Name = d["name"]
	a.Type = d["type"]
	a.Key = d["key"]

	a.OmitEmpty = false
	if d["omitEmpty"] == "true" {
		a.OmitEmpty = true
	}

	// Remap types to correct Go code
	if d["type"] == "SelectedMap" {
		a.Type = "api.SelectedMap"
	} else if d["type"] == "SelectedMapList" {
		a.Type = "api.SelectedMapList"
	} else if d["type"] == "SelectedMapListNL" {
		a.Type = "api.SelectedMapListNL"
	}

	return nil
}

type EndpointData struct {
	Path   string `yaml:"path"`
	Method string `yaml:"method"`
}

type ResourceEndpoints struct {
	Create      *EndpointData `yaml:"create,omitempty"`
	Read        *EndpointData `yaml:"read,omitempty"`
	Update      *EndpointData `yaml:"update,omitempty"`
	Delete      *EndpointData `yaml:"delete,omitempty"`
	Search      *EndpointData `yaml:"search,omitempty"`
	Reconfigure *EndpointData `yaml:"reconfigure,omitempty"`
}

type ResourceData struct {
	Name      string            `yaml:"name"`
	Filename  string            `yaml:"filename"`
	Monad     string            `yaml:"monad"`
	Endpoints ResourceEndpoints `yaml:"endpoints"`
	ReadOnly  bool              `yaml:"readOnly"`

	GetByFilter bool `yaml:"getByFilter"`
	GetAll      bool `yaml:"getAll"`

	Attrs       []AttrData            `yaml:"attrs"`
	CustomTypes map[string][]AttrData `yaml:"customTypes"`
}

type RPCData struct {
	Name        string                `yaml:"name"`
	Filename    string                `yaml:"filename"`
	RPCCalls    []RPCCallData         `yaml:"rpc_calls"`
	CustomTypes map[string][]AttrData `yaml:"customTypes"`
}

type Parameter struct {
	Name             string `yaml:"name"`
	Key              string `yaml:"key"`
	Optional         bool   `yaml:"optional"`
	IsBodyParameter  bool   `yaml:"bodyParameter"`
	IsQueryParameter bool   `yaml:"queryParameter"`
	CustomType       string `yaml:"customType"`
}

// KeyOrName returns the wire/URL key the API expects for this parameter.
// Defaults to Name when Key is unset; lets schemas use a Go-safe variable
// name (e.g. keyType) while emitting the actual API key (e.g. ?type=).
func (p *Parameter) KeyOrName() string {
	if p.Key != "" {
		return p.Key
	}
	return p.Name
}

type RPCCallData struct {
	Name       string       `yaml:"name"`
	Endpoint   EndpointData `yaml:"endpoint"`
	Parameters []Parameter  `yaml:"params"`
	ResultType string       `yaml:"result_type"`
}

type ControllerData struct {
	Name      string         `yaml:"name"`
	Resources []ResourceData `yaml:"resources"`
	RPC       []RPCData      `yaml:"rpc"`
}

func validateEndpoint(name string, endpoint *EndpointData) error {
	if endpoint == nil {
		return fmt.Errorf("%s endpoint is required", name)
	}
	if endpoint.Path == "" || !strings.HasPrefix(endpoint.Path, "/") {
		return fmt.Errorf("%s endpoint path must start with /", name)
	}
	if endpoint.Method == "" || endpoint.Method != strings.ToUpper(endpoint.Method) {
		return fmt.Errorf("%s endpoint method must be explicit and uppercase", name)
	}
	for _, character := range endpoint.Method {
		if character < 'A' || character > 'Z' {
			return fmt.Errorf("%s endpoint method must contain only A-Z", name)
		}
	}
	return nil
}

func (c *ControllerData) Validate() error {
	for index := range c.Resources {
		resource := &c.Resources[index]
		prefix := fmt.Sprintf("resource %s", resource.Name)
		if err := validateEndpoint(prefix+" read", resource.Endpoints.Read); err != nil {
			return err
		}
		if resource.ReadOnly {
			if resource.Endpoints.Create != nil || resource.Endpoints.Update != nil || resource.Endpoints.Delete != nil {
				return fmt.Errorf("%s is read-only but declares a write endpoint", prefix)
			}
		} else {
			for name, endpoint := range map[string]*EndpointData{
				"create": resource.Endpoints.Create,
				"update": resource.Endpoints.Update,
				"delete": resource.Endpoints.Delete,
			} {
				if err := validateEndpoint(prefix+" "+name, endpoint); err != nil {
					return err
				}
			}
		}
		for name, endpoint := range map[string]*EndpointData{
			"search":      resource.Endpoints.Search,
			"reconfigure": resource.Endpoints.Reconfigure,
		} {
			if endpoint != nil {
				if err := validateEndpoint(prefix+" "+name, endpoint); err != nil {
					return err
				}
			}
		}
	}
	for _, rpc := range c.RPC {
		for index := range rpc.RPCCalls {
			call := &rpc.RPCCalls[index]
			if err := validateEndpoint(fmt.Sprintf("RPC %s%s", rpc.Name, call.Name), &call.Endpoint); err != nil {
				return err
			}
		}
	}
	return nil
}

func decodeController(yamlFile []byte) (*ControllerData, error) {
	c := &ControllerData{}
	decoder := yaml.NewDecoder(bytes.NewReader(yamlFile))
	decoder.KnownFields(true)
	if err := decoder.Decode(c); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func newController(file string) *ControllerData {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	c, err := decodeController(yamlFile)
	if err != nil {
		panic(err)
	}
	return c
}

func GetController(name string) *ControllerData {
	p := fmt.Sprintf("%s/%s.yml", relativePathToSchema, name)

	// Check if controller schema file exists
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}

	// Load controller schema
	return newController(p)
}

func GetControllerNames() []string {
	files, err := os.ReadDir(relativePathToSchema)
	if err != nil {
		panic(err)
	}

	var controllerNames []string
	for _, file := range files {
		// Skip directories and non-YAML entries (.DS_Store, README.md,
		// editor swap files, etc.). Without this guard, a single stray
		// file in schema/ makes the generator panic with an opaque
		// yaml.Unmarshal error.
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}
		controllerNames = append(
			controllerNames,
			// Get load controller name from schema file
			newController(fmt.Sprintf("%s/%s", relativePathToSchema, name)).Name,
		)
	}
	return controllerNames
}

var relativePathToSchema = "../../schema"
