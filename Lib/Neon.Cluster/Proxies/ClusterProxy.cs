﻿//-----------------------------------------------------------------------------
// FILE:	    ClusterProxy.cs
// CONTRIBUTOR: Jeff Lill
// COPYRIGHT:	Copyright (c) 2016-2018 by neonFORGE, LLC.  All rights reserved.

using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Diagnostics.Contracts;
using System.IO;
using System.Linq;
using System.Net;
using System.Net.NetworkInformation;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

using Newtonsoft.Json;
using Newtonsoft.Json.Linq;
using Consul;

using Neon.Common;
using Neon.Docker;
using Neon.IO;
using Neon.Net;
using Neon.Retry;
using Neon.Time;

namespace Neon.Cluster
{
    /// <summary>
    /// Remotely manages a neonCLUSTER.
    /// </summary>
    public class ClusterProxy : IDisposable
    {
        //---------------------------------------------------------------------
        // Local types

        /// <summary>
        /// Enumerates how <see cref="GetHealthyManager(HealthyManagerMode)"/> should
        /// behave when no there are no healthy cluster managers.
        /// </summary>
        public enum HealthyManagerMode
        {
            /// <summary>
            /// Throw an exception when no managers are healthy.
            /// </summary>
            Throw,

            /// <summary>
            /// Return the first manager when no managers are healthy.
            /// </summary>
            ReturnFirst,

            /// <summary>
            /// Return <c>null</c> when no managers are healthy.
            /// </summary>
            ReturnNull
        }

        //---------------------------------------------------------------------
        // Implementation

        private object                                                      syncRoot = new object();
        private ConsulClient                                                consulClient;
        private RunOptions                                                  defaultRunOptions;
        private Func<string, string, IPAddress, SshProxy<NodeDefinition>>   nodeProxyCreator;

        /// <summary>
        /// Constructs a cluster proxy from a cluster login.
        /// </summary>
        /// <param name="clusterLogin">The cluster login information.</param>
        /// <param name="nodeProxyCreator">
        /// The optional application supplied function that creates a node proxy
        /// given the node name, public address or FQDN, private address, and
        /// the node definition.
        /// </param>
        /// <param name="defaultRunOptions">
        /// Optionally specifies the <see cref="RunOptions"/> to be assigned to the 
        /// <see cref="SshProxy{TMetadata}.DefaultRunOptions"/> property for the
        /// nodes managed by the cluster proxy.  This defaults to <see cref="RunOptions.None"/>.
        /// </param>
        /// <remarks>
        /// The <paramref name="nodeProxyCreator"/> function will be called for each node in
        /// the cluster definition giving the application the chance to create the management
        /// proxy using the node's SSH credentials and also to specify logging.  A default
        /// creator that doesn't initialize SSH credentials and logging is used if a <c>null</c>
        /// argument is passed.
        /// </remarks>
        public ClusterProxy(ClusterLogin clusterLogin, Func<string, string, IPAddress, SshProxy<NodeDefinition>> nodeProxyCreator = null, RunOptions defaultRunOptions = RunOptions.None)
            : this(clusterLogin.Definition, nodeProxyCreator, defaultRunOptions)
        {
            this.ClusterLogin = clusterLogin;
        }

