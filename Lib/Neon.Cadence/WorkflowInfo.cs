﻿//-----------------------------------------------------------------------------
// FILE:	    WorkflowInfo.cs
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
using System.Diagnostics;

using Neon.Cadence;
using Neon.Cadence.Internal;
using Neon.Common;

namespace Neon.Cadence
{
    /// <summary>
    /// Returns information about an executing workflow.
    /// </summary>
    public class WorkflowInfo
    {
        /// <summary>
        /// Returns the workflow domain.
        /// </summary>
        public string Domain { get; internal set; }

        /// <summary>
        /// Returns the workflow ID.
        /// </summary>
        public string WorkflowId { get; internal set; }

        /// <summary>
        /// Returns the workflow's current run ID.
        /// </summary>
        public string RunId { get; internal set; }

        /// <summary>
        /// Returns the workflow's workflow type name.
        /// </summary>
        public string WorkflowType { get; internal set; }

        /// <summary>
        /// Returns the workflow task list.
        /// </summary>
        public string TaskList { get; internal set; }

        /// <summary>
        /// Returns the maximum time the workflow is allowed to run from
        /// the time the workflow was started until it completed.
        /// </summary>
        public TimeSpan ExecutionStartToCloseTimeout { get; internal set; }

        /// <summary>
        /// Returns the workflow's child policy.
        /// </summary>
        public ChildPolicy ChildPolicy { get; internal set; }
    }
}
