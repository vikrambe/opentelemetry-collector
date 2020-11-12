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

package memorylimiter

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/processor/memorylimiter/internal/iruntime"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

func TestNew(t *testing.T) {
	type args struct {
		nextConsumer        consumer.TracesConsumer
		checkInterval       time.Duration
		memoryLimitMiB      uint32
		memorySpikeLimitMiB uint32
	}
	sink := new(consumertest.TracesSink)
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "zero_checkInterval",
			args: args{
				nextConsumer: sink,
			},
			wantErr: errCheckIntervalOutOfRange,
		},
		{
			name: "zero_memAllocLimit",
			args: args{
				nextConsumer:  sink,
				checkInterval: 100 * time.Millisecond,
			},
			wantErr: errLimitOutOfRange,
		},
		{
			name: "memSpikeLimit_gt_memAllocLimit",
			args: args{
				nextConsumer:        sink,
				checkInterval:       100 * time.Millisecond,
				memoryLimitMiB:      1,
				memorySpikeLimitMiB: 2,
			},
			wantErr: errMemSpikeLimitOutOfRange,
		},
		{
			name: "success",
			args: args{
				nextConsumer:   sink,
				checkInterval:  100 * time.Millisecond,
				memoryLimitMiB: 1024,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createDefaultConfig().(*Config)
			cfg.CheckInterval = tt.args.checkInterval
			cfg.MemoryLimitMiB = tt.args.memoryLimitMiB
			cfg.MemorySpikeLimitMiB = tt.args.memorySpikeLimitMiB
			got, err := newMemoryLimiter(zap.NewNop(), cfg)
			if err != tt.wantErr {
				t.Errorf("newMemoryLimiter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				assert.NoError(t, got.shutdown(context.Background()))
			}
		})
	}
}

// TestMetricsMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestMetricsMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &memoryLimiter{
		decision: dropDecision{
			memAllocLimit: 1024,
		},
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
	}
	mp, err := processorhelper.NewMetricsProcessor(
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: typeStr,
				NameVal: typeStr,
			},
		},
		consumertest.NewMetricsNop(),
		ml,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	md := pdata.NewMetrics()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.memCheck()
	assert.NoError(t, mp.ConsumeMetrics(ctx, md))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.memCheck()
	assert.Equal(t, errForcedDrop, mp.ConsumeMetrics(ctx, md))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.memCheck()
	assert.NoError(t, mp.ConsumeMetrics(ctx, md))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.memCheck()
	assert.Equal(t, errForcedDrop, mp.ConsumeMetrics(ctx, md))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.decision.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.memCheck()
	assert.NoError(t, mp.ConsumeMetrics(ctx, md))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.memCheck()
	assert.Equal(t, errForcedDrop, mp.ConsumeMetrics(ctx, md))

}

// TestTraceMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestTraceMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &memoryLimiter{
		decision: dropDecision{
			memAllocLimit: 1024,
		},
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
	}
	tp, err := processorhelper.NewTraceProcessor(
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: typeStr,
				NameVal: typeStr,
			},
		},
		consumertest.NewTracesNop(),
		ml,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	td := pdata.NewTraces()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.memCheck()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.memCheck()
	assert.Equal(t, errForcedDrop, tp.ConsumeTraces(ctx, td))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.memCheck()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.memCheck()
	assert.Equal(t, errForcedDrop, tp.ConsumeTraces(ctx, td))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.decision.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.memCheck()
	assert.NoError(t, tp.ConsumeTraces(ctx, td))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.memCheck()
	assert.Equal(t, errForcedDrop, tp.ConsumeTraces(ctx, td))

}

// TestLogMemoryPressureResponse manipulates results from querying memory and
// check expected side effects.
func TestLogMemoryPressureResponse(t *testing.T) {
	var currentMemAlloc uint64
	ml := &memoryLimiter{
		decision: dropDecision{
			memAllocLimit: 1024,
		},
		readMemStatsFn: func(ms *runtime.MemStats) {
			ms.Alloc = currentMemAlloc
		},
	}
	lp, err := processorhelper.NewLogsProcessor(
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: typeStr,
				NameVal: typeStr,
			},
		},
		consumertest.NewLogsNop(),
		ml,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(ml.shutdown))
	require.NoError(t, err)

	ctx := context.Background()
	ld := pdata.NewLogs()

	// Below memAllocLimit.
	currentMemAlloc = 800
	ml.memCheck()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memAllocLimit.
	currentMemAlloc = 1800
	ml.memCheck()
	assert.Equal(t, errForcedDrop, lp.ConsumeLogs(ctx, ld))

	// Check ballast effect
	ml.ballastSize = 1000

	// Below memAllocLimit accounting for ballast.
	currentMemAlloc = 800 + ml.ballastSize
	ml.memCheck()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memAllocLimit even accountiing for ballast.
	currentMemAlloc = 1800 + ml.ballastSize
	ml.memCheck()
	assert.Equal(t, errForcedDrop, lp.ConsumeLogs(ctx, ld))

	// Restore ballast to default.
	ml.ballastSize = 0

	// Check spike limit
	ml.decision.memSpikeLimit = 512

	// Below memSpikeLimit.
	currentMemAlloc = 500
	ml.memCheck()
	assert.NoError(t, lp.ConsumeLogs(ctx, ld))

	// Above memSpikeLimit.
	currentMemAlloc = 550
	ml.memCheck()
	assert.Equal(t, errForcedDrop, lp.ConsumeLogs(ctx, ld))
}

