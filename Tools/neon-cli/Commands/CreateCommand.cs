﻿//-----------------------------------------------------------------------------
// FILE:	    CreateCommand.cs
// CONTRIBUTOR: Jeff Lill
// COPYRIGHT:	Copyright (c) 2016-2017 by neonFORGE, LLC.  All rights reserved.

using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Security.Cryptography;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

using Newtonsoft;
using Newtonsoft.Json;

using Neon.Cluster;
using Neon.Common;

namespace NeonCli
{
    /// <summary>
    /// Implements the <b>create</b> command.
    /// </summary>
    public class CreateCommand : CommandBase
    {
        private const string usage = @"
USAGE:

    neon create cypher              - Generates 16-byte encryption key
    neon create password [OPTIONS]  - Generates secure password
";
        /// <inheritdoc/>
        public override string[] Words
        {
            get { return new string[] { "create" }; }
        }

        /// <inheritdoc/>
        /// <inheritdoc/>
        public override void Help()
        {
            Console.WriteLine(usage);
        }

        /// <inheritdoc/>
        public override void Run(CommandLine commandLine)
        {
            Help();
        }

        /// <inheritdoc/>
        public override ShimInfo Shim(DockerShim shim)
        {
            return new ShimInfo(isShimmed: true);
        }
    }
}
