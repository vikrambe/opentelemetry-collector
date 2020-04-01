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

package component

import "context"

// Component is either a receiver, exporter, processor or extension.
type Component interface {
	// Start tells the component to start. Host parameter can be used for communicating
	// with the host after Start() has already returned. If error is returned by
	// Start() then the collector startup will be aborted.
	// If this is an exporter component it may prepare for exporting
	// by connecting to the endpoint.
	Start(host Host) error

	// Shutdown is invoked during service shutdown.
	Shutdown() error
}

// Host represents the entity that is hosting a Component. It is used to allow communication
// between the Component and its host (normally the service.Application is the host).
type Host interface {
	// ReportFatalError is used to report to the host that the extension
	// encountered a fatal error (i.e.: an error that the instance can't recover
	// from) after its start function had already returned.
	ReportFatalError(err error)

	// Context returns a context provided by the host to be used on the component
	// operations.
	Context() context.Context
}

// Factory interface must be implemented by all component factories.
type Factory interface {
	// Type gets the type of the component created by this factory.
	Type() string
}
