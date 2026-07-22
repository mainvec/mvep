package client

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
		errMsg  string
	}{

		{
			name:    "empty base URL",
			config:  ClientConfig{},
			wantErr: true,
			errMsg:  "base URL is required",
		},
		{
			name: "valid config with defaults",
			config: ClientConfig{
				BaseURL: "http://localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "valid config with all options",
			config: ClientConfig{
				BaseURL:  "http://localhost:8080",
				BasePath: "/api",
				Encoder:  "application/json",
				Timeout:  60 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error, got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("NewClient() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("NewClient() returned nil client")
				return
			}

			// Check defaults are applied
			if tt.config.Encoder == "" && client.config.Encoder != DefaultEncoder {
				t.Errorf("Expected default encoder %s, got %s", DefaultEncoder, client.config.Encoder)
			}

			if tt.config.Timeout == 0 && client.config.Timeout != 30*time.Second {
				t.Errorf("Expected default timeout 30s, got %v", client.config.Timeout)
			}
		})
	}
}

func TestRegisterPackage_NilPackage(t *testing.T) {
	client, err := NewClient(ClientConfig{
		BaseURL: "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.RegisterPackage(nil)
	if err == nil {
		t.Error("RegisterPackage(nil) expected error, got nil")
	}
}

func TestSendCmd_NilCommand(t *testing.T) {
	client, err := NewClient(ClientConfig{
		BaseURL: "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.SendCmd(context.Background(), nil)
	if err == nil {
		t.Error("SendCmd(nil) expected error, got nil")
	}
}

func TestSendCmd_NoPackageRegistered(t *testing.T) {
	client, err := NewClient(ClientConfig{
		BaseURL: "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Try to send a command without any registered packages
	_, err = client.SendCmd(context.Background(), struct{}{})
	if err == nil {
		t.Error("SendCmd without registered package expected error, got nil")
	}
}
