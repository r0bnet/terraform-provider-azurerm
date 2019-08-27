package powerbi

import (
	"github.com/Azure/azure-sdk-for-go/services/powerbidedicated/mgmt/2017-10-01/powerbidedicated"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/common"
)

type Client struct {
	CapacitiesClient *powerbidedicated.CapacitiesClient
}

func BuildClient(o *common.ClientOptions) *Client {

	CapacitiesClient := powerbidedicated.NewCapacitiesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&CapacitiesClient.Client, o.ResourceManagerAuthorizer)

	return &Client{
		CapacitiesClient: &CapacitiesClient,
	}
}
