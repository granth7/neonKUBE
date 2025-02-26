﻿//-----------------------------------------------------------------------------
// FILE:	    RoundtripDataFactory.cs
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
using System.Diagnostics.Contracts;
using System.Globalization;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

using Microsoft.Extensions.DependencyInjection;

using Newtonsoft.Json;
using Newtonsoft.Json.Converters;
using Newtonsoft.Json.Linq;
using Newtonsoft.Json.Serialization;

using Neon.Diagnostics;

namespace Neon.Data
{
    /// <summary>
    /// Used to instantiate code generated classes that implement <see cref=" IRoundtripData"/>
    /// as generated by the <c>Neon.ModelGen</c> assembly.
    /// </summary>
    public static class RoundtripDataFactory 
    {
        private static Dictionary<string, MethodInfo>   classNameToJObjectCreateMethod     = new Dictionary<string, MethodInfo>();
        private static Dictionary<string, MethodInfo>   classNameToStreamCreateMethod      = new Dictionary<string, MethodInfo>();
        private static Dictionary<string, MethodInfo>   classNameToStreamAsyncCreateMethod = new Dictionary<string, MethodInfo>();
        private static Type[]                           createFromJObjectArgTypes          = new Type[] { typeof(JObject) };
        private static Type[]                           createFromStreamArgTypes           = new Type[] { typeof(Stream), typeof(Encoding) };

        /// <summary>
        /// Constructs an instance of <typeparamref name="TResult"/> from a <see cref="JObject"/>.
        /// </summary>
        /// <typeparam name="TResult">The result type.</typeparam>
        /// <param name="jObject">The source <see cref="JObject"/>.</param>
        /// <returns>The new <typeparamref name="TResult"/> instance.</returns>
        public static TResult CreateFrom<TResult>(JObject jObject)
        {
            return (TResult)CreateFrom(typeof(TResult), jObject);
        }

        /// <summary>
        /// Constructs an instance of <paramref name="resultType"/> from a <see cref="JObject"/>.
        /// </summary>
        /// <param name="resultType">The result type.</param>
        /// <param name="jObject">The source <see cref="JObject"/>.</param>
        /// <returns>The new instance as an <see cref="object"/>.</returns>
        public static object CreateFrom(Type resultType, JObject jObject)
        {
            Covenant.Requires(resultType != null);
            Covenant.Requires(jObject != null);
#if DEBUG
            Covenant.Requires<ArgumentException>(resultType.Implements<IRoundtripData>());
#endif
            MethodInfo createMethod;

            lock (classNameToJObjectCreateMethod)
            {
                if (!classNameToJObjectCreateMethod.TryGetValue(resultType.FullName, out createMethod))
                {
                    createMethod = resultType.GetMethod("CreateFrom", BindingFlags.Public | BindingFlags.Static, null, createFromJObjectArgTypes, null);
#if DEBUG
                    Covenant.Assert(createMethod != null, $"Cannot locate generated [{resultType.FullName}.CreateFrom(JObject)] method.");
#endif
                    classNameToJObjectCreateMethod.Add(resultType.FullName, createMethod);
                }
            }

            return createMethod.Invoke(null, new object[] { jObject });
        }

        /// <summary>
        /// Constructs an instance of <typeparamref name="TResult"/> from a byte array.
        /// </summary>
        /// <typeparam name="TResult">The result type.</typeparam>
        /// <param name="bytes">The source bytes.</param>
        /// <returns>The new <typeparamref name="TResult"/> instance.</returns>
        public static TResult CreateFrom<TResult>(byte[] bytes)
        {
            return (TResult)CreateFrom(typeof(TResult), bytes);
        }

        /// <summary>
        /// Constructs an instance of <paramref name="resultType"/> from a byte array.
        /// </summary>
        /// <param name="resultType">The result type.</param>
        /// <param name="bytes">The source bytes.</param>
        /// <returns>The new instance as an <see cref="object"/>.</returns>
        public static object CreateFrom(Type resultType, byte[] bytes)
        {
            Covenant.Requires(resultType != null);
            Covenant.Requires(bytes != null);

