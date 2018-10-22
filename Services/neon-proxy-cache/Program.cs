﻿//-----------------------------------------------------------------------------
// FILE:	    Program.cs
// CONTRIBUTOR: Jeff Lill
// COPYRIGHT:	Copyright (c) 2016-2018 by neonFORGE, LLC.  All rights reserved.

using System;
using System.Collections.Generic;
using System.Diagnostics.Contracts;
using System.IO;
using System.Linq;
using System.Net;
using System.Reflection;
using System.Runtime.Loader;
using System.Security.Cryptography;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

using Consul;
using ICSharpCode.SharpZipLib.Zip;
using Newtonsoft.Json;

using Neon.Common;
using Neon.Cryptography;
using Neon.Diagnostics;
using Neon.Docker;
using Neon.Hive;
using Neon.HiveMQ;
using Neon.Tasks;
using Neon.Time;

namespace NeonProxyCache
{
    /// <summary>
    /// <para>
    /// Implements the <b>neon-proxy-cache</b> service by launching and then managing a Varnish subprocess.  This
    /// service listens for HiveMQ notifications from <b>neon-proxy-manager</b>, indicating that the HAProxy/Varnish
    /// may have changed and that the Varnish process should be notified of the changes.  This is built into the
    /// <a href="https://hub.docker.com/r/nhive/neon-proxy-cache/">nhive/neon-proxy-cache</a> image and will run
    /// as the main container process.
    /// </para>
    /// <para>
    /// This service handles cache warming by perodically retrieving designated pages and files from the target services
    /// and the service also handles HiveMQ notifications commanding that items be purged from the caches.
    /// </para>
    /// </summary>
    public static partial class Program
    {
        private const string DefMemoryLimitString = "100MB";
        private const long DefMemoryLimit         = 100 * NeonHelper.Mega;

        private const string MinMemoryLimitString = "25MMB";
        private const long MinMemoryLimit         = 25 * NeonHelper.Mega;

        // Environment variables:

        private static string                   configKey;
        private static string                   configHashKey;
        private static long                     memoryLimit;
        private static TimeSpan                 warnInterval;
        private static bool                     debugMode;

        // File system paths:

        private const string tmpfsFolder        = "/dev/shm";
        private const string configFolder       = tmpfsFolder + "/varnish";
        private const string configPath         = configFolder + "/varnish.vcl";
        private const string configUpdateFolder = tmpfsFolder + "/varnish-update";
        private const string configUpdatePath   = configUpdateFolder + "/varnish.vcl";

        // Service state:

        private static string                   serviceName;
        private static ProcessTerminator        terminator;
        private static bool                     isPublic = false;
        private static INeonLogger              log;
        private static HiveProxy                hive;
        private static ConsulClient             consul;
        private static DateTime                 errorTimeUtc = DateTime.MinValue;
        private static object                   syncLock     = new object();
        private static CancellationTokenSource  cts          = new CancellationTokenSource();

