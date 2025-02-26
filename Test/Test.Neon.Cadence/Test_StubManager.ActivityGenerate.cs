﻿//-----------------------------------------------------------------------------
// FILE:        Test_StubManager.WorkflowGenerate.cs
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
using System.IO;
using System.Linq;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Security.Cryptography;
using System.Text;
using System.Threading.Tasks;

using Neon.Cadence;
using Neon.Cadence.Internal;
using Neon.Common;
using Neon.Cryptography;
using Neon.Data;
using Neon.IO;
using Neon.Xunit;
using Neon.Xunit.Cadence;

using Newtonsoft.Json;
using Xunit;

using Test.Neon.Models;
using Newtonsoft.Json.Linq;

namespace TestCadence
{
    public partial class Test_StubManager
    {
        //---------------------------------------------------------------------

        public interface IActivityEntryVoidNoArgs : IActivityBase
        {
            [WorkflowMethod]
            Task RunAsync();
        }

        public class ActivityEntryVoidNoArgs : ActivityBase, IActivityEntryVoidNoArgs
        {
            public async Task RunAsync()
            {
                await Task.CompletedTask;
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Generate_ActivityEntryVoidNoArgs()
        {
            Assert.NotNull(StubManager.CreateActivityStub<IActivityEntryVoidNoArgs>(client, new DummyWorkflow(), "my-activity"));
            Assert.NotNull(StubManager.CreateLocalActivityStub<IActivityEntryVoidNoArgs, ActivityEntryVoidNoArgs>(client, new DummyWorkflow()));
        }

        //---------------------------------------------------------------------

        public interface IActivityEntryVoidWithArgs : IActivityBase
        {
            [WorkflowMethod]
            Task RunAsync(string arg1, int arg2);
        }

        public class ActivityEntryVoidWithArgs : ActivityBase, IActivityEntryVoidWithArgs
        {
            public async Task RunAsync(string arg1, int arg2)
            {
                await Task.CompletedTask;
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Generate_ActivityEntryVoidWithArgs()
        {
            Assert.NotNull(StubManager.CreateActivityStub<IActivityEntryVoidWithArgs>(client, new DummyWorkflow(), "my-activity"));
            Assert.NotNull(StubManager.CreateLocalActivityStub<IActivityEntryVoidWithArgs, ActivityEntryVoidWithArgs>(client, new DummyWorkflow()));
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Generate_ActivityEntryVoidWithOptions()
        {
            Assert.NotNull(StubManager.CreateActivityStub<IActivityEntryVoidWithArgs>(client, new DummyWorkflow(), "my-activity", options: new ActivityOptions(), domain: "my-domain"));
            Assert.NotNull(StubManager.CreateLocalActivityStub<IActivityEntryVoidWithArgs, ActivityEntryVoidWithArgs>(client, new DummyWorkflow(), options: new LocalActivityOptions()));
        }

        //---------------------------------------------------------------------

        public interface IActivityEntryResultWithArgs : IActivityBase
        {
            [WorkflowMethod]
            Task<int> RunAsync(string arg1, int arg2);
        }

        public class ActivityEntryResultWithArgs : ActivityBase, IActivityEntryResultWithArgs
        {
            public async Task<int> RunAsync(string arg1, int arg2)
            {
                return await Task.FromResult(1);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Generate_ActivityResultWithArgs()
        {
            Assert.NotNull(StubManager.CreateActivityStub<IActivityEntryResultWithArgs>(client, new DummyWorkflow(), "my-activity"));
            Assert.NotNull(StubManager.CreateLocalActivityStub<IActivityEntryResultWithArgs, ActivityEntryResultWithArgs>(client, new DummyWorkflow()));
        }

        //---------------------------------------------------------------------

        public interface IActivitySignalNoArgs : IActivityBase
        {
            [WorkflowMethod]
            Task RunAsync();

            [SignalMethod("my-signal")]
            Task SignalAsync();
        }

        public class ActivitySignalNoArgs : ActivityBase, IActivitySignalNoArgs
        {
            public async Task RunAsync()
            {
                await Task.CompletedTask;
            }

            public async Task SignalAsync()
            {
                await Task.CompletedTask;
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Generate_ActivitySignalNoArgs()
        {
            Assert.NotNull(StubManager.CreateActivityStub<IActivitySignalNoArgs>(client, new DummyWorkflow(), "my-activity"));
            Assert.NotNull(StubManager.CreateLocalActivityStub<IActivitySignalNoArgs, ActivitySignalNoArgs>(client, new DummyWorkflow()));
        }

        //---------------------------------------------------------------------

        public interface IActivitySignalWithArgs : IActivityBase
        {
            [WorkflowMethod]
            Task RunAsync();

            [QueryMethod("my-signal")]
            Task SignalAsync(string arg1, int arg2);
        }

        public class ActivitySignalWithArgs : ActivityBase, IActivitySignalWithArgs
        {
            public async Task RunAsync()
            {
                await Task.CompletedTask;
            }

            public async Task SignalAsync(string arg1, int arg2)
            {
                await Task.CompletedTask;
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Generate_ActivitySignalWithArgs()
        {
            Assert.NotNull(StubManager.CreateActivityStub<IActivitySignalWithArgs>(client, new DummyWorkflow(), "my-activity"));
            Assert.NotNull(StubManager.CreateLocalActivityStub<IActivitySignalWithArgs, ActivitySignalWithArgs>(client, new DummyWorkflow()));
        }

        //---------------------------------------------------------------------

        public interface IActivityQueryVoidNoArgs : IActivityBase
        {
            [WorkflowMethod]
            Task RunOneAsync();

            [QueryMethod("my-query")]
            Task QueryAsync();
        }

        public class ActivityQueryVoidNoArgs : ActivityBase, IActivityQueryVoidNoArgs
        {
            public async Task RunOneAsync()
            {
                await Task.CompletedTask;
            }

            public async Task QueryAsync()
            {
                await Task.CompletedTask;
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Generate_ActivityQueryVoidNoArgs()
        {
            Assert.NotNull(StubManager.CreateActivityStub<IActivityQueryVoidNoArgs>(client, new DummyWorkflow(), "my-activity"));
            Assert.NotNull(StubManager.CreateLocalActivityStub<IActivityQueryVoidNoArgs, ActivityQueryVoidNoArgs>(client, new DummyWorkflow()));
        }

        //---------------------------------------------------------------------

        public interface IActivityQueryVoidWithArgs : IActivityBase
        {
            [WorkflowMethod]
            Task RunOneAsync();

            [QueryMethod("my-query")]
            Task QueryAsync(string arg1, bool arg2);
        }

        public class ActivityQueryVoidWithArgs : ActivityBase, IActivityQueryVoidWithArgs
        {
            public async Task RunOneAsync()
            {
                await Task.CompletedTask;
            }

            public async Task QueryAsync(string arg1, bool arg2)
            {
                await Task.CompletedTask;
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Generate_ActivityQueryVoidWithArgs()
        {
            Assert.NotNull(StubManager.CreateActivityStub<IActivityQueryVoidWithArgs>(client, new DummyWorkflow(), "my-activity"));
            Assert.NotNull(StubManager.CreateLocalActivityStub<IActivityQueryVoidWithArgs, ActivityQueryVoidWithArgs>(client, new DummyWorkflow()));
        }

        //---------------------------------------------------------------------

        public interface IActivityQueryResultWithArgs : IActivityBase
        {
            [WorkflowMethod]
            Task RunOneAsync();

            [QueryMethod("my-query")]
            Task<string> QueryAsync(string arg1, bool arg2);
        }

        public class ActivityQueryResultWithArgs : ActivityBase, IActivityQueryResultWithArgs
        {
            public async Task RunOneAsync()
            {
                await Task.CompletedTask;
            }

            public async Task<string> QueryAsync(string arg1, bool arg2)
            {
                return await Task.FromResult("hello world!");
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Generate_ActivityQueryResultWithArgs()
        {
            Assert.NotNull(StubManager.CreateActivityStub<IActivityQueryResultWithArgs>(client, new DummyWorkflow(), "my-activity"));
            Assert.NotNull(StubManager.CreateLocalActivityStub<IActivityQueryResultWithArgs, ActivityQueryResultWithArgs>(client, new DummyWorkflow()));
        }

        //---------------------------------------------------------------------

        public interface IActivityMultiMethods : IActivityBase
        {
            [WorkflowMethod]
            Task RunAsync();

            [WorkflowMethod(Name = "one")]
            Task<int> RunAsync(string arg1);

            [WorkflowMethod(Name = "two")]
            Task<int> RunAsync(string arg1, string arg2);

            [QueryMethod("my-query1")]
            Task<string> QueryAsync();

            [QueryMethod("my-query2")]
            Task<string> QueryAsync(string arg1);

            [QueryMethod("my-query3")]
            Task<string> QueryAsync(string arg1, string arg2);

            [QueryMethod("my-signal1")]
            Task<string> SignalAsync();

            [QueryMethod("my-signal2")]
            Task<string> SignalAsync(string arg1);

            [QueryMethod("my-signal3")]
            Task<string> SignalAsync(string arg1, string arg2);
        }

        public class ActivityMultiMethods : ActivityBase, IActivityMultiMethods
        {
            public async Task RunAsync()
            {
                await Task.CompletedTask;
            }

            public async Task<int> RunAsync(string arg1)
            {
                return await Task.FromResult(1);
            }

            public async Task<int> RunAsync(string arg1, string arg2)
            {
                return await Task.FromResult(2);
            }

            public async Task<string> QueryAsync()
            {
                return await Task.FromResult("my-query1");
            }

            public async Task<string> QueryAsync(string arg1)
            {
                return await Task.FromResult("my-query2");
            }

            public async Task<string> QueryAsync(string arg1, string arg2)
            {
                return await Task.FromResult("my-query3");
            }

            public async Task<string> SignalAsync()
            {
                return await Task.FromResult("my-signal11");
            }

            public async Task<string> SignalAsync(string arg1)
            {
                return await Task.FromResult("my-signal2");
            }

            public async Task<string> SignalAsync(string arg1, string arg2)
            {
                return await Task.FromResult("my-signal3");
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Generate_ActivityMultiMethods()
        {
            Assert.NotNull(StubManager.CreateActivityStub<IActivityMultiMethods>(client, new DummyWorkflow(), "my-activity"));
            Assert.NotNull(StubManager.CreateLocalActivityStub<IActivityMultiMethods, ActivityMultiMethods>(client, new DummyWorkflow()));
        }
    }
}
