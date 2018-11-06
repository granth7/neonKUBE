﻿//-----------------------------------------------------------------------------
// FILE:	    Test_HiveLoadBalancer.cs
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
        public async Task Http_Public_Uncached_DefaultPort()
        {
            await TestHttpRule("http-public-defaultport", HiveHostPorts.ProxyPublicHttp, HiveConst.PublicNetwork, hive.PublicLoadBalancer);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Http_Public_Uncached_NonDefaultPort()
        {
            await TestHttpRule("http-public-nondefaultport", HiveHostPorts.ProxyPublicLastUserPort, HiveConst.PublicNetwork, hive.PublicLoadBalancer);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Http_Public_Uncached_NoHostname()
        {
            await TestHttpRule("http-public-nohostname", HiveHostPorts.ProxyPublicLastUserPort, HiveConst.PublicNetwork, hive.PublicLoadBalancer, useIPAddress: true);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Http_Public_Cached_DefaultPort()
        {
            await TestHttpRule("http-public-cached-defaultport", HiveHostPorts.ProxyPublicHttp, HiveConst.PublicNetwork, hive.PublicLoadBalancer, useCache: true);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Http_Public_Cached_NonDefaultPort()
        {
            await TestHttpRule("http-public-cached-nondefaultport", HiveHostPorts.ProxyPublicLastUserPort, HiveConst.PublicNetwork, hive.PublicLoadBalancer, useCache: true);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Http_Public_Cached_NoHostname()
        {
            await TestHttpRule("http-public-cached-nohostname", HiveHostPorts.ProxyPublicLastUserPort, HiveConst.PublicNetwork, hive.PublicLoadBalancer, useCache: true, useIPAddress: true);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Http_Public_Uncached_MultiHostnames_DefaultPort()
        {
            await TestHttpMultipleFrontends("http-public-multihostnames-defaultport", new string[] { "vegomatic1.test", "vegomatic2.test" }, HiveHostPorts.ProxyPublicHttp, HiveConst.PublicNetwork, hive.PublicLoadBalancer);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Http_Public_Uncached_MultiHostnames_NonDefaultPort()
        {
            await TestHttpMultipleFrontends("http-public-multihostnames-nondefaultport", new string[] { "vegomatic1.test", "vegomatic2.test" }, HiveHostPorts.ProxyPublicLastUserPort, HiveConst.PublicNetwork, hive.PublicLoadBalancer);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Http_Public_Cached_MultiHostnames_DefaultPort()
        {
            await TestHttpMultipleFrontends("http-public-multihostnames-defaultport", new string[] { "vegomatic1.test", "vegomatic2.test" }, HiveHostPorts.ProxyPublicHttp, HiveConst.PublicNetwork, hive.PublicLoadBalancer, useCache: true);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHive)]
        public async Task Http_Public_Cached_MultiHostnames_NondefaultPort()
        {
            await TestHttpMultipleFrontends("http-public-multihostnames-nondefaultport", new string[] { "vegomatic1.test", "vegomatic2.test" }, HiveHostPorts.ProxyPublicLastUserPort, HiveConst.PublicNetwork, hive.PublicLoadBalancer, useCache: true);
        }
    }
}
