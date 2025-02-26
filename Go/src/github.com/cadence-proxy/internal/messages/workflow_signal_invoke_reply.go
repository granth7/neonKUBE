//-----------------------------------------------------------------------------
// FILE:		workflow_signal_invoke_reply.go
// CONTRIBUTOR: John C Burns
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
	messagetypes "github.com/cadence-proxy/internal/messages/types"
)

type (

	// WorkflowSignalInvokeReply is a WorkflowReply of MessageType
	// WorkflowSignalInvokeReply.  It holds a reference to a WorkflowReply in memory
	// and is the reply type to a WorkflowSignalInvokeRequest
	WorkflowSignalInvokeReply struct {
		*WorkflowReply
	}
)

// NewWorkflowSignalInvokeReply is the default constructor for
// a WorkflowSignalInvokeReply
//
// returns *WorkflowSignalInvokeReply -> a pointer to a newly initialized
// WorkflowSignalInvokeReply in memory
func NewWorkflowSignalInvokeReply() *WorkflowSignalInvokeReply {
	reply := new(WorkflowSignalInvokeReply)
	reply.WorkflowReply = NewWorkflowReply()
	reply.SetType(messagetypes.WorkflowSignalInvokeReply)

	return reply
}

// -------------------------------------------------------------------------
// IProxyMessage interface methods for implementing the IProxyMessage interface

// Clone inherits docs from WorkflowReply.Clone()
func (reply *WorkflowSignalInvokeReply) Clone() IProxyMessage {
	workflowSignalInvokeReply := NewWorkflowSignalInvokeReply()
	var messageClone IProxyMessage = workflowSignalInvokeReply
	reply.CopyTo(messageClone)

	return messageClone
}

// CopyTo inherits docs from WorkflowReply.CopyTo()
func (reply *WorkflowSignalInvokeReply) CopyTo(target IProxyMessage) {
	reply.WorkflowReply.CopyTo(target)
}
