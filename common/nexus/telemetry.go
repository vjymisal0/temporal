package nexus

import (
	"github.com/nexus-rpc/sdk-go/nexus"
	enumspb "go.temporal.io/api/enums/v1"
)

// HandlerWorkflowIDFromLinks returns the workflow ID from the first WorkflowExecutionStarted link.
func HandlerWorkflowIDFromLinks(links []nexus.Link) string {
	for _, link := range links {
		workflowEvent, err := ConvertNexusLinkToLinkWorkflowEvent(link)
		if err == nil && workflowEvent.GetEventRef().GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
			return workflowEvent.GetWorkflowId()
		}
	}
	return ""
}
