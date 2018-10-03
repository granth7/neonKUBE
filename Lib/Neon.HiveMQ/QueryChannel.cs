﻿//-----------------------------------------------------------------------------
// FILE:	    QueryChannel.cs
// CONTRIBUTOR: Jeff Lill
// COPYRIGHT:	Copyright (c) 2016-2018 by neonFORGE, LLC.  All rights reserved.

using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics.Contracts;
using System.IO;
using System.Threading;
using System.Threading.Tasks;

using EasyNetQ;
using EasyNetQ.DI;
using EasyNetQ.Logging;
using EasyNetQ.Management.Client;

using RabbitMQ;
using RabbitMQ.Client;

using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

using Neon.Common;
using Neon.Diagnostics;
using Neon.Net;

namespace Neon.HiveMQ
{
    /// <summary>
    /// <para>
    /// Implements query/response messaging operations for a <see cref="MessageBus"/>.  
    /// Message producers and consumers each need to declare a channel with the 
    /// same name by calling one of the <see cref="MessageBus"/> to be able to
    /// broadcast and consume messages.
    /// </para>
    /// <note>
    /// <see cref="QueryChannel"/> has nothing to do with an underlying
    /// RabbitMQ channel.  These are two entirely different concepts.
    /// </note>
    /// </summary>
    public class QueryChannel : Channel
    {
        private static readonly TimeSpan defaultTimeout = TimeSpan.FromSeconds(15);

        /// <summary>
        /// Internal constructor.
        /// </summary>
        /// <param name="messageBus">The <see cref="MessageBus"/>.</param>
        /// <param name="name">The channel name.</param>
        internal QueryChannel(MessageBus messageBus, string name)
            : base(messageBus, name)
        {
        }

        /// <summary>
        /// Synchronously sends a query message on the channel and waits for a response.  
        /// </summary>
        /// <typeparam name="TQuery">The query message type.</typeparam>
        /// <typeparam name="TResponse">The response message type.</typeparam>
        /// <param name="request">The request message.</param>
        /// <param name="timeout">The maximum time to wait (defaults to <b>15 seconds</b>).</param>
        /// <returns>The response message.</returns>
        /// <exception cref="TimeoutException">Thrown if the timeout expired before a response was received.</exception>
        /// <remarks>
        /// <note>
        /// Synchronous queries are not particularily efficient and their use
        /// should be restricted to situations where query traffic will be low.
        /// We recommend that most applications, especially services, use
        /// <see cref="QueryAsync{TQuery, TResponse}(TQuery, TimeSpan, CancellationToken)"/>
        /// instead.
        /// </note>
        /// </remarks>
        public TResponse Query<TQuery, TResponse>(TQuery request, TimeSpan timeout = default)
            where TQuery : class, new()
            where TResponse : class, new()
        {
            Covenant.Requires<ArgumentNullException>(request != null);
            Covenant.Requires<ArgumentException>(timeout >= TimeSpan.Zero);

            if (timeout == TimeSpan.Zero)
            {
                timeout = defaultTimeout;
            }

            throw new NotImplementedException();
        }

        /// <summary>
        /// Asynchronously sends a query message on the channel and waits for a response.
        /// </summary>
        /// <typeparam name="TQuery">The query message type.</typeparam>
        /// <typeparam name="TResponse">The response message type.</typeparam>
        /// <param name="request">The request message.</param>
        /// <param name="timeout">The maximum time to wait (defaults to <b>15 seconds</b>).</param>
        /// <param name="cancellationToken">The optional cancellation token.</param>
        /// <returns>The response message.</returns>
        /// <exception cref="TimeoutException">Thrown if the timeout expired before a response was received.</exception>
        public async Task<TResponse> QueryAsync<TQuery, TResponse>(TQuery request, TimeSpan timeout = default, CancellationToken cancellationToken = default)
            where TQuery : class, new()
            where TResponse : class, new()
        {
            Covenant.Requires<ArgumentNullException>(request != null);
            Covenant.Requires<ArgumentException>(timeout >= TimeSpan.Zero);

            if (timeout == TimeSpan.Zero)
            {
                timeout = defaultTimeout;
            }

            await Task.CompletedTask;
            throw new NotImplementedException();
        }
    }
}
