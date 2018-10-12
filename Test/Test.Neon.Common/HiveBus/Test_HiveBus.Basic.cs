﻿//-----------------------------------------------------------------------------
// FILE:	    Test_HiveBus.Basic.cs
// CONTRIBUTOR: Jeff Lill
// COPYRIGHT:	Copyright (c) 2016-2018 by neonFORGE, LLC.  All rights reserved.

using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

using Neon.Common;
using Neon.HiveMQ;
using Neon.Xunit;
using Neon.Xunit.RabbitMQ;

using Xunit;

namespace TestCommon
{
    public partial class Test_HiveBus : IClassFixture<RabbitMQFixture>
    {
        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHiveMQ)]
        public void BasicDuplicateSubscription()
        {
            // Verify that the basic channel doesn't allow multiple
            // subscriptions to the same message type.

            using (var bus = fixture.Settings.ConnectHiveBus())
            {
                var channel  = bus.GetBasicChannel("test");

                channel.Consume<TestMessage1>(message => { });

                Assert.Throws<InvalidOperationException>(() => channel.Consume<TestMessage1>(message => { }));
                Assert.Throws<InvalidOperationException>(() => channel.Consume<TestMessage1>((message, context) => { }));
                Assert.Throws<InvalidOperationException>(() => channel.Consume<TestMessage1>(message => Task.CompletedTask));
                Assert.Throws<InvalidOperationException>(() => channel.Consume<TestMessage1>((message, context) => Task.CompletedTask));
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHiveMQ)]
        public void Basic()
        {
            // Verify that we can synchronously publish and consume two 
            // different message types via a basic channel.

            using (var bus = fixture.Settings.ConnectHiveBus())
            {
                var channel   = bus.GetBasicChannel("test");
                var received1 = (TestMessage1)null;
                var received2 = (TestMessage2)null;

                channel.Consume<TestMessage1>(message => received1 = message.Body);
                channel.Consume<TestMessage2>(message => received2 = message.Body);

                channel.Publish(new TestMessage1() { Text = "Hello World!" });
                NeonHelper.WaitFor(() => received1 != null && received1.Text == "Hello World!", timeout: timeout);

                channel.Publish(new TestMessage2() { Text = "Hello World!" });
                NeonHelper.WaitFor(() => received2 != null && received2.Text == "Hello World!", timeout: timeout);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHiveMQ)]
        public void BasicContext()
        {
            // Verify that we can synchronously publish and consume from
            // a basic channel while receiving additional context info.

            using (var bus = fixture.Settings.ConnectHiveBus())
            {
                var channel   = bus.GetBasicChannel("test");
                var received  = (TestMessage1)null;
                var contextOK = false;

                channel.Consume<TestMessage1>(
                    (message, context) =>
                    {
                        received  = message.Body;
                        contextOK = context.Queue == channel.Name;
                    });


                channel.Publish(new TestMessage1() { Text = "Hello World!" });

                NeonHelper.WaitFor(() => received != null && received.Text == "Hello World!" && contextOK, timeout: timeout);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHiveMQ)]
        public async Task BasicAsync()
        {
            // Verify that we can asynchronously publish and consume from
            // a basic channel.

            using (var bus = fixture.Settings.ConnectHiveBus())
            {
                var channel  = bus.GetBasicChannel("test");
                var received = (TestMessage1)null;

                channel.Consume<TestMessage1>(
                    async message =>
                    {
                        received = message.Body;

                        await Task.CompletedTask;
                    });


                await channel.PublishAsync(new TestMessage1() { Text = "Hello World!" });

                NeonHelper.WaitFor(() => received != null && received.Text == "Hello World!", timeout: timeout);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHiveMQ)]
        public async Task BasicContextAsync()
        {
            // Verify that we can asynchronously publish and consume from
            // a basic channel while receiving additional context info.

            using (var bus = fixture.Settings.ConnectHiveBus())
            {
                var channel   = bus.GetBasicChannel("test");
                var received  = (TestMessage1)null;
                var contextOK = false;

                channel.Consume<TestMessage1>(
                    async (message, context) =>
                    {
                        received  = message.Body;
                        contextOK = context.Queue == channel.Name;

                        await Task.CompletedTask;
                    });


                await channel.PublishAsync(new TestMessage1() { Text = "Hello World!" });

                NeonHelper.WaitFor(() => received != null && received.Text == "Hello World!" && contextOK, timeout: timeout);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHiveMQ)]
        public async Task BasicChannels ()
        {
            // Verify that messages published from one channel can be
            // received on another.

            using (var bus = fixture.Settings.ConnectHiveBus())
            {
                var receiveChannel = bus.GetBasicChannel("test");
                var received       = (TestMessage1)null;
                var contextOK      = false;

                receiveChannel.Consume<TestMessage1>(
                    async (message, context) =>
                    {
                        received = message.Body;
                        contextOK = context.Queue == receiveChannel.Name;

                        await Task.CompletedTask;
                    });

                var publishChannel = bus.GetBasicChannel("test");

                await publishChannel.PublishAsync(new TestMessage1() { Text = "Hello World!" });

                NeonHelper.WaitFor(() => received != null && received.Text == "Hello World!" && contextOK, timeout: timeout);
            }
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonHiveMQ)]
        public async Task BasicLoadbalance()
        {
            // Verify that messages are load balanced across multiple consumer
            // channels by sending a bunch of messages and ensuring that all of 
            // the consumers saw at least one message.

            const int channelCount = 10;
            const int messageCount = channelCount * 100;

            var consumerMessages = new List<TestMessage1>[channelCount];

            for (int i = 0; i < consumerMessages.Length; i++)
            {
                consumerMessages[i] = new List<TestMessage1>();
            }

            using (var bus = fixture.Settings.ConnectHiveBus())
            {
                for (int channelID = 0; channelID < channelCount; channelID++)
                {
                    var consumeChannel = bus.GetBasicChannel("test");
                    var id             = channelID;

                    consumeChannel.Consume<TestMessage1>(
                        async message =>
                        {
                            lock (consumerMessages)
                            {
                                consumerMessages[id].Add(message.Body);
                            }

                            await Task.CompletedTask;
                        });
                }

                var publishChannel = bus.GetBasicChannel("test");

                for (int i = 0; i < messageCount; i++)
                {
                    await publishChannel.PublishAsync(new TestMessage1() { Text = "{i}" });
                }

                NeonHelper.WaitFor(() => consumerMessages.Where(cm => cm.Count == 0).IsEmpty(), timeout: timeout);
            }
        }
    }
}
