package api_extensions

import "testing"

func TestHAProxyTLSIdentityCRUDDefersReconfigureToTypedServiceRPC(t *testing.T) {
	if HAProxyTLSIdentityOpts.Reconfigure.Path != "" {
		t.Fatalf("HAProxyTLSIdentity CRUD must not implicitly reconfigure, got path %q", HAProxyTLSIdentityOpts.Reconfigure.Path)
	}
	if HAProxyTLSIdentityOpts.Reconfigure.Method != "" {
		t.Fatalf("HAProxyTLSIdentity CRUD must not implicitly reconfigure, got method %q", HAProxyTLSIdentityOpts.Reconfigure.Method)
	}
}
