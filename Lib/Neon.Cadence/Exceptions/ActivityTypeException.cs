﻿//-----------------------------------------------------------------------------
// FILE:	    ActivityTypeException.cs
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

using Neon.Cadence.Internal;

namespace Neon.Cadence
{
    /// <summary>
    /// Thrown when ak activity interface or implementation is not valid.
    /// </summary>
    public class ActivityTypeException : Exception
    {
        /// <summary>
        /// Default constructor.
        /// </summary>
        public ActivityTypeException()
        {
        }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="message">The message.</param>
        /// <param name="innerException">Optionally specifies an inner exception.</param>
        public ActivityTypeException(string message, Exception innerException = null)
            : base(message, innerException)
        {
        }
    }
}
