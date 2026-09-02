// Generates all generated files in this package:
//go:generate go run ../../../../cmd/tools/genrpcserverinterceptors

package logtags

import (
	"strings"

	"go.temporal.io/server/common/api"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/tasktoken"
)

type (
	WorkflowTags struct {
		serializer *tasktoken.Serializer
		logger     log.Logger
	}

	extractedTags struct {
		tags        []tag.Tag
		namespace   string
		namespaceID string
	}
)

func NewWorkflowTags(
	serializer *tasktoken.Serializer,
	logger log.Logger,
) *WorkflowTags {
	return &WorkflowTags{
		serializer: serializer,
		logger:     logger,
	}
}

func (wt *WorkflowTags) Extract(req any, fullMethod string) []tag.Tag {
	return wt.extract(req, fullMethod).tags
}

// ExtractForSpan returns workflow tags and namespace identifiers for span attributes.
func (wt *WorkflowTags) ExtractForSpan(
	req any,
	fullMethod string,
) (tags []tag.Tag, namespace string, namespaceID string) {
	extracted := wt.extract(req, fullMethod)
	return extracted.tags, extracted.namespace, extracted.namespaceID
}

func (wt *WorkflowTags) extract(req any, fullMethod string) extractedTags {
	if req == nil {
		return extractedTags{}
	}
	switch {
	case strings.HasPrefix(fullMethod, api.WorkflowServicePrefix):
		return wt.extractFromWorkflowServiceServerMessage(req)
	case strings.HasPrefix(fullMethod, api.OperatorServicePrefix):
		// OperatorService doesn't have a single API with workflow tags.
		return extractedTags{}
	case strings.HasPrefix(fullMethod, api.AdminServicePrefix):
		return wt.extractFromAdminServiceServerMessage(req)
	case strings.HasPrefix(fullMethod, api.HistoryServicePrefix):
		return wt.extractFromHistoryServiceServerMessage(req)
	case strings.HasPrefix(fullMethod, api.MatchingServicePrefix):
		return wt.extractFromMatchingServiceServerMessage(req)
	default:
		return extractedTags{}
	}
}

func (wt *WorkflowTags) fromTaskToken(taskTokenBytes []byte) []tag.Tag {
	if len(taskTokenBytes) == 0 {
		return nil
	}
	taskToken, err := wt.serializer.Deserialize(taskTokenBytes)
	if err != nil {
		wt.logger.Warn("unable to deserialize task token while getting workflow tags", tag.Error(err))
		return nil
	}
	return []tag.Tag{tag.WorkflowID(taskToken.WorkflowId), tag.WorkflowRunID(taskToken.RunId)}
}