func TestGetDecision(t *testing.T) {
	t.Run("fixed_limit", func(t *testing.T) {
		d, err := getDecision(&Config{MemoryLimitMiB: 100, MemorySpikeLimitMiB: 20}, zap.NewNop())
		require.NoError(t, err)
		assert.Equal(t, &dropDecision{
			memAllocLimit: 100 * mibBytes,
			memSpikeLimit: 20 * mibBytes,
		}, d)
	})
	t.Run("fixed_limit_error", func(t *testing.T) {
		d, err := getDecision(&Config{MemoryLimitMiB: 20, MemorySpikeLimitMiB: 100}, zap.NewNop())
		require.Error(t, err)
		assert.Nil(t, d)
	})

	t.Cleanup(func() {
		getMemoryFn = iruntime.TotalMemory
	})
	getMemoryFn = func() (int64, error) {
		return 100 * mibBytes, nil
	}
	t.Run("percentage_limit", func(t *testing.T) {
		d, err := getDecision(&Config{MemoryLimitPercentage: 50, MemorySpikePercentage: 10}, zap.NewNop())
		require.NoError(t, err)
		assert.Equal(t, &dropDecision{
			memAllocLimit: 50 * mibBytes,
			memSpikeLimit: 10 * mibBytes,
		}, d)
	})
	t.Run("percentage_limit_error", func(t *testing.T) {
		d, err := getDecision(&Config{MemoryLimitPercentage: 101, MemorySpikePercentage: 10}, zap.NewNop())
		require.Error(t, err)
		assert.Nil(t, d)
		d, err = getDecision(&Config{MemoryLimitPercentage: 99, MemorySpikePercentage: 101}, zap.NewNop())
		require.Error(t, err)
		assert.Nil(t, d)
	})
}

func TestDropDecision(t *testing.T) {
	decison1000Limit30Spike30, err := newPercentageDecision(1000, 60, 30)
	require.NoError(t, err)
	decison1000Limit60Spike50, err := newPercentageDecision(1000, 60, 50)
	require.NoError(t, err)
	decison1000Limit40Spike20, err := newPercentageDecision(1000, 40, 20)
	require.NoError(t, err)
	decison1000Limit40Spike60, err := newPercentageDecision(1000, 40, 60)
	require.Error(t, err)
	assert.Nil(t, decison1000Limit40Spike60)

	tests := []struct {
		name       string
		decision   dropDecision
		ms         *runtime.MemStats
		shouldDrop bool
	}{
		{
			name:       "should drop over limit",
			decision:   *decison1000Limit30Spike30,
			ms:         &runtime.MemStats{Alloc: 600},
			shouldDrop: true,
		},
		{
			name:       "should not drop",
			decision:   *decison1000Limit30Spike30,
			ms:         &runtime.MemStats{Alloc: 100},
			shouldDrop: false,
		},
		{
			name: "should not drop spike, fixed decision",
			decision: dropDecision{
				memAllocLimit: 600,
				memSpikeLimit: 500,
			},
			ms:         &runtime.MemStats{Alloc: 300},
			shouldDrop: true,
		},
		{
			name:       "should drop, spike, percentage decision",
			decision:   *decison1000Limit60Spike50,
			ms:         &runtime.MemStats{Alloc: 300},
			shouldDrop: true,
		},
		{
			name:       "should drop, spike, percentage decision",
			decision:   *decison1000Limit40Spike20,
			ms:         &runtime.MemStats{Alloc: 250},
			shouldDrop: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shouldDrop := test.decision.shouldDrop(test.ms)
			assert.Equal(t, test.shouldDrop, shouldDrop)
		})
	}
}
