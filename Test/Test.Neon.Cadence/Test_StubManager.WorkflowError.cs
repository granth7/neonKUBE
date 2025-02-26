﻿//-----------------------------------------------------------------------------
// FILE:        Test_StubManager.WorkflowError.cs
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

        public interface IErrorGenericWorkflow<T> : IWorkflowBase
        {
            [WorkflowMethod]
            Task DoIt();
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowGenericsNotAllowed()
        {
            // We don't support workflow interfaces with generic parameters.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IErrorGenericWorkflow<int>>(client));
        }

        //---------------------------------------------------------------------

        public interface IErrorNoEntryPointWorkflow : IWorkflowBase
        {
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowNoEntryPoint()
        {
            // Workflows need to have at least one entry point.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IErrorNoEntryPointWorkflow>(client));
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowNullClient()
        {
            // A non-NULL client is required.

            Assert.Throws<ArgumentNullException>(() => StubManager.CreateWorkflowStub<IErrorNoEntryPointWorkflow>(null));
        }

        //---------------------------------------------------------------------

        public class IErrorNotInterfaceWorkflow : WorkflowBase
        {
            [WorkflowMethod]
            public async Task EntryPoint()
            {
                await Task.CompletedTask;
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowNotInterface()
        {
            // Only workflow interfaces are allowed.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IErrorNotInterfaceWorkflow>(client));
        }

        //---------------------------------------------------------------------

        internal class IErrorNotPublicWorkflow : WorkflowBase
        {
            [WorkflowMethod]
            public async Task EntryPoint()
            {
                await Task.CompletedTask;
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowNotPublic()
        {
            // Workflow interfaces must be public.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IErrorNotPublicWorkflow>(client));
        }

        //---------------------------------------------------------------------

        public interface IErrorNonTaskEntryPointWorkflow1 : IWorkflowBase
        {
            [WorkflowMethod]
            void EntryPoint();
        }

        public interface IErrorNonTaskEntryPointWorkflow2 : IWorkflowBase
        {
            [WorkflowMethod]
            List<int> EntryPoint();
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowNonTaskEntryPoint()
        {
            // Workflow entry points methods need to return a Task.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IErrorNonTaskEntryPointWorkflow1>(client));
            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IErrorNonTaskEntryPointWorkflow2>(client));
        }

        //---------------------------------------------------------------------

        public interface IErrorNonTaskSignalWorkflow : IWorkflowBase
        {
            [WorkflowMethod]
            Task EntryPoint();

            [SignalMethod("my-signal")]
            void Signal();
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowNonTaskSignal()
        {
            // Workflow signal methods need to return a Task.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IErrorNonTaskSignalWorkflow>(client));
        }

        //---------------------------------------------------------------------

        public interface IDuplicateDefaultEntryPointsWorkflow : IWorkflowBase
        {
            [WorkflowMethod]
            Task EntryPoint1();

            [WorkflowMethod]
            Task EntryPoint2();
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowDuplicateDuplicateDefaultEntryPoints()
        {
            // Verify that we detect duplicate entrypoint methods
            // with the default name.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IDuplicateDefaultEntryPointsWorkflow>(client));
        }

        //---------------------------------------------------------------------

        public interface IDuplicateEntryPointsWorkflow : IWorkflowBase
        {
            [WorkflowMethod(Name = "duplicate")]
            Task EntryPoint1();

            [WorkflowMethod(Name = "duplicate")]
            Task EntryPoint2();
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowDuplicateDuplicateEntryPoints()
        {
            // Verify that we detect duplicate entrypoint methods
            // with explicit names.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IDuplicateEntryPointsWorkflow>(client));
        }

        //---------------------------------------------------------------------

        public interface IErrorDuplicateSignalsWorkflow : IWorkflowBase
        {
            [WorkflowMethod]
            Task EntryPoint();

            [SignalMethod("my-signal")]
            Task Signal1();

            [SignalMethod("my-signal")]
            Task Signal2();
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowDuplicateSignals()
        {
            // Verify that we detect duplicate signal names.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IErrorDuplicateSignalsWorkflow>(client));
        }

        //---------------------------------------------------------------------

        public interface IErrorNonTaskQueryWorkflow : IWorkflowBase
        {
            [WorkflowMethod]
            Task EntryPoint();

            [QueryMethod("my-query")]
            void Query();
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowNonTaskQuery()
        {
            // Workflow query methods need to return a Task.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IErrorNonTaskQueryWorkflow>(client));
        }

        //---------------------------------------------------------------------

        public interface IErrorDuplicateQueriesWorkflow : IWorkflowBase
        {
            [WorkflowMethod]
            Task EntryPoint();

            [QueryMethod("my-query")]
            Task Query1();

            [QueryMethod("my-query")]
            Task Query2();
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void Error_WorkflowDuplicateQueries()
        {
            // Verify that we detect duplicate query names.

            Assert.Throws<WorkflowTypeException>(() => StubManager.CreateWorkflowStub<IErrorDuplicateQueriesWorkflow>(client));
        }
    }
}
