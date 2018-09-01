﻿//-----------------------------------------------------------------------------
// FILE:	    RabbitMQOptions.cs
// CONTRIBUTOR: Jeff Lill
// COPYRIGHT:	Copyright (c) 2016-2018 by neonFORGE, LLC.  All rights reserved.

using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics.Contracts;
using System.IO;
using System.Linq;
using System.Net;
using System.Net.Http;
using System.Text;
using System.Text.RegularExpressions;
using System.Threading;
using System.Threading.Tasks;

using Newtonsoft.Json;
using Newtonsoft.Json.Converters;
using Newtonsoft.Json.Serialization;

using Neon.Common;
using Neon.IO;

namespace Neon.Hive
{
    /// <summary>
    /// Specifies the options for configuring the hive integrated
    /// <a href="https://rabbitmq.com/">RabbitMQ message queue</a>
    /// cluster.
    /// </summary>
    public class RabbitMQOptions
    {
        private const string defaultRamLimit         = "500MB";
        private const double defaultRamHighWatermark = 0.50;
        private const string defaultUsername         = "sysadmin";
        private const string defaultPassword         = "password";
        private const string defaultPartitionMode    = "autoheal";
        private const string defaultRabbitMQImage    = HiveConst.NeonPublicRegistry + "/rabbitmq:latest";

        /// <summary>
        /// Specifies the maximum RAM to be allocated to each RabbitMQ node container.
        /// This can be a long byte count or a long with units like <b>512MB</b> or <b>2GB</b>.
        /// This defaults to <b>500MB</b> which is the minimum supported value.
        /// </summary>
        [JsonProperty(PropertyName = "RamLimit", Required = Required.Default)]
        [DefaultValue(defaultRamLimit)]
        public string RamLimit { get; set; } = defaultRamLimit;

        /// <summary>
        /// <para>
        /// Specifies the how much of <see cref="RamLimit"/> each node can allocate for
        /// caching and internal use expressed as a number between 0.0 - 1.0.  This
        /// defaults to <c>0.50</c> indicating that up to half of <see cref="RamLimit"/>
        /// may be used.
        /// </para>
        /// <note>
        /// The default value is very conservative especially as you increase <see cref="RamLimit"/>.
        /// For larger RAM values you should be able allocate a larger percentage of RAM for
        /// this data.
        /// </note>
        /// </summary>
        [JsonProperty(PropertyName = "RamHighWatermark", Required = Required.Default)]
        [DefaultValue(defaultRamHighWatermark)]
        public double RamHighWatermark { get; set;  } = defaultRamHighWatermark;

        /// <summary>
        /// Specifies the username used to secure the cluster.  This defaults to <b>sysadmin</b>.
        /// </summary>
        [JsonProperty(PropertyName = "Username", Required = Required.Default)]
        [DefaultValue(defaultUsername)]
        public string Username { get; set; } = defaultUsername;

        /// <summary>
        /// Specifies the password used to secure the cluster.  This defaults to <b>password</b>.
        /// </summary>
        [JsonProperty(PropertyName = "Password", Required = Required.Default)]
        [DefaultValue(defaultPassword)]
        public string Password { get; set; } = defaultPassword;

        /// <summary>
        /// Specifies the shared secret clustered RabbitMQ nodes will use for mutual authentication.
        /// A secure password will be generated if this isn't specified.
        /// </summary>
        [JsonProperty(PropertyName = "ErlangCookie", Required = Required.Default)]
        [DefaultValue(null)]
        public string ErlangCookie { get; set; }

        /// <summary>
        /// <para>
        /// Specifies how the RabbitMQ cluster will deal with network partitions.  The possible
        /// values are <b>autoheal</b>, <b>pause_minority</b>, or <b>pause_if_all_down</b>.
        /// This defaults to <b>autoheal</b> to favor availability over the potential for data
        /// loss.  The other modes may require manual intervention to being the cluster back
        /// online even after simply shutting the cluster down.
        /// </para>
        /// <para>
        /// See <a href="https://www.rabbitmq.com/partitions.html">Clustering and Network Partitions</a>
        /// for more information.
        /// </para>
        /// </summary>
        [JsonProperty(PropertyName = "PartitionMode", Required = Required.Default)]
        [DefaultValue(defaultPartitionMode)]
        public string PartitionMode { get; set; } = defaultPartitionMode;

        /// <summary>
        /// The Docker image to be used to provision the <b>neon-rabbitmq</b> service.
        /// This defaults to <b>nhive/rabbitmq:latest</b>.
        /// </summary>
        [JsonProperty(PropertyName = "RabbitMQImage", Required = Required.Default, DefaultValueHandling = DefaultValueHandling.Include)]
        [DefaultValue(defaultRabbitMQImage)]
        public string RabbitMQImage { get; set; } = defaultRabbitMQImage;

        /// <summary>
        /// Validates the options and also ensures that all <c>null</c> properties are
        /// initialized to their default values.
        /// </summary>
        /// <param name="hiveDefinition">The hive definition.</param>
        /// <exception cref="HiveDefinitionException">Thrown if the definition is not valid.</exception>
        public void Validate(HiveDefinition hiveDefinition)
        {
            RamLimit      = RamLimit ?? defaultRamLimit;
            Username      = Username ?? defaultUsername;
            Password      = Password ?? defaultPassword;
            PartitionMode = PartitionMode ?? defaultPartitionMode;
            PartitionMode = PartitionMode.ToLowerInvariant();
            RabbitMQImage = RabbitMQImage ?? defaultRabbitMQImage;

            if (string.IsNullOrWhiteSpace(ErlangCookie))
            {
                ErlangCookie = NeonHelper.GetRandomPassword(20);
            }

            if (RamHighWatermark <= 0.0)
            {
                RamHighWatermark = defaultRamHighWatermark;
            }

            if (HiveDefinition.ValidateSize(RamLimit, this.GetType(), nameof(RamLimit)) < 500 * NeonHelper.Mega)
            {
                throw new HiveDefinitionException($"[{nameof(RabbitMQOptions)}.{nameof(RamLimit)}={RamLimit}] cannot be less than [500MB].");
            }

            if (RamHighWatermark <= 0.0 || RamHighWatermark > 1.0)
            {
                throw new HiveDefinitionException($"[{nameof(RabbitMQOptions)}.{nameof(RamHighWatermark)}={RamHighWatermark}] must be a positive number between 0..1.");
            }

            switch (PartitionMode)
            {
                case "autoheal":
                case "pause_minority":
                case "pause_if_all_down":

                    break;

                default:

                    throw new HiveDefinitionException($"[{nameof(RabbitMQOptions)}.{nameof(PartitionMode)}={PartitionMode}] is not valid.  Specify one of [autoheal], [pause_minority], or [pause_if_all_down].");
            }

            // We need to assign hive nodes to host the RabbitMQ instances.  We're going to do
            // this by examining and possibly setting the RabbitMQ node labels.  If no hive nodes
            // have this label set, then we'll set these for all manager nodes.  Otherwise, we'll
            // deploy to the marked nodes.

            if (hiveDefinition.Nodes.Count(n => n.Labels.RabbitMQ) == 0)
            {
                foreach (var manager in hiveDefinition.Managers)
                {
                    manager.Labels.RabbitMQ = true;
                }
            }
        }
    }
}
