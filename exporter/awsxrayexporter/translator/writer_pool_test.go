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
	"bytes"
	"encoding/json"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	semconventions "github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestWriterPoolBasic(t *testing.T) {
	size := 1024
	wp := newWriterPool(size)
	span := constructWriterPoolSpan()
	w := wp.borrow()
	assert.NotNil(t, w)
	assert.NotNil(t, w.buffer)
	assert.NotNil(t, w.encoder)
	assert.Equal(t, size, w.buffer.Cap())
	assert.Equal(t, 0, w.buffer.Len())
	if err := w.Encode(span); err != nil {
		assert.Fail(t, "invalid json")
	}
	jsonStr := w.String()
	assert.Equal(t, len(jsonStr), w.buffer.Len())
	wp.release(w)
}

func BenchmarkWithoutPool(b *testing.B) {
	logger := zap.NewNop()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		span := constructWriterPoolSpan()
		b.StartTimer()
		buffer := bytes.NewBuffer(make([]byte, 0, 2048))
		encoder := json.NewEncoder(buffer)
		encoder.Encode(span)
		logger.Info(buffer.String())
	}
}

func BenchmarkWithPool(b *testing.B) {
	logger := zap.NewNop()
	wp := newWriterPool(2048)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		span := constructWriterPoolSpan()
		b.StartTimer()
		w := wp.borrow()
		w.Encode(span)
		logger.Info(w.String())
	}
}

func constructWriterPoolSpan() *tracepb.Span {
	attributes := make(map[string]interface{})
	attributes[semconventions.AttributeComponent] = semconventions.ComponentTypeHTTP
	attributes[semconventions.AttributeHTTPMethod] = "GET"
	attributes[semconventions.AttributeHTTPURL] = "https://api.example.com/users/junit"
	attributes[semconventions.AttributeHTTPClientIP] = "192.168.15.32"
	attributes[semconventions.AttributeHTTPStatusCode] = 200
	return constructHTTPServerSpan(attributes)
}