        /// <summary>
        /// Constructs a cluster proxy from a cluster definition.
        /// </summary>
        /// <param name="clusterDefinition">The cluster definition.</param>
        /// <param name="nodeProxyCreator">
        /// The application supplied function that creates a management proxy
        /// given the node name, public address or FQDN, private address, and
        /// the node definition.
        /// </param>
        /// <param name="defaultRunOptions">
        /// Optionally specifies the <see cref="RunOptions"/> to be assigned to the 
        /// <see cref="SshProxy{TMetadata}.DefaultRunOptions"/> property for the
        /// nodes managed by the cluster proxy.  This defaults to <see cref="RunOptions.None"/>.
        /// </param>
        /// <remarks>
        /// The <paramref name="nodeProxyCreator"/> function will be called for each node in
        /// the cluster definition giving the application the chance to create the management
        /// proxy using the node's SSH credentials and also to specify logging.  A default
        /// creator that doesn't initialize SSH credentials and logging is used if a <c>null</c>
        /// argument is passed.
        /// </remarks>
        public ClusterProxy(ClusterDefinition clusterDefinition, Func<string, string, IPAddress, SshProxy<NodeDefinition>> nodeProxyCreator = null, RunOptions defaultRunOptions = RunOptions.None)
        {
            Covenant.Requires<ArgumentNullException>(clusterDefinition != null);

            if (nodeProxyCreator == null)
            {
                nodeProxyCreator =
                    (name, publicAddress, privateAddress) =>
                    {
                        var login = NeonClusterHelper.ClusterLogin;

                        if (login != null)
                        {
                            return new SshProxy<NodeDefinition>(name, publicAddress, privateAddress, login.GetSshCredentials());
                        }
                        else
                        {
                            // Note that the proxy returned won't actually work because we're not 
                            // passing valid SSH credentials.  This is useful for situations where
                            // we need a cluster proxy for global things (like managing a hosting
                            // environment) where we won't need access to specific cluster nodes.

                            return new SshProxy<NodeDefinition>(name, publicAddress, privateAddress, SshCredentials.FromUserPassword("null", ""));
                        }
                    };
            }

            this.Definition        = clusterDefinition;
            this.ClusterLogin      = new ClusterLogin();
            this.defaultRunOptions = defaultRunOptions;
            this.nodeProxyCreator  = nodeProxyCreator;

            this.Docker              = new DockerManager(this);
            this.Certificate         = new CertificateManager(this);
            this.Dashboard           = new DashboardManager(this);
            this.DnsHosts            = new DnsHostsManager(this);
            this.PublicLoadBalancer  = new LoadBalanceManager(this, "public");
            this.PrivateLoadBalancer = new LoadBalanceManager(this, "private");
            this.Registry            = new RegistryManager(this);
            this.Globals             = new GlobalsManager(this);
            this.Vault               = new VaultManager(this);

            CreateNodes();
        }

        /// <summary>
        /// Releases all resources associated with the instance.
        /// </summary>
        public void Dispose()
        {
            Dispose(true);
            GC.SuppressFinalize(this);
        }

        /// <summary>
        /// Releases all associated resources.
        /// </summary>
        /// <param name="disposing">Pass <c>true</c> if we're disposing, <c>false</c> if we're finalizing.</param>
        protected virtual void Dispose(bool disposing)
        {
            lock (syncRoot)
            {
                if (consulClient != null)
                {
                    consulClient.Dispose();
                    consulClient = null;
                }

                Vault.Dispose();
            }
        }

        /// <summary>
        /// Returns the cluster name.
        /// </summary>
        public string Name
        {
            get { return Definition.Name; }
        }

        /// <summary>
        /// The associated <see cref="IHostingManager"/> or <c>null</c>.
        /// </summary>
        public IHostingManager HostingManager { get; set; }

        /// <summary>
        /// Returns the cluster login information.
        /// </summary>
        public ClusterLogin ClusterLogin { get; set; }

        /// <summary>
        /// Indicates that any <see cref="SshProxy{TMetadata}"/> instances belonging
        /// to this cluster proxy should use public address/DNS names for SSH connections
        /// rather than their private cluster address.  This defaults to <c>false</c>
        /// and must be modified before establising a node connection to have any effect.
        /// </summary>
        /// <remarks>
        /// <para>
        /// When this is <c>false</c>, connections will be established using node
        /// private addresses.  This implies that the current client has direct
        /// access to the cluster LAN via a direct connection or a VPN.
        /// </para>
        /// <para>
        /// Setting this to <c>true</c> is usually limited to cluster setup scenarios
        /// before the VPN is configured.  Exactly which public addresses and ports will
        /// be used when this is <c>true</c> is determined by the <see cref="HostingManager"/> 
        /// implementation for the current environment.
        /// </para>
        /// </remarks>
        public bool UseNodePublicAddress { get; set; } = false;

        /// <summary>
        /// Returns the cluster definition.
        /// </summary>
        public ClusterDefinition Definition { get; private set; }

        /// <summary>
        /// Returns the read-only list of cluster node proxies.
        /// </summary>
        public IReadOnlyList<SshProxy<NodeDefinition>> Nodes { get; private set; }

        /// <summary>
        /// Returns the first cluster manager node as sorted by name.
        /// </summary>
        public SshProxy<NodeDefinition> FirstManager { get; private set; }

        /// <summary>
        /// Manages Docker components.
        /// </summary>
        public DockerManager Docker { get; private set; }

        /// <summary>
        /// Manages the cluster TLS certificates.
        /// </summary>
        public CertificateManager Certificate { get; private set; }

        /// <summary>
        /// Manages the cluster dashboards.
        /// </summary>
        public DashboardManager Dashboard { get; private set; }

        /// <summary>
        /// Manages the local cluster DNS.
        /// </summary>
        public DnsHostsManager DnsHosts { get; private set; }

