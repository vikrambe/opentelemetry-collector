// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

var traceFile = &File{
	Name: "trace",
	imports: []string{
		`otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"`,
		`otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"`,
		`otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"`,
	},
	testImports: []string{
		`"testing"`,
		``,
		`otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"`,
		`"github.com/stretchr/testify/assert"`,
	},
	structs: []baseStruct{
		resourceSpansSlice,
		resourceSpans,
		instrumentationLibrarySpansSlice,
		instrumentationLibrarySpans,
		spanSlice,
		span,
		spanEventSlice,
		spanEvent,
		spanLinkSlice,
		spanLink,
		spanStatus,
	},
}

var resourceSpansSlice = &sliceStruct{
	structName: "ResourceSpansSlice",
	element:    resourceSpans,
}

var resourceSpans = &messageStruct{
	structName:     "ResourceSpans",
	description:    "// InstrumentationLibrarySpans is a collection of spans from a LibraryInstrumentation.",
	originFullName: "otlptrace.ResourceSpans",
	fields: []baseField{
		resourceField,
		&sliceField{
			fieldMame:               "InstrumentationLibrarySpans",
			originFieldName:         "InstrumentationLibrarySpans",
			returnSlice:             instrumentationLibrarySpansSlice,
			constructorDefaultValue: "0",
		},
	},
}

var instrumentationLibrarySpansSlice = &sliceStruct{
	structName: "InstrumentationLibrarySpansSlice",
	element:    instrumentationLibrarySpans,
}

var instrumentationLibrarySpans = &messageStruct{
	structName:     "InstrumentationLibrarySpans",
	description:    "// InstrumentationLibrarySpans is a collection of spans from a LibraryInstrumentation.",
	originFullName: "otlptrace.InstrumentationLibrarySpans",
	fields: []baseField{
		instrumentationLibraryField,
		&sliceField{
			fieldMame:               "Spans",
			originFieldName:         "Spans",
			returnSlice:             spanSlice,
			constructorDefaultValue: "0",
		},
	},
}

var spanSlice = &sliceStruct{
	structName: "SpanSlice",
	element:    span,
}

var span = &messageStruct{
	structName: "Span",
	description: "// Span represents a single operation within a trace.\n" +
		"// See Span definition in OTLP: https://github.com/open-telemetry/opentelemetry-proto/blob/master/opentelemetry/proto/trace/v1/trace.proto#L37",
	originFullName: "otlptrace.Span",
	fields: []baseField{
		traceIDField,
		spanIDField,
		traceStateField,
		parentSpanIDField,
		nameField,
		&primitiveTypedField{
			fieldMame:       "Kind",
			originFieldName: "Kind",
			returnType:      "SpanKind",
			rawType:         "otlptrace.Span_SpanKind",
			defaultVal:      "SpanKindUNSPECIFIED",
			testVal:         "SpanKindSERVER",
		},
		startTimeField,
		endTimeField,
		attributes,
		droppedAttributesCount,
		&sliceField{
			fieldMame:               "Events",
			originFieldName:         "Events",
			returnSlice:             spanEventSlice,
			constructorDefaultValue: "0",
		},
		&primitiveField{
			fieldMame:       "DroppedEventsCount",
			originFieldName: "DroppedEventsCount",
			returnType:      "uint32",
			defaultVal:      "uint32(0)",
			testVal:         "uint32(17)",
		},
		&sliceField{
			fieldMame:               "Links",
			originFieldName:         "Links",
			returnSlice:             spanLinkSlice,
			constructorDefaultValue: "0",
		},
		&primitiveField{
			fieldMame:       "DroppedLinksCount",
			originFieldName: "DroppedLinksCount",
			returnType:      "uint32",
			defaultVal:      "uint32(0)",
			testVal:         "uint32(17)",
		},
		&messageField{
			fieldMame:       "Status",
			originFieldName: "Status",
			returnMessage:   spanStatus,
		},
	},
}

var spanEventSlice = &sliceStruct{
	structName: "SpanEventSlice",
	element:    spanEvent,
}

var spanEvent = &messageStruct{
	structName: "SpanEvent",
	description: "// SpanEvent is a time-stamped annotation of the span, consisting of user-supplied\n" +
		"// text description and key-value pairs. See OTLP for event definition.",
	originFullName: "otlptrace.Span_Event",
	fields: []baseField{
		timeField,
		nameField,
		attributes,
		droppedAttributesCount,
	},
}

var spanLinkSlice = &sliceStruct{
	structName: "SpanLinkSlice",
	element:    spanLink,
}

var spanLink = &messageStruct{
	structName: "SpanLink",
	description: "// SpanLink is a pointer from the current span to another span in the same trace or in a\n" +
		"// different trace. See OTLP for link definition.",
	originFullName: "otlptrace.Span_Link",
	fields: []baseField{
		traceIDField,
		spanIDField,
		traceStateField,
		attributes,
		droppedAttributesCount,
	},
}

var spanStatus = &messageStruct{
	structName: "SpanStatus",
	description: "// SpanStatus is an optional final status for this span. Semantically when Status wasn't set\n" +
		"// it is means span ended without errors and assume Status.Ok (code = 0).",
	originFullName: "otlptrace.Status",
	fields: []baseField{
		&primitiveTypedField{
			fieldMame:       "Code",
			originFieldName: "Code",
			returnType:      "StatusCode",
			rawType:         "otlptrace.Status_StatusCode",
			defaultVal:      "StatusCode(0)",
			testVal:         "StatusCode(1)",
		},
		&primitiveField{
			fieldMame:       "Message",
			originFieldName: "Message",
			returnType:      "string",
			defaultVal:      `""`,
			testVal:         `"cancelled"`,
		},
	},
}

var traceIDField = &primitiveTypedField{
	fieldMame:       "TraceID",
	originFieldName: "TraceId",
	returnType:      "TraceID",
	rawType:         "[]byte",
	defaultVal:      "NewTraceID(nil)",
	testVal:         "NewTraceID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1})",
}

var spanIDField = &primitiveTypedField{
	fieldMame:       "SpanID",
	originFieldName: "SpanId",
	returnType:      "SpanID",
	rawType:         "[]byte",
	defaultVal:      "NewSpanID(nil)",
	testVal:         "NewSpanID([]byte{1, 2, 3, 4, 5, 6, 7, 8})",
}

var parentSpanIDField = &primitiveTypedField{
	fieldMame:       "ParentSpanID",
	originFieldName: "ParentSpanId",
	returnType:      "SpanID",
	rawType:         "[]byte",
	defaultVal:      "NewSpanID(nil)",
	testVal:         "NewSpanID([]byte{8, 7, 6, 5, 4, 3, 2, 1})",
}

var traceStateField = &primitiveTypedField{
	fieldMame:       "TraceState",
	originFieldName: "TraceState",
	returnType:      "TraceState",
	rawType:         "string",
	defaultVal:      `TraceState("")`,
	testVal:         `TraceState("congo=congos")`,
}

var droppedAttributesCount = &primitiveField{
	fieldMame:       "DroppedAttributesCount",
	originFieldName: "DroppedAttributesCount",
	returnType:      "uint32",
	defaultVal:      "uint32(0)",
	testVal:         "uint32(17)",
}
