package nexusrpc

import (
	"net/http"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/temporalnexus"
	"go.temporal.io/server/common/telemetry"
)

const maxNexusTelemetryLinks = 10

// ServerSpanAttributes describes Nexus request attributes added to a server trace span.
type ServerSpanAttributes struct {
	SpanName  string
	Endpoint  string
	Service   string
	Operation string
	RequestID string
}

// ServerSpanResultAttributes describes Nexus response attributes added to a server trace span.
type ServerSpanResultAttributes struct {
	Outcome string
	Links   []nexus.Link
}

// AnnotateServerSpanResult adds Nexus response attributes to span.
func AnnotateServerSpanResult(
	span trace.Span,
	attrs ServerSpanResultAttributes,
) {
	// Non-recording spans discard attributes, so avoid constructing them.
	if !span.IsRecording() {
		return
	}
	kvs := []attribute.KeyValue{attribute.String(telemetry.NexusOutcomeKey, attrs.Outcome)}
	linkURLs := make([]string, 0, min(len(attrs.Links), maxNexusTelemetryLinks))
	handlerWorkflows := make(map[handlerWorkflowIdentity]struct{})
	var handlerWorkflow handlerWorkflowIdentity
	for _, link := range attrs.Links {
		workflowEvent, err := temporalnexus.ConvertNexusLinkToLinkWorkflowEvent(link)
		if err != nil || workflowEvent.GetEventRef().GetEventType() != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
			continue
		}
		identity := handlerWorkflowIdentity{
			namespace:  workflowEvent.GetNamespace(),
			workflowID: workflowEvent.GetWorkflowId(),
			runID:      workflowEvent.GetRunId(),
		}
		if _, duplicate := handlerWorkflows[identity]; duplicate {
			continue
		}
		handlerWorkflows[identity] = struct{}{}
		handlerWorkflow = identity
		if len(linkURLs) < maxNexusTelemetryLinks {
			canonicalLink := temporalnexus.ConvertLinkWorkflowEventToNexusLink(workflowEvent)
			linkURLs = append(linkURLs, canonicalLink.URL.String())
		}
	}
	if len(linkURLs) > 0 {
		kvs = append(kvs, attribute.StringSlice(telemetry.NexusLinksKey, linkURLs))
	}
	if len(handlerWorkflows) == 1 {
		kvs = append(kvs,
			attribute.String(telemetry.NexusHandlerNamespaceKey, handlerWorkflow.namespace),
			attribute.String(telemetry.NexusHandlerRunIDKey, handlerWorkflow.runID),
			attribute.String(telemetry.NexusHandlerWorkflowIDKey, handlerWorkflow.workflowID),
		)
	}
	span.SetAttributes(kvs...)
}

type handlerWorkflowIdentity struct {
	namespace  string
	workflowID string
	runID      string
}

// AnnotateServerSpan adds Nexus request attributes to span.
func AnnotateServerSpan(span trace.Span, attrs ServerSpanAttributes) {
	// Non-recording spans discard attributes, so avoid constructing them.
	if !span.IsRecording() {
		return
	}
	if attrs.SpanName != "" {
		span.SetName(attrs.SpanName)
	}
	kvs := []attribute.KeyValue{
		attribute.String(telemetry.NexusServiceKey, attrs.Service),
		attribute.String(telemetry.NexusOperationKey, attrs.Operation),
	}
	if attrs.Endpoint != "" {
		kvs = append(kvs, attribute.String(telemetry.NexusEndpointKey, attrs.Endpoint))
	}
	if attrs.RequestID != "" {
		kvs = append(kvs, attribute.String(telemetry.NexusRequestIDKey, attrs.RequestID))
	}
	span.SetAttributes(kvs...)
}

// AnnotateClientRequest returns req with Nexus attributes for its HTTP client span.
func AnnotateClientRequest(req *http.Request, targetNamespaceName string) *http.Request {
	attrs := make([]attribute.KeyValue, 0, 2)
	if targetNamespaceName != "" {
		attrs = append(attrs, attribute.String(telemetry.NexusNamespaceKey, targetNamespaceName))
	}
	if requestID := req.Header.Get(headerRequestID); requestID != "" {
		attrs = append(attrs, attribute.String(telemetry.NexusRequestIDKey, requestID))
	}
	if len(attrs) > 0 {
		return telemetry.WithHTTPClientSpanAttributes(req, attrs...)
	}
	return req
}
