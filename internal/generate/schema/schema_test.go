package schema

import (
	"path/filepath"
	"testing"
)

func TestAllRepositorySchemasUseEndpointObjects(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "..", "schema", "*.yml"))
	if err != nil {
		t.Fatalf("glob schemas: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no schema files found")
	}

	for _, file := range files {
		file := file
		t.Run(filepath.Base(file), func(t *testing.T) {
			t.Parallel()
			controller := newController(file)
			if controller.Name == "" {
				t.Fatal("controller name is empty")
			}
		})
	}
}

func TestLegacyEndpointSyntaxIsRejected(t *testing.T) {
	tests := map[string]string{
		"scalar CRUD endpoints": `
name: test
resources:
  - name: Item
    filename: item
    monad: item
    endpoints:
      add: "/item/add"
      get: "/item/get"
      update: "/item/set"
      delete: "/item/del"
`,
		"getMethod exception": `
name: test
resources:
  - name: Item
    filename: item
    monad: item
    endpoints:
      create: {path: "/item/add", method: POST}
      read: {path: "/item/get", method: GET}
      getMethod: POST
      update: {path: "/item/set", method: POST}
      delete: {path: "/item/del", method: POST}
`,
		"split RPC endpoint and method": `
name: test
rpc:
  - name: Service
    filename: service
    rpc_calls:
      - name: Status
        endpoint: "/service/status"
        method: GET
        result_type: api.ActionResult
`,
		"missing explicit method": `
name: test
resources:
  - name: Item
    filename: item
    monad: item
    endpoints:
      create: {path: "/item/add", method: POST}
      read: {path: "/item/get"}
      update: {path: "/item/set", method: POST}
      delete: {path: "/item/del", method: POST}
`,
	}

	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := decodeController([]byte(source))
			if err == nil {
				t.Fatal("legacy endpoint syntax was accepted")
			}
		})
	}
}
