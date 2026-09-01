package nexusrpc

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	servertelemetry "go.temporal.io/server/common/telemetry"
)

func TestAnnotateServerSpanResultSanitizesHandlerLinks(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
	_, span := provider.Tracer("test").Start(context.Background(), "test")

	startedURL, err := url.Parse("temporal:///namespaces/handler-ns/workflows/handler-wf/handler-run/history?eventID=1&eventType=WorkflowExecutionStarted&referenceType=EventReference&secret=do-not-export")
	require.NoError(t, err)
	applicationURL, err := url.Parse("https://example.com/resource?token=do-not-export")
	require.NoError(t, err)
	AnnotateServerSpanResult(span, ServerSpanResultAttributes{
		Outcome: "async_success",
		Links: []nexus.Link{
			{URL: applicationURL, Type: "example.application.Resource"},
			{URL: startedURL, Type: "temporal.api.common.v1.Link.WorkflowEvent"},
		},
	})
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	attrs := attributesByKey(spans[0].Attributes)
	require.Equal(t, []string{
		"temporal:///namespaces/handler-ns/workflows/handler-wf/handler-run/history?eventID=1&eventType=WorkflowExecutionStarted&referenceType=EventReference",
	}, attrs[servertelemetry.NexusLinksKey].Value.AsStringSlice())
	require.Equal(t, "handler-ns", attrs[servertelemetry.NexusHandlerNamespaceKey].Value.AsString())
	require.Equal(t, "handler-run", attrs[servertelemetry.NexusHandlerRunIDKey].Value.AsString())
	require.Equal(t, "handler-wf", attrs[servertelemetry.NexusHandlerWorkflowIDKey].Value.AsString())
}

func TestAnnotateServerSpanResultOmitsAmbiguousHandlerIdentityAndBoundsLinks(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
	_, span := provider.Tracer("test").Start(context.Background(), "test")

	links := make([]nexus.Link, 11)
	for i := range links {
		linkURL, err := url.Parse(fmt.Sprintf("temporal:///namespaces/handler-ns/workflows/handler-wf-%d/handler-run/history?eventID=1&eventType=WorkflowExecutionStarted&referenceType=EventReference", i))
		require.NoError(t, err)
		links[i] = nexus.Link{URL: linkURL, Type: "temporal.api.common.v1.Link.WorkflowEvent"}
	}
	AnnotateServerSpanResult(span, ServerSpanResultAttributes{
		Outcome: "async_success",
		Links:   links,
	})
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	attrs := attributesByKey(spans[0].Attributes)
	require.Len(t, attrs[servertelemetry.NexusLinksKey].Value.AsStringSlice(), 10)
	require.NotContains(t, attrs, servertelemetry.NexusHandlerNamespaceKey)
	require.NotContains(t, attrs, servertelemetry.NexusHandlerRunIDKey)
	require.NotContains(t, attrs, servertelemetry.NexusHandlerWorkflowIDKey)
}

func TestAnnotateServerSpanResultTreatsDuplicateHandlerLinksAsUnambiguous(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
	_, span := provider.Tracer("test").Start(context.Background(), "test")

	linkURL, err := url.Parse("temporal:///namespaces/handler-ns/workflows/handler-wf/handler-run/history?eventID=1&eventType=WorkflowExecutionStarted&referenceType=EventReference")
	require.NoError(t, err)
	link := nexus.Link{URL: linkURL, Type: "temporal.api.common.v1.Link.WorkflowEvent"}
	AnnotateServerSpanResult(span, ServerSpanResultAttributes{
		Outcome: "async_success",
		Links:   []nexus.Link{link, link},
	})
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	attrs := attributesByKey(spans[0].Attributes)
	require.Equal(t, "handler-ns", attrs[servertelemetry.NexusHandlerNamespaceKey].Value.AsString())
	require.Equal(t, "handler-run", attrs[servertelemetry.NexusHandlerRunIDKey].Value.AsString())
	require.Equal(t, "handler-wf", attrs[servertelemetry.NexusHandlerWorkflowIDKey].Value.AsString())
}

func attributesByKey(attrs []attribute.KeyValue) map[string]attribute.KeyValue {
	result := make(map[string]attribute.KeyValue, len(attrs))
	for _, attr := range attrs {
		result[string(attr.Key)] = attr
	}
	return result
}