        /// <summary>
        /// Manages the cluster's public load balancer.
        /// </summary>
        public LoadBalanceManager PublicLoadBalancer { get; private set; }

        /// <summary>
        /// Manages the cluster's private load balancer.
        /// </summary>
        public LoadBalanceManager PrivateLoadBalancer { get; private set; }

        /// <summary>
        /// Manages the cluster's Docker registry credentials and local registry.
        /// </summary>
        public RegistryManager Registry { get; private set; }

        /// <summary>
        /// Manages the cluster's global settings.
        /// </summary>
        public GlobalsManager Globals { get; private set; }

        /// <summary>
        /// Manages the cluster Vault.
        /// </summary>
        public VaultManager Vault { get; private set; }

        /// <summary>
        /// Returns the named load balancer manager.
        /// </summary>
        /// <param name="name">The load balancer name (one of <b>public</b> or <b>private</b>).</param>
        public LoadBalanceManager GetLoadBalancerManager(string name)
        {
            Covenant.Requires<ArgumentNullException>(!string.IsNullOrEmpty(name));

            switch (name.ToLowerInvariant())
            {
                case "public":

                    return PublicLoadBalancer;

                case "private":

                    return PrivateLoadBalancer;

                default:

                    throw new ArgumentException($"[{name}] is not a valid proxy name.  Specify [public] or [private].");
            }
        }

        /// <summary>
        /// Specifies the <see cref="RunOptions"/> to use when executing commands that 
        /// include secrets.  This defaults to <see cref="RunOptions.Redact"/> for best 
        /// security but may be changed to just <see cref="RunOptions.None"/> when debugging
        /// cluster setup.
        /// </summary>
        public RunOptions SecureRunOptions { get; set; } = RunOptions.Redact | RunOptions.FaultOnError;

        /// <summary>
        /// Enumerates the cluster manager node proxies sorted in ascending order by name.
        /// </summary>
        public IEnumerable<SshProxy<NodeDefinition>> Managers
        {
            get { return Nodes.Where(n => n.Metadata.IsManager).OrderBy(n => n.Name); }
        }

        /// <summary>
        /// Enumerates the cluster worker node proxies sorted in ascending order by name.
        /// </summary>
        public IEnumerable<SshProxy<NodeDefinition>> Workers
        {
            get { return Nodes.Where(n => n.Metadata.IsWorker).OrderBy(n => n.Name); }
        }

        /// <summary>
        /// Enumerates the cluster pet node proxies sorted in ascending order by name.
        /// </summary>
        public IEnumerable<SshProxy<NodeDefinition>> Pets
        {
            get { return Nodes.Where(n => n.Metadata.IsPet).OrderBy(n => n.Name); }
        }

        /// <summary>
        /// Initializes or reinitializes the <see cref="Nodes"/> list.  This is called during
        /// construction and also in rare situations where the node proxies need to be 
        /// recreated (e.g. after configuring node static IP addresses).
        /// </summary>
        public void CreateNodes()
        {
            var nodes = new List<SshProxy<NodeDefinition>>();

            foreach (var nodeDefinition in Definition.SortedNodes)
            {
                var node = nodeProxyCreator(nodeDefinition.Name, nodeDefinition.PublicAddress, IPAddress.Parse(nodeDefinition.PrivateAddress ?? "0.0.0.0"));

                node.Cluster           = this;
                node.DefaultRunOptions = defaultRunOptions;
                node.Metadata          = nodeDefinition;
                nodes.Add(node);
            }

            this.Nodes        = nodes;
            this.FirstManager = Nodes.Where(n => n.Metadata.IsManager).OrderBy(n => n.Name).First();
        }

        /// <summary>
        /// Returns the <see cref="SshProxy{TMetadata}"/> instance for a named node.
        /// </summary>
        /// <param name="nodeName">The node name.</param>
        /// <returns>The node proxy instance.</returns>
        /// <exception cref="KeyNotFoundException">Thrown if the name node is not present in the cluster.</exception>
        public SshProxy<NodeDefinition> GetNode(string nodeName)
        {
            var node = Nodes.SingleOrDefault(n => string.Compare(n.Name, nodeName, StringComparison.OrdinalIgnoreCase) == 0);

            if (node == null)
            {
                throw new KeyNotFoundException($"The node [{nodeName}] is not present in the cluster.");
            }

            return node;
        }

