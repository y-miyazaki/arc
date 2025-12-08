package helpers

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

// mockClient is a mock AWS client for testing.
type mockClient struct {
	region string
}

func TestCreateRegionalClients(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	regions := []string{"us-east-1", "eu-west-1", "ap-northeast-1"}

	factory := func(cfg *aws.Config, region string) *mockClient {
		return &mockClient{region: region}
	}

	clients, err := CreateRegionalClients(cfg, regions, factory)

	assert.NoError(t, err)
	assert.Len(t, clients, 3)
	assert.Equal(t, "us-east-1", clients["us-east-1"].region)
	assert.Equal(t, "eu-west-1", clients["eu-west-1"].region)
	assert.Equal(t, "ap-northeast-1", clients["ap-northeast-1"].region)
}

func TestCreateRegionalClients_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	factory := func(cfg *aws.Config, region string) *mockClient {
		return &mockClient{region: region}
	}

	clients, err := CreateRegionalClients(cfg, []string{}, factory)

	assert.NoError(t, err)
	assert.Len(t, clients, 0)
}
