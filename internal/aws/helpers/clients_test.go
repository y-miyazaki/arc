package helpers

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient is a mock AWS client for testing.
type mockClient struct {
	region string
}

func TestCreateRegionalClients(t *testing.T) {
	t.Parallel()

	cfg := &aws.Config{
		Region: "us-east-1",
	}
	factory := func(_ *aws.Config, region string) *mockClient {
		return &mockClient{region: region}
	}

	tests := []struct {
		name    string
		regions []string
		wantLen int
	}{
		{name: "creates a client for each region", regions: []string{"us-east-1", "eu-west-1", "ap-northeast-1"}, wantLen: 3},
		{name: "empty regions", regions: []string{}, wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clients, err := CreateRegionalClients(cfg, tt.regions, factory)
			require.NoError(t, err)
			assert.Len(t, clients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Equal(t, region, clients[region].region)
			}
		})
	}
}
