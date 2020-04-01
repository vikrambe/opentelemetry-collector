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

package builder

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

var errUnusedReceiver = errors.New("receiver defined but not used by any pipeline")

// builtReceiver is a receiver that is built based on a config. It can have
// a trace and/or a metrics component.
type builtReceiver struct {
	receiver component.Receiver
}

// Stop the receiver.
func (rcv *builtReceiver) Stop() error {
	return rcv.receiver.Shutdown()
}

// Start the receiver.
func (rcv *builtReceiver) Start(host component.Host) error {
	return rcv.receiver.Start(host)
}

// Receivers is a map of receivers created from receiver configs.
type Receivers map[configmodels.Receiver]*builtReceiver

// StopAll stops all receivers.
func (rcvs Receivers) StopAll() error {
	var errs []error
	for _, rcv := range rcvs {
		err := rcv.Stop()
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return oterr.CombineErrors(errs)
	}
	return nil
}

// StartAll starts all receivers.
func (rcvs Receivers) StartAll(logger *zap.Logger, host component.Host) error {
	for cfg, rcv := range rcvs {
		logger.Info("Receiver is starting...", zap.String("receiver", cfg.Name()))

		if err := rcv.Start(host); err != nil {
			return err
		}
		logger.Info("Receiver started.", zap.String("receiver", cfg.Name()))
	}
	return nil
}

// ReceiversBuilder builds receivers from config.
type ReceiversBuilder struct {
	logger         *zap.Logger
	config         *configmodels.Config
	builtPipelines BuiltPipelines
	factories      map[string]component.ReceiverFactoryBase
}

// NewReceiversBuilder creates a new ReceiversBuilder. Call BuildProcessors() on the returned value.
func NewReceiversBuilder(
	logger *zap.Logger,
	config *configmodels.Config,
	builtPipelines BuiltPipelines,
	factories map[string]component.ReceiverFactoryBase,
) *ReceiversBuilder {
	return &ReceiversBuilder{logger, config, builtPipelines, factories}
}

// BuildProcessors receivers from config.
func (rb *ReceiversBuilder) Build() (Receivers, error) {
	receivers := make(Receivers)

	// BuildProcessors receivers based on configuration.
	for _, cfg := range rb.config.Receivers {
		rcv, err := rb.buildReceiver(cfg)
		if err != nil {
			if err == errUnusedReceiver {
				rb.logger.Info("Ignoring receiver as it is not used by any pipeline", zap.String("receiver", cfg.Name()))
				continue
			}
			return nil, err
		}
		receivers[cfg] = rcv
	}

	return receivers, nil
}

// hasReceiver returns true if the pipeline is attached to specified receiver.
func hasReceiver(pipeline *configmodels.Pipeline, receiverName string) bool {
	for _, name := range pipeline.Receivers {
		if name == receiverName {
			return true
		}
	}
	return false
}

type attachedPipelines map[configmodels.DataType][]*builtPipeline

func (rb *ReceiversBuilder) findPipelinesToAttach(config configmodels.Receiver) (attachedPipelines, error) {
	// A receiver may be attached to multiple pipelines. Pipelines may consume different
	// data types. We need to compile the list of pipelines of each type that must be
	// attached to this receiver according to configuration.

	pipelinesToAttach := make(attachedPipelines)
	pipelinesToAttach[configmodels.TracesDataType] = make([]*builtPipeline, 0)
	pipelinesToAttach[configmodels.MetricsDataType] = make([]*builtPipeline, 0)

	// Iterate over all pipelines.
	for _, pipelineCfg := range rb.config.Service.Pipelines {
		// Get the first processor of the pipeline.
		pipelineProcessor := rb.builtPipelines[pipelineCfg]
		if pipelineProcessor == nil {
			return nil, fmt.Errorf("cannot find pipeline processor for pipeline %s",
				pipelineCfg.Name)
		}

		// Is this receiver attached to the pipeline?
		if hasReceiver(pipelineCfg, config.Name()) {
			// Yes, add it to the list of pipelines of corresponding data type.
			pipelinesToAttach[pipelineCfg.InputType] =
				append(pipelinesToAttach[pipelineCfg.InputType], pipelineProcessor)
		}
	}

	return pipelinesToAttach, nil
}

