// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import (
	"strings"
	"testing"
	"time"

	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/wrappers"
	semconventions "github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	"github.com/stretchr/testify/assert"
)

func TestClientSpanWithURLAttribute(t *testing.T) {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPURL] = "https://api.example.com/users/junit"
	attributes[semconventions.AttributeHTTPStatusCode] = 200
	span := constructHTTPClientSpan(attributes)

	filtered, httpData := makeHTTP(span)

	assert.NotNil(t, httpData)
	assert.NotNil(t, filtered)
	w := testWriters.borrow()
	if err := w.Encode(httpData); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	testWriters.release(w)
	assert.True(t, strings.Contains(jsonStr, "https://api.example.com/users/junit"))
}

func TestClientSpanWithSchemeHostTargetAttributes(t *testing.T) {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPScheme] = "https"
	attributes[semconventions.AttributeHTTPHost] = "api.example.com"
	attributes[semconventions.AttributeHTTPTarget] = "/users/junit"
	attributes[semconventions.AttributeHTTPStatusCode] = 200
	attributes["user.id"] = "junit"
	span := constructHTTPClientSpan(attributes)

	filtered, httpData := makeHTTP(span)

	assert.NotNil(t, httpData)
	assert.NotNil(t, filtered)
	w := testWriters.borrow()
	if err := w.Encode(httpData); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	testWriters.release(w)
	assert.True(t, strings.Contains(jsonStr, "https://api.example.com/users/junit"))
}

func TestClientSpanWithPeerAttributes(t *testing.T) {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPScheme] = "http"
	attributes[semconventions.AttributePeerHost] = "kb234.example.com"
	attributes[semconventions.AttributePeerPort] = 8080
	attributes[semconventions.AttributePeerIpv4] = "10.8.17.36"
	attributes[semconventions.AttributeHTTPTarget] = "/users/junit"
	attributes[semconventions.AttributeHTTPStatusCode] = 200
	span := constructHTTPClientSpan(attributes)

	filtered, httpData := makeHTTP(span)

	assert.NotNil(t, httpData)
	assert.NotNil(t, filtered)
	w := testWriters.borrow()
	if err := w.Encode(httpData); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	testWriters.release(w)
	assert.True(t, strings.Contains(jsonStr, "http://kb234.example.com:8080/users/junit"))
}

func TestClientSpanWithPeerIp4Attributes(t *testing.T) {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPScheme] = "http"
	attributes[semconventions.AttributePeerIpv4] = "10.8.17.36"
	attributes[semconventions.AttributePeerPort] = "8080"
	attributes[semconventions.AttributeHTTPTarget] = "/users/junit"
	span := constructHTTPClientSpan(attributes)

	filtered, httpData := makeHTTP(span)
	assert.NotNil(t, httpData)
	assert.NotNil(t, filtered)
	w := testWriters.borrow()
	if err := w.Encode(httpData); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	testWriters.release(w)
	assert.True(t, strings.Contains(jsonStr, "http://10.8.17.36:8080/users/junit"))
}

func TestClientSpanWithPeerIp6Attributes(t *testing.T) {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPScheme] = "https"
	attributes[semconventions.AttributePeerIpv6] = "2001:db8:85a3::8a2e:370:7334"
	attributes[semconventions.AttributePeerPort] = "443"
	attributes[semconventions.AttributeHTTPTarget] = "/users/junit"
	span := constructHTTPClientSpan(attributes)

	filtered, httpData := makeHTTP(span)
	assert.NotNil(t, httpData)
	assert.NotNil(t, filtered)
	w := testWriters.borrow()
	if err := w.Encode(httpData); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	testWriters.release(w)
	assert.True(t, strings.Contains(jsonStr, "https://2001:db8:85a3::8a2e:370:7334/users/junit"))
}

func TestServerSpanWithURLAttribute(t *testing.T) {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPURL] = "https://api.example.com/users/junit"
	attributes[semconventions.AttributeHTTPClientIP] = "192.168.15.32"
	attributes[semconventions.AttributeHTTPStatusCode] = 200
	span := constructHTTPServerSpan(attributes)

	filtered, httpData := makeHTTP(span)

	assert.NotNil(t, httpData)
	assert.NotNil(t, filtered)
	w := testWriters.borrow()
	if err := w.Encode(httpData); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	testWriters.release(w)
	assert.True(t, strings.Contains(jsonStr, "https://api.example.com/users/junit"))
}

func TestServerSpanWithSchemeHostTargetAttributes(t *testing.T) {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPScheme] = "https"
	attributes[semconventions.AttributeHTTPHost] = "api.example.com"
	attributes[semconventions.AttributeHTTPTarget] = "/users/junit"
	attributes[semconventions.AttributeHTTPClientIP] = "192.168.15.32"
	attributes[semconventions.AttributeHTTPStatusCode] = 200
	span := constructHTTPServerSpan(attributes)

	filtered, httpData := makeHTTP(span)

	assert.NotNil(t, httpData)
	assert.NotNil(t, filtered)
	w := testWriters.borrow()
	if err := w.Encode(httpData); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	testWriters.release(w)
	assert.True(t, strings.Contains(jsonStr, "https://api.example.com/users/junit"))
}

