﻿//-----------------------------------------------------------------------------
// FILE:	    InternalWorkflowCloseStatus.cs
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
    /// Enumerates the states for a pending activity.
    /// </summary>
    internal enum InternalWorkflowCloseStatus
    {
        // WARNING: These values must match those defined by [WorkflowCloseStatus].

        /// <summary>
        /// The workflow completed successfully.
        /// </summary>
        COMPLETED = 0,

        /// <summary>
        /// The workflow failed.
        /// </summary>
        FAILED = 1,

        /// <summary>
        /// The workflow was cancelled.
        /// </summary>
        CANCELLED = 2,

        /// <summary>
        /// The workflow was terminated.
        /// </summary>
        TERMINATED = 3,

        /// <summary>
        /// The workflow was restarted (aka <i>continued as new</i>).
        /// </summary>
        CONTINUED_AS_NEW = 4,

        /// <summary>
        /// The workflow timed out.
        /// </summary>
        TIMED_OUT = 5
    }
}
