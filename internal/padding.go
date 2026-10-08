// Copyright 2023-2024 The Connect Authors
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
	"fmt"
	"slices"
	"strings"

	conformancev1 "connectrpc.com/conformance/internal/gen/proto/go/connectrpc/conformance/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// PadBytesField appends zero bytes to the bytes field of holder, which is msg
// or a message nested in it, so that msg serializes to exactly size bytes.
//
// Each byte of padding adds at least one byte to the size, and more when it
// lengthens a varint: the field's length, the length of a message enclosing
// it, or, for a field that was empty, its tag and length. So the padding
// length is adjusted until the size matches, and some sizes can't be reached.
func PadBytesField(msg proto.Message, holder protoreflect.Message, field protoreflect.FieldDescriptor, size int64) error {
	unpadded := int64(proto.Size(msg))
	if unpadded >= size {
		if unpadded == size {
			return nil
		}
		return fmt.Errorf("can't pad to %d bytes; message is already %d bytes", size, unpadded)
	}
	// TODO: Do we care if the padding is highly compressible? We'll assume not
	//       and use zero values for now.
	data := holder.Get(field).Bytes()
	// The padding never needs to be longer than the shortfall, so allocate that
	// once and shorten it as needed.
	padded := append(slices.Clip(data), make([]byte, size-unpadded)...)
	padding := len(padded) - len(data)
	got := unpadded
	const maxAdjustments = 4
	for range maxAdjustments {
		holder.Set(field, protoreflect.ValueOfBytes(padded[:len(data)+padding]))
		got = int64(proto.Size(msg))
		if got == size {
			return nil
		}
		padding = min(max(padding+int(size-got), 0), len(padded)-len(data))
	}
	return fmt.Errorf("can't pad to exactly %d bytes; closest we can get is %d", size, got)
}

// payloadMessage is a response message that carries a conformance payload.
type payloadMessage interface {
	proto.Message
	GetPayload() *conformancev1.ConformancePayload
}

// PadPayload sets the padding of the payload of resp to zero bytes so that resp
// serializes to exactly size bytes. A size of zero leaves resp unchanged.
func PadPayload(resp payloadMessage, size uint32) error {
	if size == 0 {
		return nil
	}
	holder := resp.GetPayload().ProtoReflect()
	field := holder.Descriptor().Fields().ByName("padding")
	if err := PadBytesField(resp, holder, field, int64(size)); err != nil {
		return fmt.Errorf("pad response: %w", err)
	}
	return nil
}

// PadErrorMessage pads the message of err with "x" characters so that it is
// size bytes long. A message that is already at least that long is left
// unchanged.
func PadErrorMessage(err *conformancev1.Error, size uint32) {
	if pad := int(size) - len(err.GetMessage()); pad > 0 {
		err.Message = new(err.GetMessage() + strings.Repeat("x", pad))
	}
}

// ResponseSize returns the size to which the response at index i of def is
// padded, or zero if it isn't padded.
func ResponseSize(def *conformancev1.StreamResponseDefinition, i int) uint32 {
	if sizes := def.GetResponseSizes(); i < len(sizes) {
		return sizes[i]
	}
	return 0
}
