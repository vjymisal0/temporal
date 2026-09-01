package telemetry

import (
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const maxRPCNamespaceDepth = 5

func rpcNamespaceAttributes(payload any) []attribute.KeyValue {
	message, ok := payload.(proto.Message)
	if !ok {
		return nil
	}
	namespace, namespaceID := rpcNamespaceValues(message.ProtoReflect(), 0)
	attrs := make([]attribute.KeyValue, 0, 2)
	if namespace != "" {
		attrs = append(attrs, attribute.String(NamespaceKey, namespace))
	}
	if namespaceID != "" {
		attrs = append(attrs, attribute.String(NamespaceIDKey, namespaceID))
	}
	return attrs
}

func rpcNamespaceValues(message protoreflect.Message, depth int) (namespace string, namespaceID string) {
	if !message.IsValid() || depth > maxRPCNamespaceDepth {
		return "", ""
	}

	fields := message.Descriptor().Fields()
	namespace = rpcStringField(message, fields.ByName("namespace"))
	namespaceID = rpcStringField(message, fields.ByName("namespace_id"))
	if namespace != "" && namespaceID != "" {
		return namespace, namespaceID
	}

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.IsList() || field.IsMap() || field.Kind() != protoreflect.MessageKind || !message.Has(field) {
			continue
		}
		if !isRPCNamespaceNestedField(field.Name()) {
			continue
		}
		nestedNamespace, nestedNamespaceID := rpcNamespaceValues(message.Get(field).Message(), depth+1)
		if namespace == "" {
			namespace = nestedNamespace
		}
		if namespaceID == "" {
			namespaceID = nestedNamespaceID
		}
		if namespace != "" && namespaceID != "" {
			break
		}
	}
	return namespace, namespaceID
}

func isRPCNamespaceNestedField(name protoreflect.Name) bool {
	switch name {
	case "completion", "execution_info", "execution_state", "parent_execution", "update_ref", "workflow_state":
		return true
	default:
		return strings.HasSuffix(string(name), "_request")
	}
}

func rpcStringField(message protoreflect.Message, field protoreflect.FieldDescriptor) string {
	if field == nil || field.Kind() != protoreflect.StringKind {
		return ""
	}
	return message.Get(field).String()
}