        /// <summary>
        /// Looks for the <see cref="SshProxy{TMetadata}"/> instance for a named node.
        /// </summary>
        /// <param name="nodeName">The node name.</param>
        /// <returns>The node proxy instance or <c>null</c> if the named node does not exist.</returns>
        public SshProxy<NodeDefinition> FindNode(string nodeName)
        {
            return Nodes.SingleOrDefault(n => string.Compare(n.Name, nodeName, StringComparison.OrdinalIgnoreCase) == 0);
        }

        /// <summary>
        /// Returns a manager node that appears to be healthy.
        /// </summary>
        /// <param name="failureMode">Specifies what should happen when there are no healthy managers.</param>
        /// <returns>The healthy manager node.</returns>
        /// <exception cref="ClusterException">
        /// Thrown if no healthy managers are present and
        /// <paramref name="failureMode"/>=<see cref="HealthyManagerMode.Throw"/>.
        /// </exception>
        public SshProxy<NodeDefinition> GetHealthyManager(HealthyManagerMode failureMode = HealthyManagerMode.ReturnFirst)
        {
            // Try sending up to three pings to each manager node in parallel
            // to get a list of the health ones.  Then we'll return the first
            // healthy manager from the list (as sorted by name).
            //
            // This will consistently return the first manager node by name
            // if it's health, otherwise it will fail over to the next, etc.

            const int tryCount = 3;

            var healthyManagers = new List<SshProxy<NodeDefinition>>();
            var pingOptions     = new PingOptions(ttl: 32, dontFragment: true);
            var pingTimeout     = TimeSpan.FromSeconds(1);

            for (int i = 0; i < tryCount; i++)
            {
                Parallel.ForEach(Nodes.Where(n => n.Metadata.IsManager),
                    manager =>
                    {
                        using (var ping = new Ping())
                        {
                            var reply = ping.Send(manager.PrivateAddress, (int)pingTimeout.TotalMilliseconds);

                            if (reply.Status == IPStatus.Success)
                            {
                                lock (healthyManagers)
                                {
                                    healthyManagers.Add(manager);
                                }
                            }
                        }
                    });

                if (healthyManagers.Count > 0)
                {
                    return healthyManagers.OrderBy(n => n.Name).First();
                }
            }

            switch (failureMode)
            {
                case HealthyManagerMode.ReturnFirst:

                    return FirstManager;

                case HealthyManagerMode.ReturnNull:

                    return null;

                case HealthyManagerMode.Throw:

                    throw new ClusterException("Could not locate a healthy cluster manager node.");

                default:

                    throw new NotImplementedException($"Unexpected failure [mode={failureMode}].");
            }
        }

        /// <summary>
        /// Performs cluster configuration steps.
        /// </summary>
        /// <param name="steps">The configuration steps.</param>
        public void Configure(ConfigStepList steps)
        {
            Covenant.Requires<ArgumentNullException>(steps != null);

            foreach (var step in steps)
            {
                step.Run(this);
            }
        }

        /// <summary>
        /// Returns steps that upload a text file to a set of node proxies.
        /// </summary>
        /// <param name="nodes">The node proxies to receive the upload.</param>
        /// <param name="path">The target path on the Linux node.</param>
        /// <param name="text">The input text.</param>
        /// <param name="tabStop">Optionally expands TABs into spaces when non-zero.</param>
        /// <param name="outputEncoding">Optionally specifies the output text encoding (defaults to UTF-8).</param>
        /// <returns>The steps.</returns>
        public IEnumerable<ConfigStep> GetFileUploadSteps(IEnumerable<SshProxy<NodeDefinition>> nodes, string path, string text, int tabStop = 0, Encoding outputEncoding = null)
        {
            var steps = new ConfigStepList();

            foreach (var node in nodes)
            {
                steps.Add(UploadStep.Text(node.Name, path, text, tabStop, outputEncoding));
            }

            return steps;
        }

        /// <summary>
        /// Returns a Consul client.
        /// </summary>
        /// <returns>The <see cref="ConsulClient"/>.</returns>
        public ConsulClient Consul
        {
            get
            {
                lock (syncRoot)
                {
                    if (consulClient != null)
                    {
                        return consulClient;
                    }

                    consulClient = NeonClusterHelper.OpenConsul();
                }

                return consulClient;
            }
        }

        /// <summary>
        /// Indicates that the cluster certificates and or load balancer rules may have been changed.
        /// This has the effect of signalling <b>neon-proxy-manager</b> to to regenerate the proxy 
        /// definitions and update all of the load balancers when changes are detected.
        /// </summary>
        public void SignalLoadBalancerUpdate()
        {
            Consul.KV.PutString("neon/service/neon-proxy-manager/conf/reload", Guid.NewGuid().ToString("D")).Wait();
        }
    }
}