func TestServerSpanWithSchemeServernamePortTargetAttributes(t *testing.T) {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPScheme] = "https"
	attributes[semconventions.AttributeHTTPServerName] = "api.example.com"
	attributes[semconventions.AttributeHTTPHostPort] = 443
	attributes[semconventions.AttributeHTTPTarget] = "/users/junit"
	attributes[semconventions.AttributeHTTPClientIP] = "192.168.15.32"
	attributes[semconventions.AttributeHTTPStatusCode] = 200
	span := constructHTTPServerSpan(attributes)

	filtered, httpData := makeHTTP(span)

	assert.NotNil(t, httpData)
	assert.NotNil(t, filtered)
	w := testWriters.borrow()
	if err := w.Encode(httpData); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	testWriters.release(w)
	assert.True(t, strings.Contains(jsonStr, "https://api.example.com/users/junit"))
}

func TestServerSpanWithSchemeNamePortTargetAttributes(t *testing.T) {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPScheme] = "http"
	attributes[semconventions.AttributeHostName] = "kb234.example.com"
	attributes[semconventions.AttributeHTTPHostPort] = 8080
	attributes[semconventions.AttributeHTTPTarget] = "/users/junit"
	attributes[semconventions.AttributeHTTPClientIP] = "192.168.15.32"
	attributes[semconventions.AttributeHTTPStatusCode] = 200
	span := constructHTTPServerSpan(attributes)
	timeEvents := constructTimedEventsWithReceivedMessageEvent(span.EndTime)
	span.TimeEvents = &timeEvents

	filtered, httpData := makeHTTP(span)

	assert.NotNil(t, httpData)
	assert.NotNil(t, filtered)
	w := testWriters.borrow()
	if err := w.Encode(httpData); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	testWriters.release(w)
	assert.True(t, strings.Contains(jsonStr, "http://kb234.example.com:8080/users/junit"))
}

func TestHttpStatusFromSpanStatus(t *testing.T) {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPURL] = "https://api.example.com/users/junit"
	span := constructHTTPClientSpan(attributes)

	filtered, httpData := makeHTTP(span)

	assert.NotNil(t, httpData)
	assert.NotNil(t, filtered)
	w := testWriters.borrow()
	if err := w.Encode(httpData); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	testWriters.release(w)
	assert.True(t, strings.Contains(jsonStr, "200"))
}

func constructHTTPClientSpan(attributes map[string]interface{}) *tracepb.Span {
	endTime := time.Now().Round(time.Second)
	startTime := endTime.Add(-90 * time.Second)
	spanAttributes := constructSpanAttributes(attributes)

	return &tracepb.Span{
		TraceId:      newTraceID(),
		SpanId:       newSegmentID(),
		ParentSpanId: newSegmentID(),
		Name:         &tracepb.TruncatableString{Value: "/users/junit"},
		Kind:         tracepb.Span_CLIENT,
		StartTime:    convertTimeToTimestamp(startTime),
		EndTime:      convertTimeToTimestamp(endTime),
		Status: &tracepb.Status{
			Code:    0,
			Message: "OK",
		},
		SameProcessAsParentSpan: &wrappers.BoolValue{Value: false},
		Tracestate: &tracepb.Span_Tracestate{
			Entries: []*tracepb.Span_Tracestate_Entry{
				{Key: "foo", Value: "bar"},
				{Key: "a", Value: "b"},
			},
		},
		Attributes: &tracepb.Span_Attributes{
			AttributeMap: spanAttributes,
		},
		Resource: &resourcepb.Resource{
			Type:   "container",
			Labels: constructDefaultResourceLabels(),
		},
	}
}

func constructHTTPServerSpan(attributes map[string]interface{}) *tracepb.Span {
	endTime := time.Now().Round(time.Second)
	startTime := endTime.Add(-90 * time.Second)
	spanAttributes := constructSpanAttributes(attributes)

	return &tracepb.Span{
		TraceId:      newTraceID(),
		SpanId:       newSegmentID(),
		ParentSpanId: newSegmentID(),
		Name:         &tracepb.TruncatableString{Value: "/users/junit"},
		Kind:         tracepb.Span_SERVER,
		StartTime:    convertTimeToTimestamp(startTime),
		EndTime:      convertTimeToTimestamp(endTime),
		Status: &tracepb.Status{
			Code:    0,
			Message: "OK",
		},
		SameProcessAsParentSpan: &wrappers.BoolValue{Value: false},
		Tracestate: &tracepb.Span_Tracestate{
			Entries: []*tracepb.Span_Tracestate_Entry{
				{Key: "foo", Value: "bar"},
				{Key: "a", Value: "b"},
			},
		},
		Attributes: &tracepb.Span_Attributes{
			AttributeMap: spanAttributes,
		},
		Resource: &resourcepb.Resource{
			Type:   "container",
			Labels: constructDefaultResourceLabels(),
		},
	}
}
