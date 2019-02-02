﻿//-----------------------------------------------------------------------------
// FILE:	    LoginImportCommand.cs
// CONTRIBUTOR: Jeff Lill
// COPYRIGHT:	Copyright (c) 2016-2019 by neonFORGE, LLC.  All rights reserved.

using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Net;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

using Newtonsoft;
using Newtonsoft.Json;

using Neon.Common;
using Neon.Kube;

namespace NeonCli
{
    /// <summary>
    /// Implements the <b>login import</b> command.
    /// </summary>
    public class LoginImportCommand : CommandBase
    {
        private const string usage = @"
Imports an extended Kubernetes context from a file generated by
a previous [neon login export] command.

USAGE:

    neon login import --force] PATH

ARGUMENTS:

    PATH        - Path to the context file.

OPTIONS:

    --force     - Don't prompt to replace an existing context.
";

        /// <inheritdoc/>
        public override string[] Words
        {
            get { return new string[] { "login", "import" }; }
        }

        /// <inheritdoc/>
        public override string[] ExtendedOptions
        {
            get { return new string[] { "--force" }; }
        }

        /// <inheritdoc/>
        public override void Help()
        {
            Console.WriteLine(usage);
        }

        /// <inheritdoc/>
        public override void Run(CommandLine commandLine)
        {
            if (commandLine.Arguments.Length < 1)
            {
                Console.Error.WriteLine("*** ERROR: PATH is required.");
                Program.Exit(1);
            }

            var newLogin        = NeonHelper.YamlDeserialize<KubeLogin>(File.ReadAllText(commandLine.Arguments.First()));
            var existingContext = KubeHelper.Config.GetContext(newLogin.Context.Name);
            var currentContext  = KubeHelper.CurrentContext;

            if (existingContext != null)
            {
                if (!commandLine.HasOption("--force") && !Program.PromptYesNo($"*** Are you sure you want to replace [{existingContext.Name}]?"))
                {
                    return;
                }

                KubeHelper.Config.RemoveContext(existingContext);
            }

            KubeHelper.Config.Contexts.Add(newLogin.Context);

            // Add/replace the user and cluster.

            var existingCluster = KubeHelper.Config.GetCluster(newLogin.Context.Properties.Cluster);

            if (existingCluster != null)
            {
                KubeHelper.Config.Clusters.Remove(existingCluster);
            }

            KubeHelper.Config.Clusters.Add(newLogin.Cluster);

            var existingUser = KubeHelper.Config.GetUser(newLogin.Context.Properties.User);

            if (existingUser != null)
            {
                KubeHelper.Config.Users.Remove(existingUser);
            }

            KubeHelper.Config.Users.Add(newLogin.User);

            Console.Error.WriteLine($"Imported: {newLogin.Context.Name}");
        }

        /// <inheritdoc/>
        public override DockerShimInfo Shim(DockerShim shim)
        {
            return new DockerShimInfo(shimability: DockerShimability.None);
        }
    }
}
