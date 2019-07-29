﻿//-----------------------------------------------------------------------------
// FILE:        Test_EndToEnd.cs
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
// limitations under the License.Gskip

// Uncomment this to enable all tests.
#define SKIP_SLOW_TESTS

using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Reflection;
using System.Security.Cryptography;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

using Neon.Cadence;
using Neon.Cadence.Internal;
using Neon.Common;
using Neon.Cryptography;
using Neon.Data;
using Neon.IO;
using Neon.Time;
using Neon.Xunit;
using Neon.Xunit.Cadence;

using Xunit;

namespace TestCadence
{
    /// <summary>
    /// Tests low-level <see cref="CadenceClient"/> functionality against the <b>cadence-proxy</b>.
    /// </summary>
    public sealed class Test_EndToEnd : IClassFixture<CadenceFixture>, IDisposable
    {
        const int maxWaitSeconds = 5;

        private static readonly TimeSpan allowedVariation = TimeSpan.FromMilliseconds(1000);
        private static readonly TimeSpan workflowTimeout  = TimeSpan.FromSeconds(20);

        //---------------------------------------------------------------------
        // Common workflow and activity classes.

        /// <summary>
        /// <para>
        /// A very simple workflow that accepts an optional UTF-8 encoded argument
        /// string that controls what the workflow does:
        /// </para>
        /// <list type="table">
        /// <item>
        ///     <term><b>null</b></term>
        ///     <description>
        ///     Returns <b>workflow: Hello World!</b> encoded as UTF-8 directly from the workflow.
        ///     </description>
        /// </item>
        /// <item>
        ///     <term><b>"activity"</b></term>
        ///     <description>
        ///     Returns <b>activity: Hello World!</b> encoded as UTF-8 from a child activity.
        ///     </description>
        /// </item>
        /// <item>
        ///     <term><b>"local-activity"</b></term>
        ///     <description>
        ///     Returns <b>local-activity: Hello World!</b> encoded as UTF-8 from a <b>local</b> child activity.
        ///     </description>
        /// </item>
        /// <item>
        ///     <term><b>anything else</b></term>
        ///     <description>
        ///     Returns the string passed directly from the workflow.
        ///     </description>
        /// </item>
        /// </list>
        /// </summary>
        private class HelloWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                string arg = null;

                if (args != null)
                {
                    arg = Encoding.UTF8.GetString(args);
                }

