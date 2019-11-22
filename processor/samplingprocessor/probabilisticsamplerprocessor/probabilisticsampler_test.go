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

package probabilisticsamplerprocessor

import (
	"context"
	"math"
	"math/rand"
	"reflect"
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	processormetrics "github.com/open-telemetry/opentelemetry-collector/processor"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func TestNewTraceProcessor(t *testing.T) {
	tests := []struct {
		name         string
		nextConsumer consumer.TraceConsumer
		cfg          Config
		want         processor.TraceProcessor
		wantErr      bool
	}{
		{
			name:    "nil_nextConsumer",
			wantErr: true,
		},
		{
			name:         "happy_path",
			nextConsumer: &exportertest.SinkTraceExporter{},
			cfg: Config{
				SamplingPercentage: 15.5,
			},
			want: &tracesamplerprocessor{
				nextConsumer: &exportertest.SinkTraceExporter{},
			},
		},
		{
			name:         "happy_path_hash_seed",
			nextConsumer: &exportertest.SinkTraceExporter{},
			cfg: Config{
				SamplingPercentage: 13.33,
				HashSeed:           4321,
			},
			want: &tracesamplerprocessor{
				nextConsumer: &exportertest.SinkTraceExporter{},
				hashSeed:     4321,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				// The truncation below with uint32 cannot be defined at initialization (compiler error), performing it at runtime.
				tt.want.(*tracesamplerprocessor).scaledSamplingRate = uint32(tt.cfg.SamplingPercentage * percentageScaleFactor)
			}
			got, err := NewTraceProcessor(tt.nextConsumer, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTraceProcessor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTraceProcessor() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_tracesamplerprocessor_SamplingPercentageRange checks for different sampling rates and ensures
// that they are within acceptable deltas.
func Test_tracesamplerprocessor_SamplingPercentageRange(t *testing.T) {
	tests := []struct {
		name              string
		cfg               Config
		numBatches        int
		numTracesPerBatch int
		acceptableDelta   float64
	}{
		{
			name: "random_sampling_tiny",
			cfg: Config{
				SamplingPercentage: 0.03,
			},
			numBatches:        1e5,
			numTracesPerBatch: 2,
			acceptableDelta:   0.01,
		},
		{
			name: "random_sampling_small",
			cfg: Config{
				SamplingPercentage: 5,
			},
			numBatches:        1e5,
			numTracesPerBatch: 2,
			acceptableDelta:   0.01,
		},
		{
			name: "random_sampling_medium",
			cfg: Config{
				SamplingPercentage: 50.0,
			},
			numBatches:        1e5,
			numTracesPerBatch: 4,
			acceptableDelta:   0.1,
		},
		{
			name: "random_sampling_high",
			cfg: Config{
				SamplingPercentage: 90.0,
			},
			numBatches:        1e5,
			numTracesPerBatch: 1,
			acceptableDelta:   0.2,
		},
		{
			name: "random_sampling_all",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			numBatches:        1e5,
			numTracesPerBatch: 1,
			acceptableDelta:   0.0,
		},
	}
	const testSvcName = "test-svc"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := &exportertest.SinkTraceExporter{}
			tsp, err := NewTraceProcessor(sink, tt.cfg)
			if err != nil {
				t.Errorf("error when creating tracesamplerprocessor: %v", err)
				return
			}
			for _, td := range genRandomTestData(tt.numBatches, tt.numTracesPerBatch, testSvcName) {
				if err := tsp.ConsumeTraceData(context.Background(), td); err != nil {
					t.Errorf("tracesamplerprocessor.ConsumeTraceData() error = %v", err)
					return
				}
			}
			_, sampled := assertSampledData(t, sink.AllTraces(), testSvcName)
			actualPercentageSamplingPercentage := float32(sampled) / float32(tt.numBatches*tt.numTracesPerBatch) * 100.0
			delta := math.Abs(float64(actualPercentageSamplingPercentage - tt.cfg.SamplingPercentage))
			if delta > tt.acceptableDelta {
				t.Errorf(
					"got %f percentage sampling rate, want %f (allowed delta is %f but got %f)",
					actualPercentageSamplingPercentage,
					tt.cfg.SamplingPercentage,
					tt.acceptableDelta,
					delta,
				)
			}
		})
	}
}

// Test_hash ensures that the hash function supports different key lengths even if in
// practice it is only expected to receive keys with length 16 (trace id length in OC proto).
func Test_hash(t *testing.T) {
	// Statistically a random selection of such small number of keys should not result in
	// collisions, but, of course it is possible that they happen, a different random source
	// should avoid that.
	r := rand.New(rand.NewSource(1))
	fullKey := tracetranslator.UInt64ToByteTraceID(r.Uint64(), r.Uint64())
	seen := make(map[uint32]bool)
	for i := 1; i <= len(fullKey); i++ {
		key := fullKey[:i]
		hash := hash(key, 1)
		require.False(t, seen[hash], "Unexpected duplicated hash")
		seen[hash] = true
	}
}

// genRandomTestData generates a slice of consumerdata.TraceData with the numBatches elements which one with
// numTracesPerBatch spans (ie.: each span has a different trace ID). All spans belong to the specified
// serviceName.
func genRandomTestData(numBatches, numTracesPerBatch int, serviceName string) (tdd []consumerdata.TraceData) {
	r := rand.New(rand.NewSource(1))

	for i := 0; i < numBatches; i++ {
		var spans []*tracepb.Span
		for j := 0; j < numTracesPerBatch; j++ {
			span := &tracepb.Span{
				TraceId: tracetranslator.UInt64ToByteTraceID(r.Uint64(), r.Uint64()),
			}
			spans = append(spans, span)
		}
		td := consumerdata.TraceData{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: serviceName},
			},
			Spans: spans,
		}
		tdd = append(tdd, td)
	}

	return tdd
}

// assertSampledData checks for no repeated traceIDs and counts the number of spans on the sampled data for
// the given service.
func assertSampledData(t *testing.T, sampled []consumerdata.TraceData, serviceName string) (traceIDs map[string]bool, spanCount int) {
	traceIDs = make(map[string]bool)
	for _, td := range sampled {
		if processormetrics.ServiceNameForNode(td.Node) != serviceName {
			continue
		}
		for _, span := range td.Spans {
			spanCount++
			key := string(span.TraceId)
			if traceIDs[key] {
				t.Errorf("same traceID used more than once %q", key)
				return
			}
			traceIDs[key] = true
		}
	}
	return
}
