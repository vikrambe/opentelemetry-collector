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

package tests

// This file defines parametrized test scenarios and makes them public so that they can be
// also used by tests in custom builds of Collector (e.g. Collector Contrib).

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector/testbed/testbed"
)

// createConfigFile creates a collector config file that corresponds to the
// sender and receiver used in the test and returns the config file name.
func createConfigFile(
	t *testing.T,
	sender testbed.DataSender, // Sender to send test data.
	receiver testbed.DataReceiver, // Receiver to receive test data.
	resultDir string, // Directory to write config file to.

	// Map of processor names to their configs. Config is in YAML and must be
	// indented by 2 spaces. Processors will be placed between batch and queue for traces
	// pipeline. For metrics pipeline these will be sole processors.
	processors map[string]string,
) string {

	// Create a config. Note that our DataSender is used to generate a config for Collector's
	// receiver and our DataReceiver is used to generate a config for Collector's exporter.
	// This is because our DataSender sends to Collector's receiver and our DataReceiver
	// receives from Collector's exporter.

	// Prepare extra processor config section and comma-separated list of extra processor
	// names to use in corresponding "processors" settings.
	processorsSections := ""
	processorsList := ""
	if len(processors) > 0 {
		first := true
		for name, cfg := range processors {
			processorsSections += cfg + "\n"
			if !first {
				processorsList += ","
			}
			processorsList += name
			first = false
		}
	}

	// Set pipeline based on DataSender type
	var pipeline string
	switch sender.(type) {
	case testbed.TraceDataSender, testbed.TraceDataSenderOld:
		pipeline = "traces"
	case testbed.MetricDataSender, testbed.MetricDataSenderOld:
		pipeline = "metrics"
	default:
		t.Error("Invalid DataSender type")
	}

	format := `
receivers:%v
exporters:%v
processors:
  %s

extensions:
  pprof:
    save_to_file: %v/cpu.prof

service:
  extensions: [pprof]
  pipelines:
    %s:
      receivers: [%v]
      processors: [%s]
      exporters: [%v]
`

	// Put corresponding elements into the config template to generate the final config.
	config := fmt.Sprintf(
		format,
		sender.GenConfigYAMLStr(),
		receiver.GenConfigYAMLStr(),
		processorsSections,
		resultDir,
		pipeline,
		sender.ProtocolName(),
		processorsList,
		receiver.ProtocolName(),
	)

	// Write the config string to a temporary file.
	file, err := ioutil.TempFile("", "agent*.yaml")
	if err != nil {
		t.Error(err)
		return ""
	}

	defer func() {
		errClose := file.Close()
		if errClose != nil {
			t.Error(err)
		}
	}()

	if _, err = file.WriteString(config); err != nil {
		t.Error(err)
		return ""
	}

	// Return config file name.
	return file.Name()
}

// Run 10k data items/sec test using specified sender and receiver protocols.
func Scenario10kItemsPerSecond(
	t *testing.T,
	sender testbed.DataSender,
	receiver testbed.DataReceiver,
	resourceSpec testbed.ResourceSpec,
	processors map[string]string,
) {
	resultDir, err := filepath.Abs(path.Join("results", t.Name()))
	if err != nil {
		t.Fatal(err)
	}

	configFile := createConfigFile(t, sender, receiver, resultDir, processors)
	defer os.Remove(configFile)

	if configFile == "" {
		t.Fatal("Cannot create config file")
	}

	tc := testbed.NewTestCase(t, sender, receiver, testbed.WithConfigFile(configFile))
	defer tc.Stop()

	tc.SetResourceLimits(resourceSpec)
	tc.StartBackend()
	tc.StartAgent()

	tc.StartLoad(testbed.LoadOptions{
		DataItemsPerSecond: 10000,
		ItemsPerBatch:      100,
	})

	tc.Sleep(tc.Duration)

	tc.StopLoad()

	tc.WaitFor(func() bool { return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived() },
		"all data items received")

	tc.StopAgent()

	tc.ValidateData()
}

// TestCase for Scenario1kSPSWithAttrs func.
type TestCase struct {
	attrCount      int
	attrSizeByte   int
	expectedMaxCPU uint32
	expectedMaxRAM uint32
}

func genRandByteString(len int) string {
	b := make([]byte, len)
	for i := range b {
		b[i] = byte(rand.Intn(128))
	}
	return string(b)
}

// Scenario1kSPSWithAttrs runs a performance test at 1k sps with specified span attributes
// and test options.
func Scenario1kSPSWithAttrs(t *testing.T, args []string, tests []TestCase, opts ...testbed.TestCaseOption) {
	for i := range tests {
		test := tests[i]
		t.Run(fmt.Sprintf("%d*%dbytes", test.attrCount, test.attrSizeByte), func(t *testing.T) {

			tc := testbed.NewTestCase(
				t,
				testbed.NewJaegerGRPCDataSender(testbed.DefaultJaegerPort),
				testbed.NewOCDataReceiver(testbed.DefaultOCPort),
				opts...,
			)
			defer tc.Stop()

			tc.SetResourceLimits(testbed.ResourceSpec{
				ExpectedMaxCPU: test.expectedMaxCPU,
				ExpectedMaxRAM: test.expectedMaxRAM,
			})

			tc.StartBackend()
			tc.StartAgent(args...)

			options := testbed.LoadOptions{DataItemsPerSecond: 1000}
			options.Attributes = make(map[string]string)

			// Generate attributes.
			for i := 0; i < test.attrCount; i++ {
				attrName := genRandByteString(rand.Intn(199) + 1)
				options.Attributes[attrName] = genRandByteString(rand.Intn(test.attrSizeByte*2-1) + 1)
			}

			tc.StartLoad(options)
			tc.Sleep(tc.Duration)
			tc.StopLoad()

			tc.WaitFor(func() bool { return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived() },
				"all spans received")

			tc.StopAgent()

			tc.ValidateData()
		})
	}
}