            var json    = Encoding.UTF8.GetString(bytes);  // $debug(jeff.lill): DELETE THIS!
            var jToken  = JToken.Parse(json);

            switch (jToken.Type)
            {
                case JTokenType.Null:

                    return null;

                case JTokenType.Object:

                    return CreateFrom(resultType, JObject.Parse(json));

                default:

                    throw new ArgumentException("Invalid JSON: Expecting an object or NULL.");
            }
        }

        /// <summary>
        /// Constructs an instance of <paramref name="resultType"/> from a <see cref="Stream"/>.
        /// </summary>
        /// <param name="resultType">The result type.</param>
        /// <param name="stream">The source <see cref="Stream"/>.</param>
        /// <param name="encoding">Optionally specifies the encoding (defaults to UTF-8).</param>
        /// <returns>The new instance as an <see cref="object"/>.</returns>
        public static async Task<object> CreateFromAsync(Type resultType, Stream stream, Encoding encoding = null)
        {
            Covenant.Requires(resultType != null);
            Covenant.Requires(stream != null);

            if (encoding == null)
            {
                encoding = Encoding.UTF8;
            }

            MethodInfo createMethod;

            lock (classNameToStreamCreateMethod)
            {
                if (!classNameToStreamCreateMethod.TryGetValue(resultType.FullName, out createMethod))
                {
                    if (!resultType.Implements<IRoundtripData>())
                    {
                        throw new InvalidOperationException($"Type [{resultType.FullName}] does not implement [{nameof(IRoundtripData)}].");
                    }

                    createMethod = resultType.GetMethod("CreateFromAsync", BindingFlags.Public | BindingFlags.Static, null, createFromStreamArgTypes, null);
#if DEBUG
                    Covenant.Assert(createMethod != null, $"Cannot locate generated [{resultType.FullName}.CreateFromAsync(Stream, Encoding)] method.");
#endif
                    classNameToStreamCreateMethod.Add(resultType.FullName, createMethod);
                }
            }

            return await (Task<object>)createMethod.Invoke(null, new object[] { stream, encoding });
        }

        /// <summary>
        /// Attempts to construct an instance of <paramref name="resultType"/> from a <see cref="Stream"/>.
        /// </summary>
        /// <param name="resultType">The result type.</param>
        /// <param name="stream">The source <see cref="Stream"/>.</param>
        /// <param name="encoding">Optionally specifies the encoding (defaults to UTF-8).</param>
        /// <returns>
        /// <c>true</c> if the object type implements <see cref="IRoundtripData"/> and the 
        /// object was successfully deserialized.
        /// </returns>
        public static async Task<Tuple<bool, object>> TryCreateFromAsync(Type resultType, Stream stream, Encoding encoding )
        {
            Covenant.Requires(resultType != null);
            Covenant.Requires(stream != null);

            if (encoding == null)
            {
                encoding = Encoding.UTF8;
            }

            MethodInfo createMethod;

            lock (classNameToStreamAsyncCreateMethod)
            {
                if (!classNameToStreamAsyncCreateMethod.TryGetValue(resultType.FullName, out createMethod))
                {
                    if (!resultType.Implements<IRoundtripData>())
                    {
                        return Tuple.Create<bool, object>(false, null);
                    }

                    createMethod = resultType.GetMethod("CreateFromAsync", BindingFlags.Public | BindingFlags.Static, null, createFromStreamArgTypes, null);
#if DEBUG
                    Covenant.Assert(createMethod != null, $"Cannot locate generated [{resultType.FullName}.CreateFromAsync(Stream, Encoding)] method.");
#endif
                    classNameToStreamAsyncCreateMethod.Add(resultType.FullName, createMethod);
                }
            }

            return Tuple.Create<bool, object>(true, await (Task<object>)createMethod.Invoke(null, new object[] { stream, encoding }));
        }
    }
}
