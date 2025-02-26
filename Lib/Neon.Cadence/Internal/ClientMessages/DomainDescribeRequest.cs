﻿//-----------------------------------------------------------------------------
// FILE:	    DomainDescribeRequest.cs
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
    /// <b>client --> proxy:</b> Requests the details for a named domain.
    /// </summary>
    [InternalProxyMessage(InternalMessageTypes.DomainDescribeRequest)]
    internal class DomainDescribeRequest : ProxyRequest
    {
        /// <summary>
        /// Default constructor.
        /// </summary>
        public DomainDescribeRequest()
        {
            Type = InternalMessageTypes.DomainDescribeRequest;
        }

        /// <inheritdoc/>
        public override InternalMessageTypes ReplyType => InternalMessageTypes.DomainDescribeReply;

        /// <summary>
        /// <para>
        /// The target Cadence domain name. (or <c>null</c>).
        /// </para>
        /// <note>
        /// One of <see cref="Name"/> or <see cref="Uuid"/> must be non-null and non-empty.
        /// </note>
        /// </summary>
        public string Name
        {
            get => GetStringProperty(PropertyNames.Name);
            set => SetStringProperty(PropertyNames.Name, value);
        }

        /// <summary>
        /// <para>
        /// The target Cadence domain UUID (or <c>null</c>).
        /// </para>
        /// <note>
        /// One of <see cref="Name"/> or <see cref="Uuid"/> must be non-null and non-empty.
        /// </note>
        /// </summary>
        public string Uuid
        {
            get => GetStringProperty(PropertyNames.Uuid);
            set => SetStringProperty(PropertyNames.Uuid, value);
        }

        /// <inheritdoc/>
        internal override ProxyMessage Clone()
        {
            var clone = new DomainDescribeRequest();

            CopyTo(clone);

            return clone;
        }

        /// <inheritdoc/>
        protected override void CopyTo(ProxyMessage target)
        {
            base.CopyTo(target);

            var typedTarget = (DomainDescribeRequest)target;

            typedTarget.Name = this.Name;
            typedTarget.Uuid = this.Uuid;
        }
    }
}
