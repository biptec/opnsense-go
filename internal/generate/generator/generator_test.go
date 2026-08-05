package generator

import "testing"

func TestPascalIdentifier(t *testing.T) {
	tests := map[string]string{
		"api_extensions": "ApiExtensions",
		"openvpn":        "Openvpn",
		"wireguard":      "Wireguard",
		"dns-tools":      "DnsTools",
		"":               "",
	}

	for input, expected := range tests {
		if actual := pascalIdentifier(input); actual != expected {
			t.Errorf("pascalIdentifier(%q) = %q, want %q", input, actual, expected)
		}
	}
}