        /// <summary>
        /// Application entry point.
        /// </summary>
        /// <param name="args">Command line arguments.</param>
        public static async Task Main(string[] args)
        {
            LogManager.Default.SetLogLevel(Environment.GetEnvironmentVariable("LOG_LEVEL"));
            log = LogManager.Default.GetLogger(typeof(Program));

            // Create process terminator to handle termination signals.

            terminator = new ProcessTerminator(log);

            terminator.AddHandler(
                () =>
                {
                    // Cancel any operations in progress.

                    cts.Cancel();
                });

            // Read the environment variables.

            // $hack(jeff.lill:
            //
            // We're going to scan the Consul configuration key to determine whether this
            // instance is managing the public or private proxy (or bridges) so we'll
            // be completely compatible with existing deployments.
            //
            // In theory, we could have passed a new environment variable but that's not
            // worth the trouble.

            configKey = Environment.GetEnvironmentVariable("CONFIG_KEY");

            if (string.IsNullOrEmpty(configKey))
            {
                log.LogError("[CONFIG_KEY] environment variable is required.");
                Program.Exit(1);
            }

            isPublic = configKey.Contains("/public/");

            var proxyName = isPublic ? "public" : "private";

            serviceName = $"neon-proxy-{proxyName}:{GitVersion}";

            log.LogInfo(() => $"Starting [{serviceName}]");

            configHashKey = Environment.GetEnvironmentVariable("CONFIG_HASH_KEY");

            if (string.IsNullOrEmpty(configHashKey))
            {
                log.LogError("[CONFIG_HASH_KEY] environment variable is required.");
                Program.Exit(1);
            }

            var memoryLimitValue = Environment.GetEnvironmentVariable("MEMORY_LIMIT");

            if (string.IsNullOrEmpty(memoryLimitValue))
            {
                memoryLimitValue = DefMemoryLimitString;
            }

            if (!NeonHelper.TryParseCount(memoryLimitValue, out var memoryLimitDouble))
            {
                memoryLimitDouble = DefMemoryLimit;
            }

            if (memoryLimitDouble < MinMemoryLimit)
            {
                log.LogWarn(() => $"[MEMORY_LIMIT={memoryLimitValue}] is to small.  Using [{MinMemoryLimitString}] instead.");
                memoryLimitDouble = MinMemoryLimit;
            }

            memoryLimit = (long)memoryLimitDouble;

            var warnSeconds = Environment.GetEnvironmentVariable("WARN_SECONDS");

            if (string.IsNullOrEmpty(warnSeconds) || !double.TryParse(warnSeconds, out var warnSecondsValue))
            {
                warnInterval = TimeSpan.FromSeconds(300);
            }
            else
            {
                warnInterval = TimeSpan.FromSeconds(warnSecondsValue);
            }

            debugMode = "true".Equals(Environment.GetEnvironmentVariable("DEBUG"), StringComparison.InvariantCultureIgnoreCase);

            log.LogInfo(() => $"LOG_LEVEL={LogManager.Default.LogLevel.ToString().ToUpper()}");
            log.LogInfo(() => $"CONFIG_KEY={configKey}");
            log.LogInfo(() => $"CONFIG_HASH_KEY={configHashKey}");
            log.LogInfo(() => $"MEMORY_LIMIT={memoryLimit}");
            log.LogInfo(() => $"WARN_SECONDS={warnInterval}");
            log.LogInfo(() => $"DEBUG={debugMode}");

            // Ensure that the required directories exist.

            Directory.CreateDirectory(tmpfsFolder);
            Directory.CreateDirectory(configFolder);
            Directory.CreateDirectory(configUpdateFolder);

            // Establish the hive connections.

            if (NeonHelper.IsDevWorkstation)
            {
                throw new NotImplementedException("This service works only within a Linux container with Varnish installed.");

                //var vaultCredentialsSecret = "neon-proxy-manager-credentials";

                //Environment.SetEnvironmentVariable("VAULT_CREDENTIALS", vaultCredentialsSecret);

                //hive = HiveHelper.OpenHiveRemote(new DebugSecrets().VaultAppRole(vaultCredentialsSecret, $"neon-proxy-{proxyName}"));
            }
            else
            {
                hive = HiveHelper.OpenHive();
            }

            try
            {
                // Open Consul and then start the service tasks.

                log.LogInfo(() => $"Connecting: Consul");

                using (consul = HiveHelper.OpenConsul())
                {
                    log.LogInfo(() => $"Connecting: {HiveMQChannels.ProxyNotify} channel");

                    // Verify that the required Consul keys exist or loop to wait until they
                    // are created.  This will allow the service wait for pending hive setup
                    // operations to be completed.

                    while (!await consul.KV.Exists(configKey))
                    {
                        log.LogWarn(() => $"Waiting for [{configKey}] key to be present in Consul.");
                        await Task.Delay(TimeSpan.FromSeconds(5));
                    }

                    while (!await consul.KV.Exists(configHashKey))
                    {
                        log.LogWarn(() => $"Waiting for [{configHashKey}] key to be present in Consul.");
                        await Task.Delay(TimeSpan.FromSeconds(5));
                    }

                    // Crank up the service tasks.

                    await NeonHelper.WaitAllAsync(
                        ErrorPollerAsync(),
                        VarnishShim());

                    terminator.ReadyToExit();
                }
            }
            catch (Exception e)
            {
                log.LogCritical(e);
                Program.Exit(1);
            }
            finally
            {
                HiveHelper.CloseHive();
                terminator.ReadyToExit();
            }

            Program.Exit(0);
        }

        /// <summary>
        /// Returns the program version as the Git branch and commit and an optional
        /// indication of whether the program was build from a dirty branch.
        /// </summary>
        public static string GitVersion
        {
            get
            {
                var version = $"{ThisAssembly.Git.Branch}-{ThisAssembly.Git.Commit}";

#pragma warning disable 162 // Unreachable code

                //if (ThisAssembly.Git.IsDirty)
                //{
                //    version += "-DIRTY";
                //}

#pragma warning restore 162 // Unreachable code

                return version;
            }
        }

        /// <summary>
        /// Exits the service with an exit code.  This method defaults to using
        /// the <see cref="ProcessTerminator"/> to gracefully exit the program.
        /// This can be overridden by passing <paramref name="force"/><c>=true</c>.
        /// </summary>
        /// <param name="exitCode">The exit code.</param>
        /// <param name="force">Forces an immediate ungraceful exit.</param>
        public static void Exit(int exitCode, bool force = false)
        {
            log.LogInfo(() => $"Exiting: [{serviceName}]");

            if (terminator == null)
            {
                Environment.Exit(exitCode);
            }
            else
            {
                terminator.Exit(exitCode);
            }
        }
    }
}
