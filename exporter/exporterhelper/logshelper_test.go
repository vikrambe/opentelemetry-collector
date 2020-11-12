// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package exporterhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

const (
	fakeLogsExporterType   = "fake_logs_exporter"
	fakeLogsExporterName   = "fake_logs_exporter/with_name"
	fakeLogsParentSpanName = "fake_logs_parent_span_name"
)

var (
	fakeLogsExporterConfig = &configmodels.ExporterSettings{
		TypeVal: fakeLogsExporterType,
		NameVal: fakeLogsExporterName,
	}
)

func TestLogsRequest(t *testing.T) {
	mr := newLogsRequest(context.Background(), testdata.GenerateLogDataOneLog(), nil)

	partialErr := consumererror.PartialLogsError(errors.New("some error"), testdata.GenerateLogDataEmpty())
	assert.EqualValues(
		t,
		newLogsRequest(context.Background(), testdata.GenerateLogDataEmpty(), nil),
		mr.onPartialError(partialErr.(consumererror.PartialError)),
	)
}

func TestLogsExporter_InvalidName(t *testing.T) {
	me, err := NewLogsExporter(nil, zap.NewNop(), newPushLogsData(0, nil))
	require.Nil(t, me)
	require.Equal(t, errNilConfig, err)
}

func TestLogsExporter_NilPushLogsData(t *testing.T) {
	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), nil)
	require.Nil(t, me)
	require.Equal(t, errNilPushLogsData, err)
}

func TestLogsExporter_Default(t *testing.T) {
	ld := testdata.GenerateLogDataEmpty()
	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), newPushLogsData(0, nil))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.Nil(t, me.ConsumeLogs(context.Background(), ld))
	assert.Nil(t, me.Shutdown(context.Background()))
}

func TestLogsExporter_Default_ReturnError(t *testing.T) {
	ld := testdata.GenerateLogDataEmpty()
	want := errors.New("my_error")
	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), newPushLogsData(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeLogs(context.Background(), ld))
}

func TestLogsExporter_WithRecordLogs(t *testing.T) {
	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), newPushLogsData(0, nil))
	require.Nil(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForLogsExporter(t, me, nil)
}

func TestLogsExporter_WithRecordLogs_NonZeroDropped(t *testing.T) {
	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), newPushLogsData(1, nil))
	require.Nil(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForLogsExporter(t, me, nil)
}

func TestLogsExporter_WithRecordLogs_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), newPushLogsData(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForLogsExporter(t, me, want)
}

func TestLogsExporter_WithSpan(t *testing.T) {
	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), newPushLogsData(0, nil))
	require.Nil(t, err)
	require.NotNil(t, me)
	checkWrapSpanForLogsExporter(t, me, nil, 1)
}

func TestLogsExporter_WithSpan_NonZeroDropped(t *testing.T) {
	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), newPushLogsData(1, nil))
	require.Nil(t, err)
	require.NotNil(t, me)
	checkWrapSpanForLogsExporter(t, me, nil, 1)
}

func TestLogsExporter_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), newPushLogsData(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)
	checkWrapSpanForLogsExporter(t, me, want, 1)
}

func TestLogsExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), newPushLogsData(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.Nil(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestLogsExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewLogsExporter(fakeLogsExporterConfig, zap.NewNop(), newPushLogsData(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.Equal(t, me.Shutdown(context.Background()), want)
}

func newPushLogsData(droppedTimeSeries int, retError error) PushLogsData {
	return func(ctx context.Context, td pdata.Logs) (int, error) {
		return droppedTimeSeries, retError
	}
}

func checkRecordedMetricsForLogsExporter(t *testing.T, me component.LogsExporter, wantError error) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	ld := testdata.GenerateLogDataTwoLogsSameResource()
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, me.ConsumeLogs(context.Background(), ld))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		obsreporttest.CheckExporterLogsViews(t, fakeLogsExporterName, 0, int64(numBatches*ld.LogRecordCount()))
	} else {
		obsreporttest.CheckExporterLogsViews(t, fakeLogsExporterName, int64(numBatches*ld.LogRecordCount()), 0)
	}
}

func generateLogsTraffic(t *testing.T, me component.LogsExporter, numRequests int, wantError error) {
	ld := testdata.GenerateLogDataOneLog()
	ctx, span := trace.StartSpan(context.Background(), fakeLogsParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, me.ConsumeLogs(ctx, ld))
	}
}

func checkWrapSpanForLogsExporter(t *testing.T, me component.LogsExporter, wantError error, numLogRecords int64) {
	ocSpansSaver := new(testOCTraceExporter)
	trace.RegisterExporter(ocSpansSaver)
	defer trace.UnregisterExporter(ocSpansSaver)

	const numRequests = 5
	generateLogsTraffic(t, me, numRequests, wantError)

	// Inspection time!
	ocSpansSaver.mu.Lock()
	defer ocSpansSaver.mu.Unlock()

	require.NotEqual(t, 0, len(ocSpansSaver.spanData), "No exported span data")

	gotSpanData := ocSpansSaver.spanData
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeLogsParentSpanName, parentSpan.Name, "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext.SpanID, sd.ParentSpanID, "Exporter span not a child\nSpanData %v", sd)
		require.Equalf(t, errToStatus(wantError), sd.Status, "SpanData %v", sd)

		sentLogRecords := numLogRecords
		var failedToSendLogRecords int64
		if wantError != nil {
			sentLogRecords = 0
			failedToSendLogRecords = numLogRecords
		}
		require.Equalf(t, sentLogRecords, sd.Attributes[obsreport.SentLogRecordsKey], "SpanData %v", sd)
		require.Equalf(t, failedToSendLogRecords, sd.Attributes[obsreport.FailedToSendLogRecordsKey], "SpanData %v", sd)
	}
}
