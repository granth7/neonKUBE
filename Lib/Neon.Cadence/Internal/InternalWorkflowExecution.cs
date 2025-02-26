﻿//-----------------------------------------------------------------------------
// FILE:	    InternalWorkflowExecution.cs
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

using Newtonsoft.Json;

using Neon.Cadence;
using Neon.Common;

namespace Neon.Cadence.Internal
{
    /// <summary>
    /// <b>INTERNAL USE ONLY:</b> Cadence workflow execution details.
    /// </summary>
    internal class InternalWorkflowExecution
    {
        /// <summary>
        /// The original ID assigned to the workflow.
        /// </summary>
        [JsonProperty(PropertyName = "ID", DefaultValueHandling = DefaultValueHandling.IgnoreAndPopulate)]
        [DefaultValue(null)]
        public string ID { get; set; }

        /// <summary>
        /// The latest ID assigned to the workflow.  Note that this will differ
        /// from <see cref="ID"/> when the workflow has been restarted.
        /// </summary>
        [JsonProperty(PropertyName = "RunID", DefaultValueHandling = DefaultValueHandling.IgnoreAndPopulate)]
        [DefaultValue(null)]
        public string RunID { get; set; }

        /// <summary>
        /// Converts the instance into a public <see cref="WorkflowExecution"/>.
        /// </summary>
        public WorkflowExecution ToPublic()
        {
            return new WorkflowExecution(this.ID, this.RunID);
        }
    }
}
