package interfaces

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestVip(t *testing.T) {
	requireIntegration(t)
	opnsenseURL := os.Getenv("OPNSENSE_URI")
	opnsenseKey := os.Getenv("OPNSENSE_API_KEY")
	opnsenseSecret := os.Getenv("OPNSENSE_API_SECRET")

	apiClient := api.NewClient(api.Options{
		Uri:           opnsenseURL,
		APIKey:        opnsenseKey,
		APISecret:     opnsenseSecret,
		AllowInsecure: true,
		MaxBackoff:    30,
		MinBackoff:    1,
		MaxRetries:    4,
	})

	controller := Controller{Api: apiClient}
	ctx := context.Background()

	// new vip object
	vip := &Vip{
		Interface:         "wan",
		Mode:              "proxyarp",
		Network:           "192.168.0.195/32",
		Description:       "Test VIP",
		Gateway:           "",
		NoExpand:          "0",
		NoBind:            "0",
		AdvertisementBase: "1",
		AdvertisementSkew: "0",
		NoSync:            "0",
	}

	// CREATE
	key, err := controller.AddVip(ctx, vip)
	if err != nil {
		t.Fatalf("Failed to add VIP: %v", err)
	}
	t.Logf("Added VIP with key: %s", key)

	// READ
	got, err := controller.GetVip(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get VIP: %v", err)
	}
	if got.Interface != vip.Interface {
		t.Errorf("Interface mismatch: got %s, want %s", got.Interface, vip.Interface)
	}
	if got.Mode != vip.Mode {
		t.Errorf("Mode mismatch: got %s, want %s", got.Mode, vip.Mode)
	}
	if got.Network != vip.Network {
		t.Errorf("Network mismatch: got %s, want %s", got.Network, vip.Network)
	}
	if got.Description != vip.Description {
		t.Errorf("Description mismatch: got %s, want %s", got.Description, vip.Description)
	}
	if got.Gateway != vip.Gateway {
		t.Errorf("Gateway mismatch: got %s, want %s", got.Gateway, vip.Gateway)
	}

	// UPDATE
	vip.Network = "192.168.0.195/32" // change VIP
	vip.Description = "Updated VIP"
	err = controller.UpdateVip(ctx, key, vip)
	if err != nil {
		t.Fatalf("Failed to update VIP: %v", err)
	}

	// READ after update
	updated, err := controller.GetVip(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get updated VIP: %v", err)
	}
	if updated.Network != "192.168.0.195/32" {
		t.Errorf("Network mismatch after update: got %s, want %s", updated.Network, "192.168.0.195/32")
	}
	if updated.Description != "Updated VIP" {
		t.Errorf("Description mismatch after update: got %s, want %s", updated.Description, "Updated VIP")
	}

	// DELETE
	err = controller.DeleteVip(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete VIP: %v", err)
	}
	t.Logf("Deleted VIP with key: %s", key)
}

func TestVipVirtualMACJSONRoundTrip(t *testing.T) {
	input := Vip{VirtualMAC: "02:de:ad:be:ef:01"}
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(payload, []byte(`"vmac":"02:de:ad:be:ef:01"`)) {
		t.Fatalf("virtual MAC missing from JSON payload: %s", payload)
	}
	var output Vip
	if err := json.Unmarshal(payload, &output); err != nil {
		t.Fatal(err)
	}
	if output.VirtualMAC != input.VirtualMAC {
		t.Fatalf("virtual MAC round trip mismatch: got %q want %q", output.VirtualMAC, input.VirtualMAC)
	}
}
