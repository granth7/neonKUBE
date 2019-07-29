//-----------------------------------------------------------------------------
// FILE:	    child_contexts.go
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

package cadenceworkflows

import (
	"sync"

	"go.uber.org/cadence/workflow"
)

var (

	// childID is incremented (protected by a mutex) every
	// time a new cadence workflow.Context is created by a
	// child workflow
	childID int64
)

type (

	// childContextsMap holds a thread-safe map[interface{}]interface{} of
	// cadence childContextsMap with their contextID's
	childContextsMap struct {
		sync.Map
	}

	// ChildContext maps a child workflow contexts to a ChildID.
	// It holds a workflow Context, Future, Settable,
	// and cancellation function
	ChildContext struct {
		ctx        workflow.Context
		future     workflow.ChildWorkflowFuture
		cancelFunc func()
	}
)

//----------------------------------------------------------------------------
// childID methods

// NextChildID increments the global variable
// childID by 1 and is protected by a mutex lock
func NextChildID() int64 {
	mu.Lock()
	childID = childID + 1
	defer mu.Unlock()

	return childID
}

// GetChildID gets the value of the global variable
// childID and is protected by a mutex Read lock
func GetChildID() int64 {
	mu.RLock()
	defer mu.RUnlock()

	return childID
}

//----------------------------------------------------------------------------
// ChildContext instance methods

// NewChildContext is the default constructor
// for a ChildContext struct
//
// returns *ChildContext -> pointer to a newly initialized
// ChildContext in memory
func NewChildContext(ctx workflow.Context) *ChildContext {
	cctx := new(ChildContext)
	cctx.SetContext(ctx)
	return cctx
}

// GetContext gets a ChildContext's workflow.Context
//
// returns workflow.Context -> a cadence workflow context
func (cctx *ChildContext) GetContext() workflow.Context {
	return cctx.ctx
}

// SetContext sets a ChildContext's workflow.Context
//
// param value workflow.Context -> a cadence workflow context to be
// set as a ChildContext's cadence workflow.Context
func (cctx *ChildContext) SetContext(value workflow.Context) {
	cctx.ctx = value
}

// GetCancelFunction gets a ChildContext's context cancel function
//
// returns func() -> a cadence workflow context cancel function
func (cctx *ChildContext) GetCancelFunction() func() {
	return cctx.cancelFunc
}

// SetCancelFunction sets a ChildContext's cancel function
//
// param value func() -> a cadence workflow context cancel function
func (cctx *ChildContext) SetCancelFunction(value func()) {
	cctx.cancelFunc = value
}

// GetFuture gets a ChildContext's workflow.ChildWorkflowFuture
//
// returns workflow.ChildWorkflowFuture -> a cadence workflow.ChildWorkflowFuture
func (cctx *ChildContext) GetFuture() workflow.ChildWorkflowFuture {
	return cctx.future
}

// SetFuture sets a ChildContext's workflow.ChildWorkflowFuture
//
// param value workflow.ChildWorkflowFuture -> a cadence workflow.ChildWorkflowFuture to be
// set as a ChildContext's cadence workflow.ChildWorkflowFuture
func (cctx *ChildContext) SetFuture(value workflow.ChildWorkflowFuture) {
	cctx.future = value
}

//----------------------------------------------------------------------------
// childContextsMap instance methods

// Add adds a new cadence context and its corresponding ContextId into
// the childContextsMap map.  This method is thread-safe.
//
// param id int64 -> the long id passed to Cadence
// child workflow function.  This will be the mapped key
//
// param cctx *ChildContext -> pointer to the new ChildContex used to
// execute child workflow function. This will be the mapped value
//
// returns int64 -> long id of the new cadence ChildContext added to the map
func (cctxs *childContextsMap) Add(id int64, cctx *ChildContext) int64 {
	cctxs.Store(id, cctx)
	return id
}

// Remove removes key/value entry from the childContextsMap map at the specified
// ContextId.  This is a thread-safe method.
//
// param id int64 -> the long id passed to Cadence
// child workflow function.  This will be the mapped key
//
// returns int64 -> long id of the ChildContext removed from the map
func (cctxs *childContextsMap) Remove(id int64) int64 {
	cctxs.Delete(id)
	return id
}

// Get gets a ChildContext from the childContextsMap at the specified
// ContextID.  This method is thread-safe.
//
// param id int64 -> the long id passed to Cadence
// child workflow function.  This will be the mapped key
//
// returns *ChildContext -> pointer to ChildContext with the specified id
func (cctxs *childContextsMap) Get(id int64) *ChildContext {
	if v, ok := cctxs.Load(id); ok {
		if _v, _ok := v.(*ChildContext); _ok {
			return _v
		}
	}

	return nil
}
