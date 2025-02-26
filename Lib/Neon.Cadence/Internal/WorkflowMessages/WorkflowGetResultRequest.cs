﻿//-----------------------------------------------------------------------------
// FILE:	    WorkflowGetResultRequest.cs
// CONTRIBUTOR: Jeff Lill
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

using System;
using System.Collections.Generic;
using System.ComponentModel;

using Neon.Cadence;
using Neon.Common;

namespace Neon.Cadence.Internal
{
    /// <summary>
    /// <b>proxy --> client:</b> Requests the results from a workflow execution.
    /// </summary>
    [InternalProxyMessage(InternalMessageTypes.WorkflowGetResultRequest)]
    internal class WorkflowGetResultRequest : WorkflowRequest
    {
        /// <summary>
        /// Default constructor.
        /// </summary>
        public WorkflowGetResultRequest()
        {
            Type = InternalMessageTypes.WorkflowGetResultRequest;
        }

        /// <inheritdoc/>
        public override InternalMessageTypes ReplyType => InternalMessageTypes.WorkflowGetResultReply;

        /// <summary>
        /// Identifies the workflow by ID.
        /// </summary>
        public string WorkflowId
        {
            get => GetStringProperty(PropertyNames.WorkflowId);
            set => SetStringProperty(PropertyNames.WorkflowId, value);
        }

        /// <summary>
        /// Identifies the specific workflow execution.
        /// </summary>
        public string RunId
        {
            get => GetStringProperty(PropertyNames.RunId);
            set => SetStringProperty(PropertyNames.RunId, value);
        }

        /// <summary>
        /// Identifies the target domain.
        /// </summary>
        public string Domain
        {
            get => GetStringProperty(PropertyNames.Domain);
            set => SetStringProperty(PropertyNames.Domain, value);
        }

        /// <inheritdoc/>
        internal override ProxyMessage Clone()
        {
            var clone = new WorkflowGetResultRequest();

            CopyTo(clone);

            return clone;
        }

        /// <inheritdoc/>
        protected override void CopyTo(ProxyMessage target)
        {
            base.CopyTo(target);

            var typedTarget = (WorkflowGetResultRequest)target;

            typedTarget.WorkflowId = this.WorkflowId;
            typedTarget.RunId      = this.RunId;
            typedTarget.Domain     = this.Domain;
        }
    }
}
