//-----------------------------------------------------------------------------
// FILE:		workflow_set_cache_size_reply.go
// CONTRIBUTOR: John C Burnes
// COPYRIGHT:	Copyright (c) 2016-2019 by neonFORGE, LLC.  All rights reserved.
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

package messages

import (
	messagetypes "github.com/loopieio/cadence-proxy/internal/messages/types"
)

type (

	// WorkflowSetCacheSizeReply is a WorkflowReply of MessageType
	// WorkflowSetCacheSizeReply.  It holds a reference to a WorkflowReply in memory
	// and is the reply type to a WorkflowSetCacheSizeRequest
	WorkflowSetCacheSizeReply struct {
		*WorkflowReply
	}
)

// NewWorkflowSetCacheSizeReply is the default constructor for
// a WorkflowSetCacheSizeReply
//
// returns *WorkflowSetCacheSizeReply -> a pointer to a newly initialized
// WorkflowSetCacheSizeReply in memory
func NewWorkflowSetCacheSizeReply() *WorkflowSetCacheSizeReply {
	reply := new(WorkflowSetCacheSizeReply)
	reply.WorkflowReply = NewWorkflowReply()
	reply.SetType(messagetypes.WorkflowSetCacheSizeReply)

	return reply
}

// -------------------------------------------------------------------------
// IProxyMessage interface methods for implementing the IProxyMessage interface

// Clone inherits docs from WorkflowReply.Clone()
func (reply *WorkflowSetCacheSizeReply) Clone() IProxyMessage {
	workflowSetCacheSizeReply := NewWorkflowSetCacheSizeReply()
	var messageClone IProxyMessage = workflowSetCacheSizeReply
	reply.CopyTo(messageClone)

	return messageClone
}

// CopyTo inherits docs from WorkflowReply.CopyTo()
func (reply *WorkflowSetCacheSizeReply) CopyTo(target IProxyMessage) {
	reply.WorkflowReply.CopyTo(target)
}