                if (arg == null)
                {
                    return await Task.FromResult(Encoding.UTF8.GetBytes("workflow: Hello World!"));
                }
                else if (arg == "activity")
                {
                    return await CallActivityAsync<HelloActivity>(Encoding.UTF8.GetBytes("activity: Hello World!"));
                }
                else if (arg == "local-activity")
                {
                    return await CallLocalActivityAsync<HelloActivity>(Encoding.UTF8.GetBytes("local-activity: Hello World!"));
                }
                else
                {
                    return await Task.FromResult(Encoding.UTF8.GetBytes(arg));
                }
            }
        }

        /// <summary>
        /// An automatically registerable workflow that calls the 
        /// <see cref="AutoHelloActivity"/> and returns "Hello World!".
        /// </summary>
        [AutoRegister]
        private class AutoHelloWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                return await CallActivityAsync<AutoHelloActivity>();
            }
        }

        /// <summary>
        /// An automatically registerable simple workflow with a custom type name that calls
        /// the  <see cref="CustomAutoHelloActivity"/> and returns "Hello World!".
        /// </summary>
        [AutoRegister("CustomAutoHelloWorkflow")]
        private class CustomAutoHelloWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                return await CallActivityAsync("CustomAutoHelloActivity");
            }
        }

        /// <summary>
        /// This workflow does the same thing as <see cref="HelloWorkflow"/> except that it
        /// executes the child activities by name and not type.
        /// </summary>
        private class HelloWorkflowByName : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                string arg = null;

                if (args != null)
                {
                    arg = Encoding.UTF8.GetString(args);
                }

                if (arg == null)
                {
                    return await Task.FromResult(Encoding.UTF8.GetBytes("workflow: Hello World!"));
                }
                else if (arg == "activity")
                {
                    return await CallActivityAsync("hello-activity", Encoding.UTF8.GetBytes("activity: Hello World!"));
                }
                else if (arg == "local-activity")
                {
                    // NOTE: It's not possible to call local activities by name so we'll
                    // use the type here instead.

                    return await CallLocalActivityAsync<HelloActivity>(Encoding.UTF8.GetBytes("local-activity: Hello World!"));
                }
                else
                {
                    return await Task.FromResult(Encoding.UTF8.GetBytes(arg));
                }
            }
        }

        /// <summary>
        /// Test activity that returns the argument passed.
        /// </summary>
        private class HelloActivity : ActivityBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                return await Task.FromResult(args);
            }
        }

        /// <summary>
        /// Automatically registerable test activity that returns the argument passed.
        /// </summary>
        [AutoRegister]
        private class AutoHelloActivity : ActivityBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                return await Task.FromResult(Encoding.UTF8.GetBytes("Hello World!"));
            }
        }

        /// <summary>
        /// Automatically registerable test activity with a custom name that returns the argument passed.
        /// </summary>
        [AutoRegister("CustomAutoHelloActivity")]
        private class CustomAutoHelloActivity : ActivityBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                return await Task.FromResult(Encoding.UTF8.GetBytes("Hello World!"));
            }
        }

        /// <summary>
        /// Returns the activity properties as an encoded dictionary with the
        /// property values converted to strings.
        /// </summary>
        private class GetPropertiesActivity : ActivityBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var properties = new Dictionary<string, string>();

                properties.Add(nameof(Info.TaskToken), Convert.ToBase64String(Info.TaskToken));
                properties.Add(nameof(Info.WorkflowTypeName), Info.WorkflowTypeName);
                properties.Add(nameof(Info.WorkflowDomain), Info.WorkflowDomain);
                properties.Add(nameof(Info.WorkflowRun), NeonHelper.JsonSerialize(Info.WorkflowRun));
                properties.Add(nameof(Info.ActivityId), Info.ActivityId);
                properties.Add(nameof(Info.ActivityTypeName), Info.ActivityTypeName);
                properties.Add(nameof(Info.TaskList), Info.TaskList);
                properties.Add(nameof(Info.HeartbeatTimeout), Info.HeartbeatTimeout.ToString());
                properties.Add(nameof(Info.ScheduledTimeUtc), Info.ScheduledTimeUtc.ToString());
                properties.Add(nameof(Info.StartedTimeUtc), Info.StartedTimeUtc.ToString());
                properties.Add(nameof(Info.DeadlineTimeUtc), Info.DeadlineTimeUtc.ToString());
                properties.Add(nameof(Info.Attempt), Info.Attempt.ToString());

                return await Task.FromResult(NeonHelper.JsonSerializeToBytes(properties));
            }
        }

        /// <summary>
        /// Executes <see cref="HelloWorkflow"/> as a child to return "workflow: Hello World!".
        /// </summary>
        private class ExecuteChildWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                return await CallChildWorkflowAsync<HelloWorkflow>(args: null);
            }
        }

        /// <summary>
        /// Verifies the workflow variables work.
        /// </summary>
        private class VariablesWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                // Test-0: Uninitialized variables should return NULL.

                var value = await GetVariableAsync("value-0");

                if (value != null)
                {
                    throw new Exception("Test-0: Uninitialized value is not NULL.");
                }

                // Test-1: Set and retrieve a variable.

                await SetVariableAsync("value-0", new byte[] { 1 });

                value = await GetVariableAsync("value-0");

                if (value[0] != 1)
                {
                    throw new Exception("Test-1: Unexpected value.");
                }

                // Test-1: Update an existing variable.

                await SetVariableAsync("value-0", new byte[] { 2 });

                value = await GetVariableAsync("value-0");

                if (value[0] != 2)
                {
                    throw new Exception("Test-1: Unexpected value.");
                }

                //-------------------------------------------------------------
                // Repeat for another variable.

                // Test-2: Uninitialized variables should return NULL.

                value = await GetVariableAsync("value-2");

                if (value != null)
                {
                    throw new Exception("Test-2: Uninitialized value is not NULL.");
                }

                // Test-3: Set and retrieve a variable.

                await SetVariableAsync("value-2", new byte[] { 20 });

                value = await GetVariableAsync("value-2");

                if (value[0] != 20)
                {
                    throw new Exception("Test-3: Unexpected value.");
                }

                // Test-4: Update the variable.

                await SetVariableAsync("value-2", new byte[] { 30 });

                value = await GetVariableAsync("value-2");

                if (value[0] != 30)
                {
                    throw new Exception("Test-4: Unexpected value.");
                }

                //-------------------------------------------------------------
                // Test-5: Verify that we can set and get NULL variable values.

                await SetVariableAsync("value-2", null);

                value = await GetVariableAsync("value-2");

                if (value == null)
                {
                    throw new Exception("Test-4: Unexpected value.");
                }

                return await Task.FromResult((byte[])null);
            }
        }

        /// <summary>
        /// Workflow to test if workflows are running in parallel
        /// </summary>
        private class ParallelWorkflow : WorkflowBase
        {
            private static object syncLock  = new object();
            public static int runningCount  = 0;
            public static int maxParallel   = 0;

            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var sleepSeconds = NeonHelper.JsonDeserialize<Double>(args);
                lock (syncLock)
                {
                    runningCount++;
                    maxParallel  = Math.Max(runningCount, maxParallel);
                }

                await SleepAsync(TimeSpan.FromSeconds(sleepSeconds));

                lock (syncLock)
                {
                    runningCount--;
                }

                return await Task.FromResult((byte[])null);
            }
        }

        /// <summary>
        /// Runs two child workflows in parallel and returns "Hello World!" as the
        /// result.
        /// </summary>
        private class ParallelChildWorkflows : WorkflowBase
        {
            public static int numChildWorkflows = 3;

            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var sleepTime1  = 10.00;
                var sleepTime2  = 15.00;
                var sleepTime3  = 5.00;

                var child1      = await StartChildWorkflowAsync<ParallelWorkflow>(args: NeonHelper.JsonSerializeToBytes(sleepTime1));
                var child2      = await StartChildWorkflowAsync<ParallelWorkflow>(args: NeonHelper.JsonSerializeToBytes(sleepTime2));
                var child3      = await StartChildWorkflowAsync<ParallelWorkflow>(args: NeonHelper.JsonSerializeToBytes(sleepTime3));

                await WaitForChildWorkflowAsync(child1);
                await WaitForChildWorkflowAsync(child2);
                await WaitForChildWorkflowAsync(child3);
                
                return Encoding.UTF8.GetBytes($"Great Parallel Sleep!");
            }
        }

        /// <summary>
        /// Returns the workflow properties as an encoded dictionary with the
        /// property values converted to strings.
        /// </summary>
        private class GetPropertiesWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var properties = new Dictionary<string, string>();

                properties.Add(nameof(Domain), Domain);
                properties.Add(nameof(WorkflowId), WorkflowId);
                properties.Add(nameof(RunId), RunId);
                properties.Add(nameof(WorkflowTypeName), WorkflowTypeName);
                properties.Add(nameof(TaskList), TaskList);

                return await Task.FromResult(NeonHelper.JsonSerializeToBytes(properties));
            }
        }

        /// <summary>
        /// Calls <see cref="GetActivityPropertiesWorkflow"/> and returns the property
        /// dictionary returned by that activity.
        /// </summary>
        private class GetActivityPropertiesWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                return await CallActivityAsync<GetPropertiesActivity>(null);
            }
        }

        /// <summary>
        /// Returns the current time (UTC) as JSON.
        /// </summary>
        private class GetUtcNowWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                return NeonHelper.JsonSerializeToBytes(await UtcNowAsync());
            }
        }

        /// <summary>
        /// Sleeps for the timespan passed as JSON.  The result is a list of two
        /// DateTime instances.  The first is the time UTC just before sleeping
        /// and the second is the time just after sleeping.
        /// </summary>
        private class SleepWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var sleepTime  = NeonHelper.JsonDeserialize<TimeSpan>(args);
                var beforeTime = DateTime.UtcNow;

                await SleepAsync(sleepTime);

                var afterTime = DateTime.UtcNow;
                var times     = new List<DateTime>() { beforeTime, afterTime };
                
                return NeonHelper.JsonSerializeToBytes(times);
            }
        }

        /// <summary>
        /// Sleeps until the time passed as JSON.  The result is a list of two
        /// DateTime instances.  The first is the time UTC just before sleeping
        /// and the second is the time just after sleeping.
        /// </summary>
        private class SleepUntilWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var wakeTimeUtc = NeonHelper.JsonDeserialize<DateTime>(args);
                var beforeTime  = DateTime.UtcNow;

                await SleepUntilUtcAsync(wakeTimeUtc);

                var afterTime = DateTime.UtcNow;
                var times     = new List<DateTime>() { beforeTime, afterTime };

                return NeonHelper.JsonSerializeToBytes(times);
            }
        }

        /// <summary>
        /// This workflow is never registered and is used for testing related scenarios.
        /// </summary>
        private class UnregisteredWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                return await Task.FromResult((byte[])null);
            }
        }

        /// <summary>
        /// This workflow restarts if a non-null argument is passed and also keeps track of
        /// the number of times the workflow was executed.
        /// </summary>
        private class RestartableWorkflow : WorkflowBase
        {
            public static int ExecutionCount = 0;

            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                ExecutionCount++;

                if (args != null)
                {
                    // Pass [args=null] so the next run won't restart.

                    await RestartAsync(null);
                }

                return await Task.FromResult(Encoding.UTF8.GetBytes("Hello World!"));
            }
        }

        /// <summary>
        /// This workflow minimally exercises the GOLANG <b>GetVersion()</b> API.
        /// </summary>
        [AutoRegister]
        private class GetVersionWorkflow : WorkflowBase
        {
            /// <summary>
            /// This simply calls <c>GetVersion("my-change", DefaultVersion, 1)</c> and returns
            /// the result integer (which should be <b>1</b> encoded as a UTF-8 string).
            /// </summary>
            /// <param name="args"></param>
            /// <returns></returns>
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var version = await GetVersionAsync("my-change", DefaultVersion, 1);

                return await Task.FromResult(Encoding.UTF8.GetBytes(version.ToString()));
            }
        }

        private static List<string> cronCalls = new List<string>();

        /// <summary>
        /// Called by <see cref="CronWorkflow"/>, passing the number of times the workflow has
        /// been run, encoded as a UTF-8 string.  This value will be added to the static
        /// <see cref="cronCalls"/> list, which will be verified by the actualy unit test.
        /// </summary>
        [AutoRegister]
        private class CronActivity : ActivityBase
        {
            protected override async Task<byte[]> RunAsync(byte[] args)
            {
                cronCalls.Add(Encoding.UTF8.GetString(args));

                return await Task.FromResult((byte[])null);
            }
        }

        /// <summary>
        /// This workflow is designed to be deployed as a CRON workflow and will call 
        /// the <see cref="CronActivity"/> to record test information about each CRON
        /// workflow run.
        /// </summary>
        [AutoRegister]
        private class CronWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                // We're going to exercise HasPreviousResult() and GetPreviousResult() by recording
                // and incrementing the current run number and then passing it a CronActivity which
                // will add it to the [cronCalls] list which the unit test will verify.

                var callNumber = 0;

                if (await HasPreviousRunResultAsync())
                {
                    callNumber = int.Parse(Encoding.UTF8.GetString(await GetPreviousRunResultAsync()));
                }

                callNumber++;

                var callNumberBytes = (Encoding.UTF8.GetBytes(callNumber.ToString()));

                await CallLocalActivityAsync<CronActivity>(callNumberBytes);

                return await Task.FromResult(callNumberBytes);
            }
        }

        /// <summary>
        /// This workflow tests basic signal reception by setting the value
        /// to be returned by the workflow.  This works by waiting for a period
        /// of time for a signal to be received and then returning the arguments
        /// received with the signal.  The maximum wait time is passed as a 
        /// serialized integer number of seconds.
        /// </summary>
        [AutoRegister]
        private class SignalOnceWorkflow : WorkflowBase
        {
            private byte[] signalArgs;

            [SignalHandler("signal")]
            public async Task OnSignal(byte[] args)
            {
                signalArgs = args;

                await Task.CompletedTask;
            }

            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var maxWaitSeconds = int.Parse(Encoding.UTF8.GetString(args));

                // We're just going to do a simple poll for received signal arguments.

                for (int i = 0; i < maxWaitSeconds; i++)
                {
                    if (signalArgs != null)
                    {
                        break;
                    }

                    await SleepAsync(TimeSpan.FromSeconds(1));
                }

                if (signalArgs == null)
                {
                    throw new CadenceTimeoutException("Timeout waiting for signal.");
                }

                return await Task.FromResult(signalArgs);
            }
        }

        /// <summary>
        /// This workflow tests basic signal reception by waiting for two signals
        /// to be received and serializing the received arguments as a list.
        /// This works by waiting for a period of time for a signals to be received
        /// and then returning the arguments received with the signal.  The maximum
        /// wait time is passed as a serialized integer number of seconds.
        /// </summary>
        [AutoRegister]
        private class SignalTwiceWorkflow : WorkflowBase
        {
            private List<byte[]> signalArgs = new List<byte[]>();

            [SignalHandler("signal")]
            public async Task OnSignal(byte[] args)
            {
                signalArgs.Add(args);

                await Task.CompletedTask;
            }

            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var maxWaitSeconds = int.Parse(Encoding.UTF8.GetString(args));

                // We're just going to do a simple poll for two received signals.

                for (int i = 0; i < maxWaitSeconds; i++)
                {
                    if (signalArgs.Count >= 2)
                    {
                        break;
                    }

                    await SleepAsync(TimeSpan.FromSeconds(1));
                }

                if (signalArgs.Count < 2)
                {
                    throw new CadenceTimeoutException("Timeout waiting for signals.");
                }

                return await Task.FromResult(NeonHelper.JsonSerializeToBytes(signalArgs));
            }
        }

        /// <summary>
        /// This workflow tests basic workflow queries by waiting in a sleep loop
        /// for a specified number of seconds so that the unit test can submit a
        /// query.  The query simply returns the arguments passed.
        /// </summary>
        [AutoRegister]
        private class SimpleQueryWorkflow : WorkflowBase
        {
            [QueryHandler("query")]
            public async Task<byte[]> OnQuery(byte[] args)
            {
                return await Task.FromResult(args);
            }

            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var maxWaitSeconds = int.Parse(Encoding.UTF8.GetString(args));

                // Sleep for a while to give unit tests a chance to send the signal.

                for (int i = 0; i < maxWaitSeconds; i++)
                {
                    await SleepAsync(TimeSpan.FromSeconds(1));
                }

                return await Task.FromResult((byte[])null);
            }
        }

        /// <summary>
        /// Test details passed to <see cref="ActivityHeartbeatWorkflow"/>.
        /// </summary>
        private class ActivityTestArgs
        {
            public string Command { get; set; }
            public ActivityOptions Options { get; set; }
        }

        /// <summary>
        /// Exercises activity heartbeat functionality by executing a
        /// <see cref="HeartbeatActivity"/>.  The arguments will be a serialized
        /// <see cref="ActivityTestArgs"/> instance.
        /// </summary>
        [AutoRegister]
        private class ActivityHeartbeatWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var testArgs = NeonHelper.JsonDeserialize<ActivityTestArgs>(args);

                return await CallActivityAsync<HeartbeatActivity>(args: Encoding.UTF8.GetBytes(testArgs.Command), options: testArgs.Options);
            }
        }

        /// <summary>
        /// <para>
        /// Exercises activity heartbeat and external heartbeat/completion functionality. 
        /// The argument specifies the operation to be perform as a UTF-8 encoded string:
        /// </para>
        /// <list type="table">
        /// <item>
        ///     <term><b>single-heartbeat</b></term>
        ///     <description>
        ///     Simply records a heartbeat and then exits normally.  This just
        ///     verifies that recording a heartbeat doesn't fail.
        ///     </description>
        /// </item>
        /// <item>
        ///     <term><b>multi-heartbeat</b></term>
        ///     <description>
        ///     Records multiple heartbeats over a period longer than the heartbeat
        ///     timeout to ensure that the heartbeats are actually making it to
        ///     Cadence.
        ///     </description>
        /// </item>
        /// <item>
        ///     <term><b>complete-externally</b></term>
        ///     <description>
        ///     Starts a new task that will heartbeat and complete the activity
        ///     externally.  The <see cref="TaskToken"/> static will be set to
        ///     the task token so that the external test can send external heartbeats
        ///     as well as the result.
        ///     </description>
        /// </item>
        /// </list>
        /// <para>
        /// The activity returns the arguments passed for verification.
        /// </para>
        /// </summary>
        [AutoRegister]
        private class HeartbeatActivity : ActivityBase
        {
            public static byte[] TaskToken = null;

            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var command = Encoding.UTF8.GetString(args);

                switch (command)
                {
                    case "single-heartbeat":

                        await base.SendHeartbeatAsync(new byte[] { 1 });
                        break;

                    case "multi-heartbeat":

                        {
                            var heartbeatTimeout  = Info.HeartbeatTimeout;
                            var heartbeatInterval = TimeSpan.FromTicks(heartbeatTimeout.Ticks / 2);

                            for (int i = 0; i < 4; i++)
                            {
                                await Task.Delay(heartbeatInterval);

                                // Note that we're going to alternate between sending null
                                // and non-null details to exercise the method.

                                await SendHeartbeatAsync(NeonHelper.IsOdd(i) ? new byte[] { 0, 1, 2, 3, 4 } : null);
                            }
                        }
                        break;

                    case "complete-externally":

                        TaskToken = Info.TaskToken;
                        await base.CompleteExternallyAsync();
                        break;

                    default:

                        throw new InvalidOperationException($"Unsupported command: {command}");
                }

                return args;
            }
        }

        /// <summary>
        /// Arguments for the <see cref="ChildOperationsWorkflow"/>.
        /// </summary>
        /// <remarks>
        /// <list type="table">
        /// <para>
        /// This class holds test commands and arguments, as described below:
        /// </para>
        /// <item>
        ///     <term><b>signal-child</b></term>
        ///     <description>
        ///     Create a child workflow, signal it and verify that the signal
        ///     was received.
        ///     </description>
        /// </item>
        /// <item>
        ///     <term><b>cancel-child-workflow</b></term>
        ///     <description>
        ///     Create a child workflow that will sleep for a while and then
        ///     cancel the child, verifying that it was cancelled.
        ///     </description>
        /// </item>
        /// <item>
        ///     <term><b>cancel-activity</b></term>
        ///     <description>
        ///     Create a child activity that will sleep for a while and then
        ///     cancel the activity, verifying that is was cancelled.
        ///     </description>
        /// </item>
        /// </list>
        /// <para>
        /// The workflow returns a JSON encoded boolean that will be true if
        /// the test passed, false if it failed.
        /// </para>
        /// </remarks>
        private class ChildOperationsWorkflowArgs
        {
            public string Command { get; set; }
        }

        /// <summary>
        /// Used to test various workflow activity and child workflow operations.
        /// </summary>
        [AutoRegister]
        private class ChildOperationsWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                var command = NeonHelper.JsonDeserialize<ChildOperationsWorkflowArgs>(args);
                var success = false;

                byte[]                      result;
                byte[]                      signalBytes;
                ChildWorkflow               child;
                CancellationTokenSource     cancellationTokenSource = new CancellationTokenSource();

                switch (command.Command)
                {
                    case "signal-child":

                        signalBytes = new byte[] { 10 };
                        child = await StartChildWorkflowAsync<SignalOnceWorkflow>(args: Encoding.UTF8.GetBytes(maxWaitSeconds.ToString()));

                        await SignalChildWorkflowAsync(child, "signal", signalBytes);

                        result  = await this.WaitForChildWorkflowAsync(child);
                        success = NeonHelper.ArrayEquals(signalBytes, result);
                        break;

                    case "cancel-child-workflow":

                        child = await StartChildWorkflowAsync<SignalOnceWorkflow>(args: Encoding.UTF8.GetBytes(30.ToString()), cancellationToken: cancellationTokenSource.Token);

                        await CancelChildWorkflowAsync(child);

                        try
                        {
                            result = await this.WaitForChildWorkflowAsync(child);
                        }
                        catch (CadenceCancelledException)
                        {
                            success = true;
                        }
                        
                        break;

                    case "cancel-activity":

                        var activityTask = CallActivityAsync<AutoHelloActivity>(new byte[] { 0, 1, 2, 3, 4 }, cancellationToken: cancellationTokenSource.Token);

                        cancellationTokenSource.Cancel();

                        result = await activityTask;
                        break;

                    default:

                        throw new InvalidOperationException($"Unsupported command: {command}");
                }

                return NeonHelper.JsonSerializeToBytes(success);
            }
        }

        /// <summary>
        /// Used to test activity cancellation.
        /// </summary>
        [AutoRegister]
        private class CancellableActivity : ActivityBase
        {
            public static bool HasStarted = false;

            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                // Set [HasStarted=true] so that the parent workflow can wait for the
                // activity to be started and monitor the cancellation token to verify
                // that the activity is actually cancelled.  The activity result will
                // be a JSON encoded boolean indicating whether the activity was cancelled
                // (TRUE) or whether it was run to completion (FALSE).

                var cancelled = false;

                CancellationToken.Register(() => cancelled = true);

                HasStarted = true;

                await Task.Delay(TimeSpan.FromSeconds(maxWaitSeconds));

                return await Task.FromResult(NeonHelper.JsonSerializeToBytes(cancelled));
            }
        }

        /// <summary>
        /// This workflow is used to test workflow cancellation and termination.  The workflow
        /// starts execution and then sleeps for 30 seconds, giving the unit test a chance to
        /// perform the operation.
        /// </summary>
        [AutoRegister]
        private class DelayWorkflow : WorkflowBase
        {
            protected async override Task<byte[]> RunAsync(byte[] args)
            {
                await SleepAsync(TimeSpan.FromSeconds(30));

                return null;
            }
        }

        //---------------------------------------------------------------------
        // Test implementations:

        CadenceFixture      fixture;
        CadenceClient       client;
        HttpClient          proxyClient;

        public Test_EndToEnd(CadenceFixture fixture)
        {
            var settings = new CadenceSettings()
            {
                DebugPrelaunched       = false,
                Mode                   = ConnectionMode.ListenOnly,
                Debug                  = true,
                ProxyTimeoutSeconds    = 30.0,
                //DebugHttpTimeout     = TimeSpan.FromSeconds(5),
                DebugDisableHeartbeats = true,
                DebugIgnoreTimeouts    = false
            };
            
            fixture.Start(settings, keepOpen: true);

            this.fixture     = fixture;
            this.client      = fixture.Connection;
            this.proxyClient = new HttpClient() { BaseAddress = client.ProxyUri };
        }

        public void Dispose()
        {
            if (proxyClient != null)
            {
                proxyClient.Dispose();
                proxyClient = null;
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Domain()
        {
            // Exercise the Cadence domain operations.

            //-----------------------------------------------------------------
            // RegisterDomain:

            await client.RegisterDomainAsync("domain-0", "this is domain-0", "jeff@lilltek.com", retentionDays: 14);
            await Assert.ThrowsAsync<CadenceDomainAlreadyExistsException>(async () => await client.RegisterDomainAsync(name: "domain-0"));
            await Assert.ThrowsAsync<CadenceBadRequestException>(async () => await client.RegisterDomainAsync(name: null));

            //-----------------------------------------------------------------
            // DescribeDomain:

            var domainDescribeReply = await client.DescribeDomainAsync("domain-0");

            Assert.False(domainDescribeReply.Configuration.EmitMetrics);
            Assert.Equal(14, domainDescribeReply.Configuration.RetentionDays);
            Assert.Equal("domain-0", domainDescribeReply.DomainInfo.Name);
            Assert.Equal("this is domain-0", domainDescribeReply.DomainInfo.Description);
            Assert.Equal("jeff@lilltek.com", domainDescribeReply.DomainInfo.OwnerEmail);
            Assert.Equal(DomainStatus.Registered, domainDescribeReply.DomainInfo.Status);

            await Assert.ThrowsAsync<CadenceEntityNotExistsException>(async () => await client.DescribeDomainAsync("does-not-exist"));

            //-----------------------------------------------------------------
            // UpdateDomain:

            var updateDomainRequest = new DomainUpdateArgs();

            updateDomainRequest.Options.EmitMetrics    = true;
            updateDomainRequest.Options.RetentionDays  = 77;
            updateDomainRequest.DomainInfo.OwnerEmail  = "foo@bar.com";
            updateDomainRequest.DomainInfo.Description = "new description";

            await client.UpdateDomainAsync("domain-0", updateDomainRequest);

            domainDescribeReply = await client.DescribeDomainAsync("domain-0");

            Assert.True(domainDescribeReply.Configuration.EmitMetrics);
            Assert.Equal(77, domainDescribeReply.Configuration.RetentionDays);
            Assert.Equal("domain-0", domainDescribeReply.DomainInfo.Name);
            Assert.Equal("new description", domainDescribeReply.DomainInfo.Description);
            Assert.Equal("foo@bar.com", domainDescribeReply.DomainInfo.OwnerEmail);
            Assert.Equal(DomainStatus.Registered, domainDescribeReply.DomainInfo.Status);

            await Assert.ThrowsAsync<CadenceEntityNotExistsException>(async () => await client.UpdateDomainAsync("does-not-exist", updateDomainRequest));
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Ping()
        {
            // Verify that Ping works and optionally measure simple transaction throughput.

            await client.PingAsync();

            var stopwatch  = new Stopwatch();
            var iterations = 5000;

            stopwatch.Start();

            for (int i = 0; i < iterations; i++)
            {
                await client.PingAsync();
            }

            stopwatch.Stop();

            var tps = iterations * (1.0 / stopwatch.Elapsed.TotalSeconds);

            Console.WriteLine($"Transactions/sec: {tps}");
        }

#if SKIP_SLOW_TESTS
        [Fact(Skip = "Slow: Enable for full tests")]
#else
        [Fact]
#endif
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public void PingAttack()
        {
            // Measure througput with 4 threads hammering the proxy with pings.

            var syncLock   = new object();
            var totalTps   = 0.0;
            var threads    = new Thread[4];
            var iterations = 5000;

            for (int i = 0; i < threads.Length; i++)
            {
                threads[i] = new Thread(
                    new ThreadStart(
                        () =>
                        {
                            var stopwatch = new Stopwatch();

                            stopwatch.Start();

                            for (int j = 0; j < iterations; j++)
                            {
                                client.PingAsync().Wait();
                            }

                            stopwatch.Stop();

                            var tps = iterations * (1.0 / stopwatch.Elapsed.TotalSeconds);

                            lock (syncLock)
                            {
                                totalTps += tps;
                            }
                        }));

                threads[i].Start();
            }

            foreach (var thread in threads)
            {
                thread.Join();
            }

            Console.WriteLine($"Transactions/sec: {totalTps}");
            Console.WriteLine($"Latency (average): {1.0 / totalTps}");
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Worker()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);

            // Verify that creating workers with the same attributes actually
            // return the pre-existing instance with an incremented reference
            // count.

            var activityWorker1 = await client.StartWorkerAsync("test-domain", "tasks1", new WorkerOptions() { DisableWorkflowWorker = true });

            Assert.Equal(1, activityWorker1.RefCount);

            var activityWorker2 = await client.StartWorkerAsync("test-domain", "tasks1", new WorkerOptions() { DisableWorkflowWorker = true });

            Assert.Same(activityWorker1, activityWorker2);
            Assert.Equal(2, activityWorker2.RefCount);

            var workflowWorker1 = await client.StartWorkerAsync("test-domain", "tasks1", new WorkerOptions() { DisableActivityWorker = true });

            Assert.Equal(1, workflowWorker1.RefCount);

            var workflowWorker2 = await client.StartWorkerAsync("test-domain", "tasks1", new WorkerOptions() { DisableActivityWorker = true });

            Assert.Same(workflowWorker1, workflowWorker2);
            Assert.Equal(2, workflowWorker2.RefCount);

            // Verify the dispose/refcount behavior.

            activityWorker2.Dispose();
            Assert.False(activityWorker2.IsDisposed);
            Assert.Equal(1, activityWorker2.RefCount);

            activityWorker2.Dispose();
            Assert.True(activityWorker2.IsDisposed);
            Assert.Equal(0, activityWorker2.RefCount);

            workflowWorker2.Dispose();
            Assert.False(workflowWorker2.IsDisposed);
            Assert.Equal(1, workflowWorker2.RefCount);

            workflowWorker2.Dispose();
            Assert.True(workflowWorker2.IsDisposed);
            Assert.Equal(0, workflowWorker2.RefCount);

            // Verify that we're not allowed to restart workers.

            await Assert.ThrowsAsync<InvalidOperationException>(async () => await client.StartWorkerAsync("test-domain", "tasks1", new WorkerOptions() { DisableWorkflowWorker = true }));
            await Assert.ThrowsAsync<InvalidOperationException>(async () => await client.StartWorkerAsync("test-domain", "tasks1", new WorkerOptions() { DisableActivityWorker = true }));
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task HelloWorld_Workflow_ByType()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<HelloWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Run a workflow passing NULL args.

                var workflowRun = await client.StartWorkflowAsync<HelloWorkflow>(args: null, "test-domain");
                var result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal("workflow: Hello World!", Encoding.UTF8.GetString(result));

                // Run a workflow passing a string argument.

                var args = Encoding.UTF8.GetBytes("custom args");

                workflowRun = await client.StartWorkflowAsync<HelloWorkflow>(args: args, "test-domain");
                result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal(args, result);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task HelloWorld_Workflow_ByName()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<HelloWorkflowByName>("hello-workflow-by-name");

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Run a workflow passing NULL args.

                var workflowRun = await client.StartWorkflowAsync("hello-workflow-by-name", args: null, domain: "test-domain");
                var result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal("workflow: Hello World!", Encoding.UTF8.GetString(result));

                // Run a workflow passing a string argument.

                var args = Encoding.UTF8.GetBytes("custom args");

                workflowRun = await client.StartWorkflowAsync("hello-workflow-by-name", args: args, domain: "test-domain");
                result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal(args, result);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task HelloWorld_Activity_ByType()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<HelloWorkflow>();
            await client.RegisterActivityAsync<HelloActivity>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Run a workflow that invokes an activity.

                var args        = Encoding.UTF8.GetBytes("activity");
                var workflowRun = await client.StartWorkflowAsync<HelloWorkflow>(args: Encoding.UTF8.GetBytes("activity"), domain: "test-domain");
                var result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal(Encoding.UTF8.GetBytes("activity: Hello World!"), result);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task HelloWorld_Activity_ByName()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<HelloWorkflowByName>("hello-workflow-by-name");
            await client.RegisterActivityAsync<HelloActivity>("hello-activity");

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Run a workflow that invokes an activity.

                var args        = Encoding.UTF8.GetBytes("activity");
                var workflowRun = await client.StartWorkflowAsync("hello-workflow-by-name", args: Encoding.UTF8.GetBytes("activity"), domain: "test-domain");
                var result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal(Encoding.UTF8.GetBytes("activity: Hello World!"), result);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task HelloWorld_LocalActivity_ByType()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<HelloWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Run a workflow that invokes an activity.

                var args        = Encoding.UTF8.GetBytes("local-activity");
                var workflowRun = await client.StartWorkflowAsync<HelloWorkflow>(args: Encoding.UTF8.GetBytes("local-activity"), domain: "test-domain");
                var result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal(Encoding.UTF8.GetBytes("local-activity: Hello World!"), result);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task HelloWorld_ChildWorkflow_ByType()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<ExecuteChildWorkflow>();
            await client.RegisterWorkflowAsync<HelloWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Run a workflow that invokes a child workflow.

                var args        = Encoding.UTF8.GetBytes("local-activity");
                var workflowRun = await client.StartWorkflowAsync<ExecuteChildWorkflow>(args: null, domain: "test-domain");

                var result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal(Encoding.UTF8.GetBytes("workflow: Hello World!"), result);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_NonMutableValue()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<VariablesWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Verify that non-mutable workflow values work as expected.
                // The workflow will throw an exception if there's a problem.

                var workflowRun = await client.StartWorkflowAsync<VariablesWorkflow>(args: null, domain: "test-domain");

                await client.GetWorkflowResultAsync(workflowRun);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Variables()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<VariablesWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Verify that mutable workflow values work as expected.
                // The workflow will throw an exception if there's a problem.

                var workflowRun = await client.StartWorkflowAsync<VariablesWorkflow>(args: null, domain: "test-domain");

                await client.GetWorkflowResultAsync(workflowRun);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_ParallelChildWorkflows()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<ParallelChildWorkflows>();
            await client.RegisterWorkflowAsync<ParallelWorkflow>();

            ParallelWorkflow.maxParallel    = 0;
            ParallelWorkflow.runningCount   = 0;

            using (await client.StartWorkerAsync("test-domain"))
            {
                var workflowRun = await client.StartWorkflowAsync<ParallelChildWorkflows>(args: null, domain: "test-domain");
                var result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal("Great Parallel Sleep!", Encoding.UTF8.GetString(result));
                Assert.Equal(ParallelChildWorkflows.numChildWorkflows, ParallelWorkflow.maxParallel);
                Assert.Equal(0, ParallelWorkflow.runningCount);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Properties_DefaultTaskList()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<GetPropertiesWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                var workflowRun = await client.StartWorkflowAsync<GetPropertiesWorkflow>(args: null, domain: "test-domain", options: new WorkflowOptions() { WorkflowId = "my-workflow-default" });
                var result      = await client.GetWorkflowResultAsync(workflowRun);
                var properties  = NeonHelper.JsonDeserialize<Dictionary<string, string>>(result);

                Assert.Equal("test-domain", properties["Domain"]);
                Assert.Equal("my-workflow-default", properties["WorkflowId"]);
                Assert.NotNull(properties["RunId"]);
                Assert.NotEmpty(properties["RunId"]);
                Assert.NotEqual("my-workflow-default", properties["RunId"]);
                Assert.Equal(typeof(GetPropertiesWorkflow).FullName, properties["WorkflowTypeName"]);
                Assert.Equal("default", properties["TaskList"]);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Properties_NonDefaultTasklist()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<GetPropertiesWorkflow>();

            using (await client.StartWorkerAsync("test-domain", "non-default"))
            {
                var workflowRun = await client.StartWorkflowAsync<GetPropertiesWorkflow>(args: null, domain: "test-domain", taskList: "non-default", options: new WorkflowOptions() { WorkflowId = "my-workflow-nondefault" });
                var result      = await client.GetWorkflowResultAsync(workflowRun);
                var properties  = NeonHelper.JsonDeserialize<Dictionary<string, string>>(result);

                Assert.Equal("test-domain", properties["Domain"]);
                Assert.Equal("my-workflow-nondefault", properties["WorkflowId"]);
                Assert.NotNull(properties["RunId"]);
                Assert.NotEmpty(properties["RunId"]);
                Assert.NotEqual("my-workflow-nondefault", properties["RunId"]);
                Assert.Equal(typeof(GetPropertiesWorkflow).FullName, properties["WorkflowTypeName"]);
                Assert.Equal("non-default", properties["TaskList"]);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Activities_Properties()
        {
            // Get the current start time rounded down to the nearest second.
            // We need to do the rounding because the times reported to the
            // activity properties don't always seem to have high precision.

            var testStartUtc = DateTime.UtcNow;

            testStartUtc = new DateTime(testStartUtc.Year, testStartUtc.Month, testStartUtc.Day, testStartUtc.Hour, testStartUtc.Minute, testStartUtc.Second, DateTimeKind.Utc);

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterActivityAsync<GetPropertiesActivity>();
            await client.RegisterWorkflowAsync<GetActivityPropertiesWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                var workflowRun = await client.StartWorkflowAsync<GetActivityPropertiesWorkflow>(args: null, domain: "test-domain", options: new WorkflowOptions() { WorkflowId = "my-workflow-activity-properties" });
                var result      = await client.GetWorkflowResultAsync(workflowRun);
                var properties  = NeonHelper.JsonDeserialize<Dictionary<string, string>>(result);

                Assert.True(!string.IsNullOrEmpty(properties["TaskToken"]));
                Assert.NotNull(Convert.FromBase64String(properties["TaskToken"]));
                Assert.Equal(typeof(GetActivityPropertiesWorkflow).FullName, properties["WorkflowTypeName"]);
                Assert.Equal("test-domain", properties["WorkflowDomain"]);
                Assert.NotNull(NeonHelper.JsonDeserialize<WorkflowRun>(properties["WorkflowRun"]));
                Assert.False(string.IsNullOrEmpty(properties["ActivityId"]));
                Assert.Equal(typeof(GetPropertiesActivity).FullName, properties["ActivityTypeName"]);
                Assert.Equal("default", properties["TaskList"]);
                Assert.Equal("00:00:00", properties["HeartbeatTimeout"]);
                Assert.Equal("0", properties["Attempt"]);

                // $todo(jeff.lill):
                //
                // We're just going to ensure that we can parse the time related
                // properties and verify that all times make sense.

                Assert.NotNull(properties["ScheduledTimeUtc"]);
                Assert.NotNull(properties["StartedTimeUtc"]);
                Assert.NotNull(properties["DeadlineTimeUtc"]);

                var scheduledTimeUtc = DateTime.Parse(properties["ScheduledTimeUtc"]);
                var startedTimeUtc   = DateTime.Parse(properties["StartedTimeUtc"]);
                var deadlineTime     = DateTime.Parse(properties["DeadlineTimeUtc"]);

                Assert.True(testStartUtc <= scheduledTimeUtc);
                Assert.True(testStartUtc <= startedTimeUtc);
                Assert.True(testStartUtc <= deadlineTime);
                Assert.True(scheduledTimeUtc <= startedTimeUtc);
                Assert.True(startedTimeUtc <= deadlineTime);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_UtcNow()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<GetUtcNowWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                var workflowRun    = await client.StartWorkflowAsync<GetUtcNowWorkflow>(args: null, domain: "test-domain");
                var nowJsonBytes   = await client.GetWorkflowResultAsync(workflowRun);
                var workflowNowUtc = NeonHelper.JsonDeserialize<DateTime>(nowJsonBytes);
                var nowUtc         = DateTime.UtcNow;

                Assert.True(nowUtc - workflowNowUtc < allowedVariation);
                Assert.True(workflowNowUtc - nowUtc < allowedVariation);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Sleep()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<SleepWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                var sleepTime = TimeSpan.FromSeconds(5);
                var workflowRun = await client.StartWorkflowAsync<SleepWorkflow>(args: NeonHelper.JsonSerializeToBytes(sleepTime), domain: "test-domain");
                var nowJsonBytes = await client.GetWorkflowResultAsync(workflowRun);
                var times = NeonHelper.JsonDeserialize<List<DateTime>>(nowJsonBytes);

                Assert.True(times[1] > times[0]);
                Assert.True(times[1] - times[0] - sleepTime < allowedVariation);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_SleepUntil()
        {
            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<SleepUntilWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                var sleepTime    = TimeSpan.FromSeconds(5);
                var wakeTime     = DateTime.UtcNow + sleepTime;
                var workflowRun  = await client.StartWorkflowAsync<SleepUntilWorkflow>(args: NeonHelper.JsonSerializeToBytes(wakeTime), domain: "test-domain");
                var nowJsonBytes = await client.GetWorkflowResultAsync(workflowRun);
                var times        = NeonHelper.JsonDeserialize<List<DateTime>>(nowJsonBytes);

                Assert.True(times[1] > times[0]);
                Assert.True(times[1] - times[0] - sleepTime < allowedVariation);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_StartTimeout()
        {
            // Verify that we see a [CadenceTimeoutException] when we try to execute 
            // and unregistered workflow.  This als ensures that the workflow honors
            // [WorkflowOptions.ExecutionStartToCloseTimeout].

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<GetUtcNowWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Warm up Cadence by registering an running another workflow.  This will
                // help make our timeout timing more accurate and repeatable.

                await client.StartWorkflowAsync<GetUtcNowWorkflow>(args: null, domain: "test-domain");

                // This is the actual test.

                var executeTimeout = TimeSpan.FromSeconds(5);
                var startTime      = DateTime.UtcNow;

                try
                {
                    await client.StartWorkflowAsync<UnregisteredWorkflow>(domain: "test-domain", options: new WorkflowOptions() { ExecutionStartToCloseTimeout = executeTimeout });
                }
                catch (CadenceTimeoutException)
                {
                    var endTime = DateTime.UtcNow;

                    // Ensure that [ExecutionStartToCloseTimeout] and that we got the exception
                    // around 5 seconds after we started execution.

                    Assert.True(endTime - startTime <= executeTimeout + TimeSpan.FromSeconds(1));
                }
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_GetResult()
        {
            // Verify that we can retrieve a workflow result after it has completed execution.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<HelloWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Run a workflow passing NULL args and verify.

                var workflowRun = await client.StartWorkflowAsync<HelloWorkflow>(args: null, domain: "test-domain");
                var result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal("workflow: Hello World!", Encoding.UTF8.GetString(result));

                // Now retrieve the result from the completed workflow and verify.

                result = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal("workflow: Hello World!", Encoding.UTF8.GetString(result));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_GetResultError()
        {
            // Verify that an exception is thrown when waiting on a non-existent workflow.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var workflowRun = new WorkflowRun("not-present", "not-here");

                await Assert.ThrowsAsync<CadenceBadRequestException>(async () => await client.GetWorkflowResultAsync(workflowRun));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Restart()
        {
            // Verify that we can a workflow can restart itself.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterWorkflowAsync<RestartableWorkflow>();

            using (await client.StartWorkerAsync("test-domain"))
            {
                // Clear the execution count, run a restarting workflow, and then
                // verify that it executed twice.

                RestartableWorkflow.ExecutionCount = 0;

                var workflowRun = await client.StartWorkflowAsync<RestartableWorkflow>(args: new byte[] { 1 }, domain: "test-domain");
                var result      = await client.GetWorkflowResultAsync(workflowRun);

                Assert.NotNull(result);
                Assert.Equal("Hello World!", Encoding.UTF8.GetString(result));
                Assert.Equal(2, RestartableWorkflow.ExecutionCount);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task AutoRegister()
        {
            var assembly = Assembly.GetExecutingAssembly();

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);

            // Auto registers tagged workflows and activities and then executes them
            // using the default (full) type names.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);
            await client.RegisterAssemblyActivitiesAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var result   = await client.CallWorkflowAsync<AutoHelloWorkflow>(domain: "test-domain");

                Assert.NotNull(result);
                Assert.Equal("Hello World!", Encoding.UTF8.GetString(result));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task AutoRegister_ByName()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // Auto registers tagged workflows and activities and then executes them
            // using custom type names.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);
            await client.RegisterAssemblyActivitiesAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var result = await client.CallWorkflowAsync("CustomAutoHelloWorkflow", domain: "test-domain");

                Assert.NotNull(result);
                Assert.Equal("Hello World!", Encoding.UTF8.GetString(result));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Multiple_Clients()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // We're going to create a second client and then use both of them to
            // execute a workflow.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);
            await client.RegisterAssemblyActivitiesAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                using (var client2 = await CadenceClient.ConnectAsync(client.Settings))
                {
                    await client2.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
                    await client2.RegisterAssemblyWorkflowsAsync(assembly);
                    await client2.RegisterAssemblyActivitiesAsync(assembly);

                    // Client #1 calls

                    var result1 = await client.CallWorkflowAsync<AutoHelloWorkflow>(domain: "test-domain");

                    // Client #2 calls

                    Assert.NotNull(result1);
                    Assert.Equal("Hello World!", Encoding.UTF8.GetString(result1));

                    var result2 = await client2.CallWorkflowAsync<AutoHelloWorkflow>(domain: "test-domain");

                    Assert.NotNull(result2);
                    Assert.Equal("Hello World!", Encoding.UTF8.GetString(result1));
                }
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_GetVersion()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // We're going to create a second client and then use both of them to
            // execute a workflow.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var result = await client.CallWorkflowAsync<GetVersionWorkflow>(domain: "test-domain");

                Assert.Equal("1", Encoding.UTF8.GetString(result));
            }
        }

#if SKIP_SLOW_TESTS
        [Fact(Skip = "Slow: Enable for full tests")]
#else
        [Fact]
#endif
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Cron()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // Start a CRON workflow on a 1 minute interval that updates the [cronCalls] list 
            // every time the workflow is invoked.  We'll wait for the first invocation and then
            // wait to verify that we've see at least 3 invocations and that each invocation 
            // propertly incremented the call number.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);
            await client.RegisterAssemblyActivitiesAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                cronCalls.Clear();      // Clear this to reset any old test state.

                var options = new WorkflowOptions()
                {
                    WorkflowId   = "cron-workflow",
                    CronSchedule = "0/1 * * * *"
                };

                await client.StartWorkflowAsync<CronWorkflow>(domain: "test-domain", options: options);

                // Wait for the the first workflow run.  This is a quicker way to fail if the
                // workflow never runs.

                NeonHelper.WaitFor(() => cronCalls.Count >= 1, timeout: TimeSpan.FromMinutes(1.5));

                // Wait up to 2.5 minutes more for at least two more runs.

                NeonHelper.WaitFor(() => cronCalls.Count >= 3, timeout: TimeSpan.FromMinutes(2.5));

                // Verify that the run numbers look good.

                for ( int i = 1; i <= 3; i++)
                {
                    Assert.Equal(cronCalls[i - 1], i.ToString());
                }
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Signal_Simple()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // Start a workflow, signal it with some test arguments and then
            // verify that the workflow received the signal by checking that
            // it returned the signal arguments passed.
            //
            // This version of the test waits for the workflow to indicate that 
            // it's running before sending the signal.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var result = new byte[] { 10 };
                var run    = await client.StartWorkflowAsync<SignalOnceWorkflow>(domain: "test-domain", args: Encoding.UTF8.GetBytes(maxWaitSeconds.ToString()));

                await NeonHelper.WaitForAsync(async () => (await client.GetWorkflowStateAsync(run)).Execution.HasStarted, TimeSpan.FromSeconds(maxWaitSeconds));
                await client.SignalWorkflowAsync(run, "signal", result);

                Assert.Equal(result, await client.GetWorkflowResultAsync(run));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Signal_NoWait()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // Start a workflow, signal it with some test arguments and then
            // verify that the workflow received the signal by checking that
            // it returned the signal arguments passed.
            //
            // This version of the test doesn't wait for the workflow to indicate 
            // that it's running before sending the signal to ensure that the
            // [cadence-proxy] buffers signals between the time the workflow
            // starts on its side and the .NET client has a chance to begin
            // executing the workflow.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var result = new byte[] { 10 };
                var run = await client.StartWorkflowAsync<SignalOnceWorkflow>(domain: "test-domain", args: Encoding.UTF8.GetBytes(maxWaitSeconds.ToString()));

                await client.SignalWorkflowAsync(run, "signal", result);

                Assert.Equal(result, await client.GetWorkflowResultAsync(run));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Signal_Twice()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // Start a workflow, signal it twice and then verify that the workflow
            // received both signals.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var signal1 = new byte[] { 10 };
                var signal2 = new byte[] { 20 };
                var run     = await client.StartWorkflowAsync<SignalTwiceWorkflow>(domain: "test-domain", args: Encoding.UTF8.GetBytes(maxWaitSeconds.ToString()));

                await NeonHelper.WaitForAsync(async () => (await client.GetWorkflowStateAsync(run)).Execution.HasStarted, TimeSpan.FromSeconds(maxWaitSeconds));

                await client.SignalWorkflowAsync(run, "signal", signal1);
                await client.SignalWorkflowAsync(run, "signal", signal2);

                var result  = await client.GetWorkflowResultAsync(run);
                var signals = NeonHelper.JsonDeserialize<List<byte[]>>(result);

                Assert.Equal(new List<byte[]>() { signal1, signal2 }, signals);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Signal_Child()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // Have a workflow create a child workflow and signal it.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var args = new ChildOperationsWorkflowArgs()
                {
                    Command = "signal-child"
                };

                var result = await client.CallWorkflowAsync<ChildOperationsWorkflow>(NeonHelper.JsonSerializeToBytes(args));

                Assert.True(NeonHelper.JsonDeserialize<bool>(result));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Cancel_Child()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // Have a workflow create a child workflow and then cancel it.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var args = new ChildOperationsWorkflowArgs()
                {
                    Command = "cancel-child-workflow"
                };

                var result = await client.CallWorkflowAsync<ChildOperationsWorkflow>(NeonHelper.JsonSerializeToBytes(args));

                Assert.True(NeonHelper.JsonDeserialize<bool>(result));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Cancel_Activity()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // Have a workflow create a child workflow and then cancel it.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);
            await client.RegisterAssemblyActivitiesAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var args = new ChildOperationsWorkflowArgs()
                {
                    Command = "cancel-activity"
                };

                CancellableActivity.HasStarted = false;

                var workflowTask = client.CallWorkflowAsync<ChildOperationsWorkflow>(NeonHelper.JsonSerializeToBytes(args));

                NeonHelper.WaitFor(() => CancellableActivity.HasStarted, TimeSpan.FromSeconds(30));

                Assert.True(NeonHelper.JsonDeserialize<bool>(await workflowTask));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Query_Once()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // Start a workflow and then query it.  The query should return the 
            // arguments passed.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var args   = new byte[] { 100 };
                var run    = await client.StartWorkflowAsync<SimpleQueryWorkflow>(domain: "test-domain", args: Encoding.UTF8.GetBytes(maxWaitSeconds.ToString()));
                var result = await client.QueryWorkflowAsync(run, "query", args);

                Assert.Equal(args, result);

                await client.GetWorkflowResultAsync(run);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Query_Twice()
        {
            var assembly = Assembly.GetExecutingAssembly();

            // Start a workflow and then query it.  The query should return the 
            // arguments passed.

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var run = await client.StartWorkflowAsync<SimpleQueryWorkflow>(domain: "test-domain", args: Encoding.UTF8.GetBytes(maxWaitSeconds.ToString()));

                Assert.Equal(new byte[] { 1 }, await client.QueryWorkflowAsync(run, "query", new byte[] { 1 }));
                Assert.Equal(new byte[] { 2 }, await client.QueryWorkflowAsync(run, "query", new byte[] { 2 }));

                await client.GetWorkflowResultAsync(run);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Activity_Heartbeat_Single()
        {
            // Verify that running an activity that records a single heartbeat
            // doesn't barf.
            
            var assembly = Assembly.GetExecutingAssembly();

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);
            await client.RegisterAssemblyActivitiesAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var command  = "single-heartbeat";
                var args     = new ActivityTestArgs() { Command = command };
                var argBytes = NeonHelper.JsonSerializeToBytes(args);
                var run      = await client.StartWorkflowAsync<ActivityHeartbeatWorkflow>(domain: "test-domain", args: argBytes);

                await NeonHelper.WaitForAsync(async () => (await client.GetWorkflowStateAsync(run)).Execution.IsClosed, workflowTimeout, TimeSpan.FromSeconds(1));

                var result = await client.GetWorkflowResultAsync(run);

                Assert.Equal(Encoding.UTF8.GetBytes(command), result);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Activity_Heartbeat_Multiple()
        {
            // Verify that running an activity that runs longer than the heartbeat
            // interval but records heartbeats indicating that it's still alive
            // completes properly.
            
            var assembly = Assembly.GetExecutingAssembly();

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);
            await client.RegisterAssemblyActivitiesAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var command  = "multi-heartbeat";
                var args     = new ActivityTestArgs() { Command = command };
                var argBytes = NeonHelper.JsonSerializeToBytes(args);
                var run      = await client.StartWorkflowAsync<ActivityHeartbeatWorkflow>(domain: "test-domain", args: argBytes);

                await NeonHelper.WaitForAsync(async () => (await client.GetWorkflowStateAsync(run)).Execution.IsClosed, workflowTimeout, TimeSpan.FromSeconds(1));

                var result = await client.GetWorkflowResultAsync(run);

                Assert.Equal(Encoding.UTF8.GetBytes(command), result);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Activity_External()
        {
            // Verify that we can heartbeat and complete an activity externally.
            
            var assembly = Assembly.GetExecutingAssembly();

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);
            await client.RegisterAssemblyActivitiesAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                HeartbeatActivity.TaskToken = null;

                var command  = "complete-externally";
                var args     = new ActivityTestArgs() { Command = command };
                var argBytes = NeonHelper.JsonSerializeToBytes(args);
                var run      = await client.StartWorkflowAsync<ActivityHeartbeatWorkflow>(domain: "test-domain", args: argBytes);

                NeonHelper.WaitFor(() => HeartbeatActivity.TaskToken != null, workflowTimeout);

                // Record a couple heartbeats and then complete the activity.

                await client.SendActivityHeartbeatAsync(HeartbeatActivity.TaskToken, null);
                await Task.Delay(TimeSpan.FromSeconds(0.25));
                await client.SendActivityHeartbeatAsync(HeartbeatActivity.TaskToken, new byte[] { 0, 1, 2, 3, 4 });
                await Task.Delay(TimeSpan.FromSeconds(0.25));
                await client.CompleteActivityAsync(HeartbeatActivity.TaskToken, argBytes);

                // Wait for the workflow result.

                var result = await client.GetWorkflowResultAsync(run);

                Assert.Equal(argBytes, result);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Terminate()
        {
            // Verify that a workflow can be terminated.

            var assembly = Assembly.GetExecutingAssembly();

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var run = await client.StartWorkflowAsync<DelayWorkflow>(domain: "test-domain");

                await NeonHelper.WaitForAsync(async () => (await client.GetWorkflowStateAsync(run)).Execution.HasStarted, TimeSpan.FromSeconds(maxWaitSeconds));
                await client.TerminateWorkflowAsync(run, reason: "terminated", details: new byte[] { 0, 1, 2, 3, 4 });
                await NeonHelper.WaitForAsync(async () => (await client.GetWorkflowStateAsync(run)).Execution.IsClosed, TimeSpan.FromSeconds(maxWaitSeconds), TimeSpan.FromSeconds(1));

                var state = await client.GetWorkflowStateAsync(run);

                Assert.Equal(WorkflowCloseStatus.Terminated, state.Execution.WorkflowCloseStatus);
                Assert.Equal(new byte[] { 0, 1, 2, 3, 4 }, await client.GetWorkflowResultAsync(run));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCadence)]
        public async Task Workflow_Cancel()
        {
            // Verify that a workflow can be cancelled.

            var assembly = Assembly.GetExecutingAssembly();

            await client.RegisterDomainAsync("test-domain", ignoreDuplicates: true);
            await client.RegisterAssemblyWorkflowsAsync(assembly);

            using (await client.StartWorkerAsync("test-domain"))
            {
                var run = await client.StartWorkflowAsync<DelayWorkflow>(domain: "test-domain");

                await NeonHelper.WaitForAsync(async () => (await client.GetWorkflowStateAsync(run)).Execution.HasStarted, TimeSpan.FromSeconds(maxWaitSeconds));
                await client.CancelWorkflowAsync(run);
                await NeonHelper.WaitForAsync(async () => (await client.GetWorkflowStateAsync(run)).Execution.IsClosed, TimeSpan.FromSeconds(maxWaitSeconds), TimeSpan.FromSeconds(1));
                await NeonHelper.WaitForAsync(async () => (await client.GetWorkflowStateAsync(run)).Execution.WorkflowCloseStatus == WorkflowCloseStatus.Cancelled, TimeSpan.FromSeconds(maxWaitSeconds), TimeSpan.FromSeconds(1));

                var state = await client.GetWorkflowStateAsync(run);

                Assert.Equal(WorkflowCloseStatus.Cancelled, state.Execution.WorkflowCloseStatus);
            }
        }
    }
}
