package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/matchingservice/v1"
)

func TestRPCNamespaceAttributes(t *testing.T) {
	t.Run("direct namespace", func(t *testing.T) {
		attrs := rpcNamespaceAttributes(&workflowservice.StartWorkflowExecutionRequest{
			Namespace: "namespace",
		})

		require.Equal(t, []attribute.KeyValue{
			attribute.String(NamespaceKey, "namespace"),
		}, attrs)
	})

	t.Run("direct ID and nested namespace", func(t *testing.T) {
		attrs := rpcNamespaceAttributes(&matchingservice.QueryWorkflowRequest{
			NamespaceId:  "namespace-id",
			QueryRequest: &workflowservice.QueryWorkflowRequest{Namespace: "namespace"},
		})

		require.Equal(t, []attribute.KeyValue{
			attribute.String(NamespaceKey, "namespace"),
			attribute.String(NamespaceIDKey, "namespace-id"),
		}, attrs)
	})

	t.Run("empty", func(t *testing.T) {
		require.Empty(t, rpcNamespaceAttributes(&workflowservice.StartWorkflowExecutionRequest{}))
	})
}