func (rb *ReceiversBuilder) attachReceiverToPipelines(
	factory component.ReceiverFactoryBase,
	dataType configmodels.DataType,
	config configmodels.Receiver,
	rcv *builtReceiver,
	builtPipelines []*builtPipeline,
) error {
	// There are pipelines of the specified data type that must be attached to
	// the receiver. Create the receiver of corresponding data type and make
	// sure its output is fanned out to all attached pipelines.
	var err error
	var createdReceiver component.Receiver

	switch dataType {
	case configmodels.TracesDataType:
		// First, create the fan out junction point.
		junction := buildFanoutTraceConsumer(builtPipelines)

		// Now create the receiver and tell it to send to the junction point.
		createdReceiver, err = createTraceReceiver(context.Background(), factory, rb.logger, config, junction)

	case configmodels.MetricsDataType:
		junction := buildFanoutMetricConsumer(builtPipelines)
		createdReceiver, err = createMetricsReceiver(context.Background(), factory, rb.logger, config, junction)
	}

	if err != nil {
		if err == configerror.ErrDataTypeIsNotSupported {
			return fmt.Errorf(
				"receiver %s does not support %s but it was used in a "+
					"%s pipeline",
				config.Name(),
				dataType.GetString(),
				dataType.GetString())
		}
		return fmt.Errorf("cannot create receiver %s: %s", config.Name(), err.Error())
	}

	// Check if the factory really created the receiver.
	if createdReceiver == nil {
		return fmt.Errorf("factory for %q produced a nil receiver", config.Name())
	}

	if rcv.receiver != nil {
		// The receiver was previously created for this config. This can happen if the
		// same receiver type supports more than one data type. In that case we expect
		// that CreateTraceReceiver and CreateMetricsReceiver return the same value.
		if rcv.receiver != createdReceiver {
			return fmt.Errorf(
				"factory for %q is implemented incorrectly: "+
					"CreateTraceReceiver and CreateMetricsReceiver must return the same "+
					"receiver pointer when creating receivers of different data types",
				config.Name(),
			)
		}
	}
	rcv.receiver = createdReceiver

	rb.logger.Info("Receiver is enabled.",
		zap.String("receiver", config.Name()), zap.String("datatype", dataType.GetString()))

	return nil
}

func (rb *ReceiversBuilder) buildReceiver(config configmodels.Receiver) (*builtReceiver, error) {

	// First find pipelines that must be attached to this receiver.
	pipelinesToAttach, err := rb.findPipelinesToAttach(config)
	if err != nil {
		return nil, err
	}

	// Prepare to build the receiver.
	factory := rb.factories[config.Type()]
	if factory == nil {
		return nil, fmt.Errorf("receiver factory not found for type: %s", config.Type())
	}
	rcv := &builtReceiver{}

	// Now we have list of pipelines broken down by data type. Iterate for each data type.
	for dataType, pipelines := range pipelinesToAttach {
		if len(pipelines) == 0 {
			// No pipelines of this data type are attached to this receiver.
			continue
		}

		// Attach the corresponding part of the receiver to all pipelines that require
		// this data type.
		err := rb.attachReceiverToPipelines(factory, dataType, config, rcv, pipelines)
		if err != nil {
			return nil, err
		}
	}

	if rcv.receiver == nil {
		return nil, errUnusedReceiver
	}

	return rcv, nil
}

func buildFanoutTraceConsumer(pipelines []*builtPipeline) consumer.TraceConsumerBase {
	// Optimize for the case when there is only one processor, no need to create junction point.
	if len(pipelines) == 1 {
		return pipelines[0].firstTC
	}

	var pipelineConsumers []consumer.TraceConsumerBase
	anyPipelineMutatesData := false
	for _, pipeline := range pipelines {
		pipelineConsumers = append(pipelineConsumers, pipeline.firstTC)
		anyPipelineMutatesData = anyPipelineMutatesData || pipeline.MutatesConsumedData
	}

	// Create a junction point that fans out to all pipelines.
	if anyPipelineMutatesData {
		// If any pipeline mutates data use a cloning fan out connector
		// so that it is safe to modify fanned out data.
		// TODO: if there are more than 2 pipelines only clone data for pipelines that
		// declare the intent to mutate the data. Pipelines that do not mutate the data
		// can consume shared data.
		return processor.CreateTraceCloningFanOutConnector(pipelineConsumers)
	}
	return processor.CreateTraceFanOutConnector(pipelineConsumers)
}

