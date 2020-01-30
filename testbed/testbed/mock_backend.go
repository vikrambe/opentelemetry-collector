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

package testbed

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

// MockBackend is a backend that allows receiving the data locally.
type MockBackend struct {
	// Metric and trace consumers
	tc *MockTraceConsumer
	mc *MockMetricConsumer

	receiver DataReceiver

	// Log file
	logFilePath string
	logFile     *os.File

	// Start/stop flags
	isStarted bool
	stopOnce  sync.Once

	// Recording fields.
	isRecording     bool
	recordMutex     sync.Mutex
	ReceivedTraces  []consumerdata.TraceData
	ReceivedMetrics []consumerdata.MetricsData
}

// NewMockBackend creates a new mock backend that receives data using specified receiver.
func NewMockBackend(logFilePath string, receiver DataReceiver) *MockBackend {
	mb := &MockBackend{
		logFilePath: logFilePath,
		receiver:    receiver,
		tc:          &MockTraceConsumer{},
		mc:          &MockMetricConsumer{},
	}
	mb.tc.backend = mb
	mb.mc.backend = mb
	return mb
}

var _ component.Host = (*MockBackend)(nil)

func (mb *MockBackend) Context() context.Context {
	return context.Background()
}

func (mb *MockBackend) ReportFatalError(err error) {
	log.Printf("Fatal error reported: %v", err)
}

// Start a backend.
func (mb *MockBackend) Start() error {
	log.Printf("Starting mock backend...")

	var err error

	// Open log file
	mb.logFile, err = os.Create(mb.logFilePath)
	if err != nil {
		return err
	}

	err = mb.receiver.Start(mb.tc, mb.mc)
	if err != nil {
		return err
	}

	mb.isStarted = true
	return nil
}

// Stop the backend
func (mb *MockBackend) Stop() {
	mb.stopOnce.Do(func() {
		if !mb.isStarted {
			return
		}

		log.Printf("Stopping mock backend...")

		mb.logFile.Close()
		mb.receiver.Stop()

		// Print stats.
		log.Printf("Stopped backend. %s", mb.GetStats())
	})
}

// EnableRecording enables recording of all data received by MockBackend.
func (mb *MockBackend) EnableRecording() {
	mb.recordMutex.Lock()
	defer mb.recordMutex.Unlock()
	mb.isRecording = true
}

func (mb *MockBackend) GetStats() string {
	return fmt.Sprintf("Received:%5d items", mb.DataItemsReceived())
}

// DataItemsReceived returns total number of received spans and metrics.
func (mb *MockBackend) DataItemsReceived() uint64 {
	return atomic.LoadUint64(&mb.tc.spansReceived) + atomic.LoadUint64(&mb.mc.metricsReceived)
}

// ClearReceivedItems clears the list of received traces and metrics. Note: counters
// return by DataItemsReceived() are not cleared, they are cumulative.
func (mb *MockBackend) ClearReceivedItems() {
	mb.recordMutex.Lock()
	defer mb.recordMutex.Unlock()
	mb.ReceivedTraces = nil
	mb.ReceivedMetrics = nil
}

func (mb *MockBackend) ConsumeTrace(td consumerdata.TraceData) {
	mb.recordMutex.Lock()
	defer mb.recordMutex.Unlock()
	if mb.isRecording {
		mb.ReceivedTraces = append(mb.ReceivedTraces, td)
	}
}

func (mb *MockBackend) ConsumeMetric(md consumerdata.MetricsData) {
	mb.recordMutex.Lock()
	defer mb.recordMutex.Unlock()
	if mb.isRecording {
		mb.ReceivedMetrics = append(mb.ReceivedMetrics, md)
	}
}

type MockTraceConsumer struct {
	spansReceived uint64
	backend       *MockBackend
}

func (tc *MockTraceConsumer) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	atomic.AddUint64(&tc.spansReceived, uint64(len(td.Spans)))

	for _, span := range td.Spans {
		var spanSeqnum int64
		var traceSeqnum int64

		if span.Attributes != nil {
			seqnumAttr, ok := span.Attributes.AttributeMap["load_generator.span_seq_num"]
			if ok {
				spanSeqnum = seqnumAttr.GetIntValue()
			}

			seqnumAttr, ok = span.Attributes.AttributeMap["load_generator.trace_seq_num"]
			if ok {
				traceSeqnum = seqnumAttr.GetIntValue()
			}

			// Ignore the seqnums for now. We will use them later.
			_ = spanSeqnum
			_ = traceSeqnum
		}

		tc.backend.ConsumeTrace(td)
	}

	return nil
}

type MockMetricConsumer struct {
	metricsReceived uint64
	backend         *MockBackend
}

func (mc *MockMetricConsumer) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	dataPoints := 0
	for _, metric := range md.Metrics {
		for _, ts := range metric.Timeseries {
			dataPoints += len(ts.Points)
		}
	}

	atomic.AddUint64(&mc.metricsReceived, uint64(dataPoints))

	mc.backend.ConsumeMetric(md)

	return nil
}
