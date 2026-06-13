package sui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
)

func TestGrpcTargetFromNodeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		nodeURL string
		want    string
		wantErr string
	}{
		{
			name:    "http with explicit port",
			nodeURL: "http://localhost:9000",
			want:    "localhost:9000",
		},
		{
			name:    "http without port defaults to 9000",
			nodeURL: "http://example.com",
			want:    "example.com:9000",
		},
		{
			name:    "https without port defaults to 443",
			nodeURL: "https://example.com",
			want:    "example.com:443",
		},
		{
			name:    "ipv4 with explicit port",
			nodeURL: "http://1.2.3.4:8080",
			want:    "1.2.3.4:8080",
		},
		{
			name:    "ipv6 unbracketed host gets bracketed",
			nodeURL: "http://[::1]:9001",
			want:    "[::1]:9001",
		},
		{
			name:    "ipv6 without port defaults and stays bracketed",
			nodeURL: "https://[2001:db8::1]",
			want:    "[2001:db8::1]:443",
		},
		{
			name:    "scheme-only URL has no host",
			nodeURL: "http://",
			wantErr: "has no host",
		},
		{
			name:    "unparseable URL",
			nodeURL: "http://[::1",
			wantErr: "parse node URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := grpcTargetFromNodeURL(tt.nodeURL)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Empty(t, got)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewBindingsClientFromNodeURL(t *testing.T) {
	t.Parallel()

	t.Run("success with explicit token", func(t *testing.T) {
		t.Parallel()
		client, err := NewBindingsClientFromNodeURL(logger.Test(t), "http://localhost:9000", "my-token")
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("success with empty token uses default", func(t *testing.T) {
		t.Parallel()
		client, err := NewBindingsClientFromNodeURL(logger.Test(t), "http://localhost:9000", "")
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("error propagated from invalid node URL", func(t *testing.T) {
		t.Parallel()
		client, err := NewBindingsClientFromNodeURL(logger.Test(t), "http://", "my-token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no host")
		assert.Nil(t, client)
	})
}