func buildFanoutMetricConsumer(pipelines []*builtPipeline) consumer.MetricsConsumerBase {
	// Optimize for the case when there is only one processor, no need to create junction point.
	if len(pipelines) == 1 {
		return pipelines[0].firstMC
	}

	var pipelineConsumers []consumer.MetricsConsumerBase
	anyPipelineMutatesData := false
	for _, pipeline := range pipelines {
		pipelineConsumers = append(pipelineConsumers, pipeline.firstMC)
		anyPipelineMutatesData = anyPipelineMutatesData || pipeline.MutatesConsumedData
	}

	// Create a junction point that fans out to all pipelines.
	if anyPipelineMutatesData {
		// If any pipeline mutates data use a cloning fan out connector
		// so that it is safe to modify fanned out data.
		// TODO: if there are more than 2 pipelines only clone data for pipelines that
		// declare the intent to mutate the data. Pipelines that do not mutate the data
		// can consume shared data.
		return processor.CreateMetricsCloningFanOutConnector(pipelineConsumers)
	}
	return processor.CreateMetricsFanOutConnector(pipelineConsumers)
}

// createTraceReceiver is a helper function that creates trace receiver based on the current receiver type
// and type of the next consumer.
func createTraceReceiver(
	ctx context.Context,
	factory component.ReceiverFactoryBase,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumerBase,
) (component.TraceReceiver, error) {
	if factoryV2, ok := factory.(component.ReceiverFactory); ok {
		creationParams := component.ReceiverCreateParams{Logger: logger}

		// If both receiver and consumer are of the new type (can manipulate on internal data structure),
		// use ProcessorFactory.CreateTraceReceiver.
		if nextConsumerV2, ok := nextConsumer.(consumer.TraceConsumer); ok {
			return factoryV2.CreateTraceReceiver(ctx, creationParams, cfg, nextConsumerV2)
		}

		// If receiver is of the new type, but downstream consumer is of the old type,
		// use internalToOCTraceConverter compatibility shim.
		traceConverter := consumer.NewInternalToOCTraceConverter(nextConsumer.(consumer.TraceConsumerOld))
		return factoryV2.CreateTraceReceiver(ctx, creationParams, cfg, traceConverter)
	}

	// If both receiver and consumer are of the old type (can manipulate on OC traces only),
	// use Factory.CreateTraceReceiver.
	if nextConsumer, ok := nextConsumer.(consumer.TraceConsumerOld); ok {
		return factory.(component.ReceiverFactoryOld).CreateTraceReceiver(ctx, logger, cfg, nextConsumer)
	}

	// Old type receiver and a new type consumer usecase is not supported.
	// TODO: This case can be supported since we have OC->internal traces translation function
	return nil, errors.New("OC Traces -> internal data format translation is not supported")
}

// createMetricsReceiver is a helper function that creates metric receiver based
// on the current receiver type and type of the next consumer.
func createMetricsReceiver(
	ctx context.Context,
	factory component.ReceiverFactoryBase,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.MetricsConsumerBase,
) (component.MetricsReceiver, error) {
	if factoryV2, ok := factory.(component.ReceiverFactory); ok {
		creationParams := component.ReceiverCreateParams{Logger: logger}

		// If both receiver and consumer are of the new type (can manipulate on internal data structure),
		// use ProcessorFactory.CreateMetricsReceiver.
		if nextConsumerV2, ok := nextConsumer.(consumer.MetricsConsumer); ok {
			return factoryV2.CreateMetricsReceiver(ctx, creationParams, cfg, nextConsumerV2)
		}

		// If receiver is of the new type, but downstream consumer is of the old type,
		// use internalToOCMetricsConverter compatibility shim.
		metricsConverter := consumer.NewInternalToOCMetricsConverter(nextConsumer.(consumer.MetricsConsumerOld))
		return factoryV2.CreateMetricsReceiver(ctx, creationParams, cfg, metricsConverter)
	}

	// If both receiver and consumer are of the old type (can manipulate on OC metrics only),
	// use Factory.CreateMetricsReceiver.
	if nextConsumer, ok := nextConsumer.(consumer.MetricsConsumerOld); ok {
		return factory.(component.ReceiverFactoryOld).CreateMetricsReceiver(logger, cfg, nextConsumer)
	}

	// Old type receiver and a new type consumer usecase is not supported.
	// TODO: This case can be supported once we have OC->internal metrics translation function
	return nil, errors.New("OC Metrics -> internal data format translation is not supported")
}
