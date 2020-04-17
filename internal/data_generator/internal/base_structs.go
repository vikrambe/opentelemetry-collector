// Copyright 2020 OpenTelemetry Authors
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

package internal

import (
	"os"
	"strings"
)

const sliceTemplate = `// ${structName} logically represents a slice of ${elementName}.
//
// This is a reference type, if passsed by value and callee modifies it the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} struct {
	// orig points to the slice ${originName} field contained somewhere else.
	// We use pointer-to-slice to be able to modify it in functions like Resize.
	orig *[]*${originName}
}

func new${structName}(orig *[]*${originName}) ${structName} {
	return ${structName}{orig}
}

// New${structName} creates a ${structName} with 0 elements.
// Can use "Resize" to initialize with a given length.
func New${structName}() ${structName} {
	orig := []*${originName}(nil)
	return ${structName}{&orig}
}

// Len returns the number of elements in the slice.
//
// Returns "0" for a newly instance created with "New${structName}()".
func (es ${structName}) Len() int {
	return len(*es.orig)
}

// At returns the element at the given index.
//
// This function is used mostly for iterating over all the values in the slice:
// for i := 0; i < es.Len(); i++ {
//     e := es.At(i)
//     ... // Do something with the element
// }
func (es ${structName}) At(ix int) ${elementName} {
	return new${elementName}(&(*es.orig)[ix])
}

// MoveTo moves all elements from the current slice to the dest. The current slice will be cleared.
func (es ${structName}) MoveTo(dest ${structName}) {
	if es.Len() == 0 {
		// Just to ensure that we always return a Slice with nil elements.
		*es.orig = nil
		return
	}
	if dest.Len() == 0 {
		*dest.orig = *es.orig
		*es.orig = nil
		return
	}
	*dest.orig = append(*dest.orig, *es.orig...)
	*es.orig = nil
	return
}

// Resize is an operation that resizes the slice:
// 1. If newLen is 0 then the slice is replaced with a nil slice.
// 2. If the newLen < len then equivalent with slice[0:newLen].
// 3. If the newLen > len then (newLen - len) empty elements will be appended to the slice.
//
// Here is how a new ${structName} can be initialized:
// es := New${structName}()
// es.Resize(4)
// for i := 0; i < es.Len(); i++ {
//     e := es.At(i)
//     // Here should set all the values for e.
// }
func (es ${structName}) Resize(newLen int) {
	if newLen == 0 {
		(*es.orig) = []*${originName}(nil)
		return
	}
	oldLen := len(*es.orig)
	if newLen < oldLen {
		(*es.orig) = (*es.orig)[0:newLen]
		return
	}
	// TODO: Benchmark and optimize this logic.
	extraOrigs := make([]${originName}, newLen-oldLen)
	oldOrig := (*es.orig)
	for i := range extraOrigs {
		oldOrig = append(oldOrig, &extraOrigs[i])
	}
	(*es.orig) = oldOrig
}`

const sliceTestTemplate = `func Test${structName}(t *testing.T) {
	es := New${structName}()
	assert.EqualValues(t, 0, es.Len())
	es = new${structName}(&[]*${originName}{})
	assert.EqualValues(t, 0, es.Len())

	es.Resize(7)
	emptyVal := New${elementName}()
	emptyVal.InitEmpty()
	testVal := generateTest${elementName}()
	assert.EqualValues(t, 7, es.Len())
	for i := 0; i < es.Len(); i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
		fillTest${elementName}(es.At(i))
		assert.EqualValues(t, testVal, es.At(i))
	}
}

func Test${structName}MoveTo(t *testing.T) {
	// Test MoveTo to empty
	expectedSlice := generateTest${structName}()
	dest := New${structName}()
	src := generateTest${structName}()
	src.MoveTo(dest)
	assert.EqualValues(t, generateTest${structName}(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveTo empty slice
	src.MoveTo(dest)
	assert.EqualValues(t, generateTest${structName}(), dest)
	assert.EqualValues(t, 0, src.Len())
	assert.EqualValues(t, expectedSlice.Len(), dest.Len())

	// Test MoveTo not empty slice
	generateTest${structName}().MoveTo(dest)
	assert.EqualValues(t, 2*expectedSlice.Len(), dest.Len())
	for i := 0; i < expectedSlice.Len(); i++ {
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i))
		assert.EqualValues(t, expectedSlice.At(i), dest.At(i+expectedSlice.Len()))
	}
}

func Test${structName}Resize(t *testing.T) {
	es := generateTest${structName}()
	emptyVal := New${elementName}()
	emptyVal.InitEmpty()
	// Test Resize less elements.
	const resizeSmallLen = 4
	expectedEs := make(map[*${originName}]bool, resizeSmallLen)
	for i := 0; i < resizeSmallLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, resizeSmallLen, len(expectedEs))
	es.Resize(resizeSmallLen)
	assert.EqualValues(t, resizeSmallLen, es.Len())
	foundEs := make(map[*${originName}]bool, resizeSmallLen)
	for i := 0; i < es.Len(); i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)

	// Test Resize more elements.
	const resizeLargeLen = 7
	oldLen := es.Len()
	expectedEs = make(map[*${originName}]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		expectedEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, oldLen, len(expectedEs))
	es.Resize(resizeLargeLen)
	assert.EqualValues(t, resizeLargeLen, es.Len())
	foundEs = make(map[*${originName}]bool, oldLen)
	for i := 0; i < oldLen; i++ {
		foundEs[*(es.At(i).orig)] = true
	}
	assert.EqualValues(t, expectedEs, foundEs)
	for i := oldLen; i < resizeLargeLen; i++ {
		assert.EqualValues(t, emptyVal, es.At(i))
	}

	// Test Resize 0 elements.
	es.Resize(0)
	assert.EqualValues(t, New${structName}(), es)
}`

