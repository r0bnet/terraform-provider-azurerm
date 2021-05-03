package parse

// NOTE: this file is generated via 'go:generate' - manual changes will be overwritten

import (
	"fmt"
	"strings"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
)

type JobTargetGroupId struct {
	SubscriptionId  string
	ResourceGroup   string
	ServerName      string
	JobAgentName    string
	TargetGroupName string
}

func NewJobTargetGroupID(subscriptionId, resourceGroup, serverName, jobAgentName, targetGroupName string) JobTargetGroupId {
	return JobTargetGroupId{
		SubscriptionId:  subscriptionId,
		ResourceGroup:   resourceGroup,
		ServerName:      serverName,
		JobAgentName:    jobAgentName,
		TargetGroupName: targetGroupName,
	}
}

func (id JobTargetGroupId) String() string {
	segments := []string{
		fmt.Sprintf("Target Group Name %q", id.TargetGroupName),
		fmt.Sprintf("Job Agent Name %q", id.JobAgentName),
		fmt.Sprintf("Server Name %q", id.ServerName),
		fmt.Sprintf("Resource Group %q", id.ResourceGroup),
	}
	segmentsStr := strings.Join(segments, " / ")
	return fmt.Sprintf("%s: (%s)", "Job Target Group", segmentsStr)
}

func (id JobTargetGroupId) ID() string {
	fmtString := "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Sql/servers/%s/jobAgents/%s/targetGroups/%s"
	return fmt.Sprintf(fmtString, id.SubscriptionId, id.ResourceGroup, id.ServerName, id.JobAgentName, id.TargetGroupName)
}

// JobTargetGroupID parses a JobTargetGroup ID into an JobTargetGroupId struct
func JobTargetGroupID(input string) (*JobTargetGroupId, error) {
	id, err := azure.ParseAzureResourceID(input)
	if err != nil {
		return nil, err
	}

	resourceId := JobTargetGroupId{
		SubscriptionId: id.SubscriptionID,
		ResourceGroup:  id.ResourceGroup,
	}

	if resourceId.SubscriptionId == "" {
		return nil, fmt.Errorf("ID was missing the 'subscriptions' element")
	}

	if resourceId.ResourceGroup == "" {
		return nil, fmt.Errorf("ID was missing the 'resourceGroups' element")
	}

	if resourceId.ServerName, err = id.PopSegment("servers"); err != nil {
		return nil, err
	}
	if resourceId.JobAgentName, err = id.PopSegment("jobAgents"); err != nil {
		return nil, err
	}
	if resourceId.TargetGroupName, err = id.PopSegment("targetGroups"); err != nil {
		return nil, err
	}

	if err := id.ValidateNoEmptySegments(input); err != nil {
		return nil, err
	}

	return &resourceId, nil
}
