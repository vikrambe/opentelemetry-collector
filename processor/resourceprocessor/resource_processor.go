// Copyright 2019 OpenTelemetry Authors
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

package resourceprocessor

import (
	"context"

	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

type resourceTraceProcessor struct {
	resource     *resourcepb.Resource
	capabilities component.ProcessorCapabilities
	next         consumer.TraceConsumerOld
}

func newResourceTraceProcessor(next consumer.TraceConsumerOld, cfg *Config) *resourceTraceProcessor {
	resource := createResource(cfg)
	return &resourceTraceProcessor{
		next:         next,
		capabilities: component.ProcessorCapabilities{MutatesConsumedData: !isEmptyResource(resource)},
		resource:     resource,
	}
}

// ConsumeTraceData implements the TraceProcessor interface
func (rtp *resourceTraceProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	return rtp.next.ConsumeTraceData(ctx, consumerdata.TraceData{
		Node:         td.Node,
		Resource:     mergeResource(td.Resource, rtp.resource),
		Spans:        td.Spans,
		SourceFormat: td.SourceFormat,
	})
}

// GetCapabilities returns the ProcessorCapabilities assocciated with the resource processor.
func (rtp *resourceTraceProcessor) GetCapabilities() component.ProcessorCapabilities {
	return rtp.capabilities
}

// Start is invoked during service startup.
func (*resourceTraceProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (*resourceTraceProcessor) Shutdown(context.Context) error {
	return nil
}

type resourceMetricProcessor struct {
	resource     *resourcepb.Resource
	capabilities component.ProcessorCapabilities
	next         consumer.MetricsConsumerOld
}

func newResourceMetricProcessor(next consumer.MetricsConsumerOld, cfg *Config) *resourceMetricProcessor {
	resource := createResource(cfg)
	return &resourceMetricProcessor{
		resource:     resource,
		capabilities: component.ProcessorCapabilities{MutatesConsumedData: !isEmptyResource(resource)},
		next:         next,
	}
}

// GetCapabilities returns the ProcessorCapabilities assocciated with the resource processor.
func (rmp *resourceMetricProcessor) GetCapabilities() component.ProcessorCapabilities {
	return rmp.capabilities
}

// Start is invoked during service startup.
func (*resourceMetricProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (*resourceMetricProcessor) Shutdown(context.Context) error {
	return nil
}

// ConsumeMetricsData implements the MetricsProcessor interface
func (rmp *resourceMetricProcessor) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	return rmp.next.ConsumeMetricsData(ctx, consumerdata.MetricsData{
		Node:     md.Node,
		Resource: mergeResource(md.Resource, rmp.resource),
		Metrics:  md.Metrics,
	})
}

func createResource(cfg *Config) *resourcepb.Resource {
	rpb := &resourcepb.Resource{
		Type:   cfg.ResourceType,
		Labels: map[string]string{},
	}
	for k, v := range cfg.Labels {
		rpb.Labels[k] = v
	}
	return rpb
}

func mergeResource(to, from *resourcepb.Resource) *resourcepb.Resource {
	if isEmptyResource(from) {
		return to
	}
	if to == nil {
		if from.Type == "" {
			// Since resource without type would be invalid, we keep resource as nil
			return nil
		}
		to = &resourcepb.Resource{Labels: map[string]string{}}
	}
	if from.Type != "" {
		// Only change resource type if it was configured
		to.Type = from.Type
	}
	if from.Labels != nil {
		for k, v := range from.Labels {
			to.Labels[k] = v
		}
	}
	return to
}

func isEmptyResource(resource *resourcepb.Resource) bool {
	return resource.Type == "" && len(resource.Labels) == 0
}