const sliceGenerateTest = `func generateTest${structName}() ${structName} {
	tv := New${structName}()
	fillTest${structName}(tv)
	return tv
}

func fillTest${structName}(tv ${structName}) {
	tv.Resize(7)
	for i := 0; i < tv.Len(); i++ {
		fillTest${elementName}(tv.At(i))
	}
}`

const messageTemplate = `${description}
//
// This is a reference type, if passsed by value and callee modifies it the
// caller will see the modification.
//
// Must use New${structName} function to create new instances.
// Important: zero-initialized instance is not valid for use.
type ${structName} struct {
	// orig points to the pointer ${originName} field contained somewhere else.
	// We use pointer-to-pointer to be able to modify it in InitEmpty func.
	orig **${originName}
}

func new${structName}(orig **${originName}) ${structName} {
	return ${structName}{orig}
}

// New${structName} creates a new "nil" ${structName}.
// To initialize the struct call "InitEmpty".
//
// This must be used only in testing code since no "Set" method available.
func New${structName}() ${structName} {
	orig := (*${originName})(nil)
	return new${structName}(&orig)
}

// InitEmpty overwrites the current value with empty.
func (ms ${structName}) InitEmpty() {
	*ms.orig = &${originName}{}
}

// IsNil returns true if the underlying data are nil.
//
// Important: All other functions will cause a runtime error if this returns "true".
func (ms ${structName}) IsNil() bool {
	return *ms.orig == nil
}`

const messageTestHeaderTemplate = `func Test${structName}(t *testing.T) {
	ms := New${structName}()
	assert.EqualValues(t, true, ms.IsNil())
	ms.InitEmpty()
	assert.EqualValues(t, false, ms.IsNil())`

const messageTestFooterTemplate = `	assert.EqualValues(t, generateTest${structName}(), ms)
}`

const messageGenerateTestTemplate = `func generateTest${structName}() ${structName} {
	tv := New${structName}()
	tv.InitEmpty()
	fillTest${structName}(tv)
	return tv
}`

const messageFillTestHeaderTemplate = `func fillTest${structName}(tv ${structName}) {`
const messageFillTestFooterTemplate = `}`

const newLine = "\n"

type baseStruct interface {
	generateStruct(sb *strings.Builder)

	generateTests(sb *strings.Builder)

	generateTestValueHelpers(sb *strings.Builder)
}

// Will generate code only for the slice struct.
type sliceStruct struct {
	structName string
	element    *messageStruct
}

func (ss *sliceStruct) generateStruct(sb *strings.Builder) {
	sb.WriteString(os.Expand(sliceTemplate, func(name string) string {
		switch name {
		case "structName":
			return ss.structName
		case "elementName":
			return ss.element.structName
		case "originName":
			return ss.element.originFullName
		default:
			panic(name)
		}
	}))
}

func (ss *sliceStruct) generateTests(sb *strings.Builder) {
	sb.WriteString(os.Expand(sliceTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ss.structName
		case "elementName":
			return ss.element.structName
		case "originName":
			return ss.element.originFullName
		default:
			panic(name)
		}
	}))
}

func (ss *sliceStruct) generateTestValueHelpers(sb *strings.Builder) {
	sb.WriteString(os.Expand(sliceGenerateTest, func(name string) string {
		switch name {
		case "structName":
			return ss.structName
		case "elementName":
			return ss.element.structName
		default:
			panic(name)
		}
	}))
}

var _ baseStruct = (*sliceStruct)(nil)

type messageStruct struct {
	structName     string
	description    string
	originFullName string
	fields         []baseField
}

func (ms *messageStruct) generateStruct(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "originName":
			return ms.originFullName
		case "description":
			return ms.description
		default:
			panic(name)
		}
	}))
	// Write accessors for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine + newLine)
		f.generateAccessors(ms, sb)
	}
}

func (ms *messageStruct) generateTests(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageTestHeaderTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "originName":
			return ms.originFullName
		default:
			panic(name)
		}
	}))
	// Write accessors tests for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine + newLine)
		f.generateAccessorsTests(ms, sb)
	}
	sb.WriteString(newLine + newLine)
	sb.WriteString(os.Expand(messageTestFooterTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		default:
			panic(name)
		}
	}))
}

func (ms *messageStruct) generateTestValueHelpers(sb *strings.Builder) {
	sb.WriteString(os.Expand(messageGenerateTestTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		case "originName":
			return ms.originFullName
		default:
			panic(name)
		}
	}))

	sb.WriteString(newLine + newLine)
	sb.WriteString(os.Expand(messageFillTestHeaderTemplate, func(name string) string {
		switch name {
		case "structName":
			return ms.structName
		default:
			panic(name)
		}
	}))
	// Write accessors test value for the struct
	for _, f := range ms.fields {
		sb.WriteString(newLine)
		f.generateSetWithTestValue(sb)
	}
	sb.WriteString(newLine)
	sb.WriteString(os.Expand(messageFillTestFooterTemplate, func(name string) string {
		switch name {
		default:
			panic(name)
		}
	}))
}

var _ baseStruct = (*messageStruct)(nil)
