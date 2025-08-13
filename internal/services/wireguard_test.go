package services

import (
	"testing"

	"go.uber.org/zap"
)

func TestGenerateKeyPair(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewWireguardService(logger)

	privateKey, publicKey, err := service.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if privateKey == "" {
		t.Error("Private key is empty")
	}

	if publicKey == "" {
		t.Error("Public key is empty")
	}

	if privateKey == publicKey {
		t.Error("Private and public keys should be different")
	}

	// Test key lengths (base64 encoded 32-byte keys should be 44 characters)
	if len(privateKey) != 44 {
		t.Errorf("Private key length should be 44, got %d", len(privateKey))
	}

	if len(publicKey) != 44 {
		t.Errorf("Public key length should be 44, got %d", len(publicKey))
	}
}

func TestValidatePublicKey(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewWireguardService(logger)

	tests := []struct {
		name      string
		publicKey string
		wantErr   bool
	}{
		{
			name:      "valid key",
			publicKey: "abcdefghijklmnopqrstuvwxyz123456", // 32 bytes
			wantErr:   false,
		},
		{
			name:      "invalid base64",
			publicKey: "invalid-base64!@#",
			wantErr:   true,
		},
		{
			name:      "wrong length",
			publicKey: "dGVzdA==", // "test" in base64 (4 bytes)
			wantErr:   true,
		},
		{
			name:      "empty key",
			publicKey: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidatePublicKey(tt.publicKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePublicKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidIPAddress(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewWireguardService(logger)

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{
			name: "valid IPv4",
			ip:   "192.168.1.1",
			want: true,
		},
		{
			name: "valid IPv4 with CIDR",
			ip:   "10.0.0.1/24",
			want: true,
		},
		{
			name: "valid IPv6",
			ip:   "2001:db8::1",
			want: true,
		},
		{
			name: "invalid IP",
			ip:   "256.256.256.256",
			want: false,
		},
		{
			name: "empty string",
			ip:   "",
			want: false,
		},
		{
			name: "not an IP",
			ip:   "not-an-ip",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.IsValidIPAddress(tt.ip); got != tt.want {
				t.Errorf("IsValidIPAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}
