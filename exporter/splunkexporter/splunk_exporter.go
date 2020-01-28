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

package splunkexporter

import (
	"context"
	"fmt"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumererror"
	"github.com/open-telemetry/opentelemetry-collector/exporter/zipkinexporter"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exporterhelper"

	spandatatranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace/spandata"
)

// NewTraceExporter creates an exporter.TraceExporter that just drops the
// received data and logs debugging messages.
func NewTraceExporter(config configmodels.Exporter, logger *zap.Logger) (exporter.TraceExporter, error) {
	typeLog := zap.String("type", config.Type())
	nameLog := zap.String("name", config.Name())
	cfg := config.(*Config)
	splunk := NewClient(
		nil,
		cfg.HttpEventController,
		cfg.Token,
		cfg.Source,
		cfg.SourceType,
		cfg.SplunkIndex,
	)

	ze, _ := zipkinexporter.CreateZipkinExporter(config)
	return exporterhelper.NewTraceExporter(
		config,
		func(ctx context.Context, td consumerdata.TraceData) (int, error) {
			tbatch := make([]interface{}, len(td.Spans))
			for _, span := range td.Spans {
				sd, err := spandatatranslator.ProtoSpanToOCSpanData(span)
				if err != nil {
					return len(td.Spans), consumererror.Permanent(err)
				}
				zs := ze.ZipkinSpan(td.Node, sd)
				tbatch = append(tbatch, &zs)
			}
			splunk.publish(tbatch)
			fmt.Println("TraceExporter", typeLog, nameLog, zap.Int("#spans", len(td.Spans)))
			//logger.Info("TraceExporter", typeLog, nameLog, zap.Int("#spans", len(td.Spans)))
			// TODO: Add ability to record the received data
			return 0, nil
		},
		exporterhelper.WithTracing(true),
		exporterhelper.WithMetrics(true),
		exporterhelper.WithShutdown(logger.Sync),
	)
}

// NewMetricsExporter creates an exporter.MetricsExporter that just drops the
// received data and logs debugging messages.
func NewMetricsExporter(config configmodels.Exporter, logger *zap.Logger) (exporter.MetricsExporter, error) {
	typeLog := zap.String("type", config.Type())
	nameLog := zap.String("name", config.Name())
	return exporterhelper.NewMetricsExporter(
		config,
		func(ctx context.Context, md consumerdata.MetricsData) (int, error) {
			fmt.Println("MetricsExporter", typeLog, nameLog, zap.Int("#metrics", len(md.Metrics)))
			// TODO: Add ability to record the received data
			return 0, nil
		},
		exporterhelper.WithTracing(true),
		exporterhelper.WithMetrics(true),
		exporterhelper.WithShutdown(logger.Sync),
	)
}
