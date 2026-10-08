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
	"bytes"
	"testing"

	conformancev1 "connectrpc.com/conformance/internal/gen/proto/go/connectrpc/conformance/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestPadPayload(t *testing.T) {
	t.Parallel()
	for _, size := range []uint32{100, 127, 128, 129, 1024 * 1024} {
		payload := &conformancev1.ConformancePayload{Data: []byte("abc")}
		resp := &conformancev1.UnaryResponse{Payload: payload}
		require.NoError(t, PadPayload(resp, size))
		assert.Equal(t, int(size), proto.Size(resp))
		assert.Equal(t, []byte("abc"), payload.Data)
		assert.Empty(t, bytes.Trim(payload.Padding, "\x00"))
	}
}

func TestPadPayload_ZeroSize(t *testing.T) {
	t.Parallel()
	payload := &conformancev1.ConformancePayload{Data: []byte("abc")}
	resp := &conformancev1.UnaryResponse{Payload: payload}
	require.NoError(t, PadPayload(resp, 0))
	assert.Equal(t, []byte("abc"), payload.Data)
	assert.Empty(t, payload.Padding)
}

func TestPadPayload_TooSmall(t *testing.T) {
	t.Parallel()
	payload := &conformancev1.ConformancePayload{Data: []byte("abcdefgh")}
	resp := &conformancev1.UnaryResponse{Payload: payload}
	require.ErrorContains(t, PadPayload(resp, 4), "message is already")
	assert.Equal(t, []byte("abcdefgh"), payload.Data)
}

func TestPadPayload_Unreachable(t *testing.T) {
	t.Parallel()
	// The unpadded response is 7 bytes, and the first byte of padding adds 3:
	// its tag, its length, and itself.
	resp := &conformancev1.UnaryResponse{Payload: &conformancev1.ConformancePayload{Data: []byte("abc")}}
	require.ErrorContains(t, PadPayload(resp, 8), "can't pad to exactly 8 bytes")
}

func TestPadErrorMessage(t *testing.T) {
	t.Parallel()
	err := &conformancev1.Error{Message: new("oops")}
	PadErrorMessage(err, 10)
	assert.Equal(t, "oopsxxxxxx", err.GetMessage())

	err = &conformancev1.Error{Message: new("already long")}
	PadErrorMessage(err, 4)
	assert.Equal(t, "already long", err.GetMessage())

	err = &conformancev1.Error{}
	PadErrorMessage(err, 0)
	assert.Nil(t, err.Message)
}
