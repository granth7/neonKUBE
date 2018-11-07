﻿//-----------------------------------------------------------------------------
// FILE:	    Test_HiveLoadBalancer.Private.Https.cs
// CONTRIBUTOR: Jeff Lill
// COPYRIGHT:	Copyright (c) 2016-2018 by neonFORGE, LLC.  All rights reserved.

using System;
using System.Collections.Generic;
using System.Diagnostics.Contracts;
using System.IO;
using System.Linq;
using System.Net;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;

using Neon.Common;
using Neon.Cryptography;
using Neon.Hive;
using Neon.Xunit;
using Neon.Xunit.Hive;

using Xunit;

namespace TestHive
{
    public partial class Test_HiveLoadBalancer : IClassFixture<HiveFixture>
    {
        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Https_Private_Uncached_DefaultPort()
        {
            await TestHttpsRule("https-private-defaultport", HiveHostPorts.ProxyPrivateHttps, HiveConst.PrivateNetwork, hive.PrivateLoadBalancer);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Https_Private_Uncached_NonDefaultPort()
        {
            await TestHttpsRule("https-private-nondefaultport", HiveHostPorts.ProxyPrivateLastUserPort, HiveConst.PrivateNetwork, hive.PrivateLoadBalancer);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Https_Private_Uncached_NoHostname()
        {
            await TestHttpsRule("https-private-nohostname", HiveHostPorts.ProxyPrivateLastUserPort, HiveConst.PrivateNetwork, hive.PrivateLoadBalancer);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Https_Private_Cached_DefaultPort()
        {
            await TestHttpsRule("https-private-cached-defaultport", HiveHostPorts.ProxyPrivateHttps, HiveConst.PrivateNetwork, hive.PrivateLoadBalancer, useCache: true);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Https_Private_Cached_NonDefaultPort()
        {
            await TestHttpsRule("https-private-cached-nondefaultport", HiveHostPorts.ProxyPrivateLastUserPort, HiveConst.PrivateNetwork, hive.PrivateLoadBalancer, useCache: true);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Https_Private_Cached_NoHostname()
        {
            await TestHttpsRule("https-private-cached-nohostname", HiveHostPorts.ProxyPrivateLastUserPort, HiveConst.PrivateNetwork, hive.PrivateLoadBalancer, useCache: true);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Https_Private_Uncached_MultiFrontends_DefaultPort()
        {
            await TestHttpsMultipleFrontends("https-private-multifrontends-defaultport", new string[] { $"test-1.{testHostname}", $"test-2.{testHostname}" }, HiveHostPorts.ProxyPrivateHttps, HiveConst.PrivateNetwork, hive.PrivateLoadBalancer);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Https_Private_Uncached_MultiFrontends_NonDefaultPort()
        {
            await TestHttpsMultipleFrontends("https-private-multifrontend-nondefaultport", new string[] { $"test-1.{testHostname}", $"test-2.{testHostname}" }, HiveHostPorts.ProxyPrivateLastUserPort, HiveConst.PrivateNetwork, hive.PrivateLoadBalancer);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Https_Private_Cached_MultiFrontends_DefaultPort()
        {
            await TestHttpsMultipleFrontends("https-private-multifrontend-defaultport", new string[] { $"test-1.{testHostname}", $"test-2.{testHostname}" }, HiveHostPorts.ProxyPrivateHttps, HiveConst.PrivateNetwork, hive.PrivateLoadBalancer, useCache: true);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Https_Private_Cached_MultiFrontends_NonDefaultPortd()
        {
            await TestHttpsMultipleFrontends("https-private-multifrontend-nondefaultport", new string[] { $"test-1.{testHostname}", $"test-2.{testHostname}" }, HiveHostPorts.ProxyPrivateLastUserPort, HiveConst.PrivateNetwork, hive.PrivateLoadBalancer, useCache: true);
        }
    }
}
