﻿//-----------------------------------------------------------------------------
// FILE:	    ServicePlacementSpreadSettings.cs
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
using System.Runtime.Serialization;
using System.Text;

using Newtonsoft.Json;
using Newtonsoft.Json.Serialization;
using YamlDotNet.Serialization;

namespace Neon.Docker
{
    /// <summary>
    /// Service container spread placement settings.
    /// </summary>
    public class ServicePlacementSpreadSettings : INormalizable
    {
        /// <summary>
        /// Label descriptor, such as: engine.labels.az
        /// </summary>
        [JsonProperty(PropertyName = "SpreadDescriptor", Required = Required.Default, DefaultValueHandling = DefaultValueHandling.Populate)]
        [YamlMember(Alias = "SpreadDescriptor", ApplyNamingConventions = false)]
        [DefaultValue(null)]
        public string SpreadDescriptor { get; set; }

        /// <inheritdoc/>
        public void Normalize()
        {
        }
    }
}
