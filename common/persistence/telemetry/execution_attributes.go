package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/server/chasm"
	"go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/telemetry"
)

func executionSpanStartOption(request any) trace.SpanStartOption {
	return trace.WithAttributes(executionSpanAttributes(request)...)
}

type executionIdentity struct {
	archetypeID chasm.ArchetypeID
	namespaceID string
	businessID  string
	runID       string
}

func executionSpanAttributes(request any) []attribute.KeyValue {
	identity := executionIdentityFromRequest(request)
	if identity.archetypeID == 0 {
		return nil
	}

	attrs := make([]attribute.KeyValue, 0, 4)
	if identity.namespaceID != "" {
		attrs = append(attrs, attribute.String(telemetry.NamespaceIDKey, identity.namespaceID))
	}
	if identity.businessID != "" {
		attrs = append(attrs, attribute.String(telemetry.BusinessIDKey, identity.businessID))
	}
	if identity.runID != "" {
		attrs = append(attrs, attribute.String(telemetry.RunIDKey, identity.runID))
	}
	return append(attrs, attribute.Int64(telemetry.ChasmArchetypeIDKey, int64(identity.archetypeID)))
}

func executionIdentityFromRequest(request any) executionIdentity {
	switch request := request.(type) {
	case *persistence.InternalCreateWorkflowExecutionRequest:
		return executionIdentityFromSnapshot(request.ArchetypeID, request.NewWorkflowSnapshot)
	case *persistence.InternalUpdateWorkflowExecutionRequest:
		return executionIdentityFromMutation(request.ArchetypeID, request.UpdateWorkflowMutation)
	case *persistence.InternalConflictResolveWorkflowExecutionRequest:
		return executionIdentityFromSnapshot(request.ArchetypeID, request.ResetWorkflowSnapshot)
	case *persistence.InternalSetWorkflowExecutionRequest:
		return executionIdentityFromSnapshot(request.ArchetypeID, request.SetWorkflowSnapshot)
	case *persistence.GetCurrentExecutionRequest:
		return executionIdentity{
			archetypeID: request.ArchetypeID,
			namespaceID: request.NamespaceID,
			businessID:  request.WorkflowID,
		}
	case *persistence.GetWorkflowExecutionRequest:
		return executionIdentity{
			archetypeID: request.ArchetypeID,
			namespaceID: request.NamespaceID,
			businessID:  request.WorkflowID,
			runID:       request.RunID,
		}
	case *persistence.DeleteCurrentWorkflowExecutionRequest:
		return executionIdentity{
			archetypeID: request.ArchetypeID,
			namespaceID: request.NamespaceID,
			businessID:  request.WorkflowID,
			runID:       request.RunID,
		}
	case *persistence.DeleteWorkflowExecutionRequest:
		return executionIdentity{
			archetypeID: request.ArchetypeID,
			namespaceID: request.NamespaceID,
			businessID:  request.WorkflowID,
			runID:       request.RunID,
		}
	default:
		return executionIdentity{}
	}
}

func executionIdentityFromMutation(
	archetypeID chasm.ArchetypeID,
	mutation persistence.InternalWorkflowMutation,
) executionIdentity {
	return executionIdentity{
		archetypeID: archetypeID,
		namespaceID: mutation.NamespaceID,
		businessID:  mutation.WorkflowID,
		runID:       mutation.RunID,
	}
}

func executionIdentityFromSnapshot(
	archetypeID chasm.ArchetypeID,
	snapshot persistence.InternalWorkflowSnapshot,
) executionIdentity {
	return executionIdentity{
		archetypeID: archetypeID,
		namespaceID: snapshot.NamespaceID,
		businessID:  snapshot.WorkflowID,
		runID:       snapshot.RunID,
	}
}
