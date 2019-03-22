﻿//-----------------------------------------------------------------------------
// FILE:	    Test_CustomOutput.cs
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
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Net;
using System.Reflection;
using System.Runtime.Serialization;
using System.Text;
using System.Threading.Tasks;

using Neon.CodeGen;
using Neon.Common;
using Neon.Xunit;

using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

using Xunit;

namespace TestCodeGen.CustomOutput
{
    [Target("1")]
    [Target("client")]
    public interface Class1
    {
        string Field { get; set; }
    }

    [Target("2")]
    [Target("client")]
    public interface Class2
    {
        string Field { get; set; }
    }

    [Target("3")]
    public interface Class3
    {
        string Field { get; set; }
    }

    [Target("1")]
    [Target("client")]
    [ServiceModel]
    public interface Service1Controller
    {
        void Hello();
    }

    [Target("2")]
    [ServiceModel]
    public interface Service2Controller
    {
        void World();
    }

    [NoCodeGen]
    public class Test_CustomOutput
    {
        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCodeGen)]
        public void AllTargets()
        {
            // Verify that all types are generated when no targets
            // are specified.

            var settings = new CodeGeneratorSettings()
            {
                SourceNamespace = typeof(Test_CustomOutput).Namespace,
            };

            var generator = new CodeGenerator(settings);
            var output    = generator.Generate(Assembly.GetExecutingAssembly());

            Assert.False(output.HasErrors);

            var assemblyStream = CodeGenerator.Compile(output.SourceCode, "test-assembly", references => CodeGenTestHelper.ReferenceHandler(references));
            var sourceCode     = output.SourceCode;

            Assert.Contains("class Class1", sourceCode);
            Assert.Contains("class Class2", sourceCode);
            Assert.Contains("class Class3", sourceCode);
            Assert.Contains("class Service1", sourceCode);
            Assert.Contains("class Service2", sourceCode);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCodeGen)]
        public void Filter1()
        {
            // Verify that only those types tagged with [Target("1")] are generated.

            var settings = new CodeGeneratorSettings()
            {
                SourceNamespace = typeof(Test_CustomOutput).Namespace,
                Targets         = new List<string>() { "1" }
            };

            var generator = new CodeGenerator(settings);
            var output    = generator.Generate(Assembly.GetExecutingAssembly());

            Assert.False(output.HasErrors);

            var assemblyStream = CodeGenerator.Compile(output.SourceCode, "test-assembly", references => CodeGenTestHelper.ReferenceHandler(references));
            var sourceCode     = output.SourceCode;

            Assert.Contains("class Class1", sourceCode);
            Assert.DoesNotContain("class Class2", sourceCode);
            Assert.DoesNotContain("class Class3", sourceCode);
            Assert.Contains("class Service1", sourceCode);
            Assert.DoesNotContain("class Service2", sourceCode);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCodeGen)]
        public void Filter2()
        {
            // Verify that only those types tagged with [Target("2")] are generated.

            var settings = new CodeGeneratorSettings()
            {
                SourceNamespace = typeof(Test_CustomOutput).Namespace,
                Targets         = new List<string>() { "2" }
            };

            var generator = new CodeGenerator(settings);
            var output    = generator.Generate(Assembly.GetExecutingAssembly());

            Assert.False(output.HasErrors);

            var assemblyStream = CodeGenerator.Compile(output.SourceCode, "test-assembly", references => CodeGenTestHelper.ReferenceHandler(references));
            var sourceCode     = output.SourceCode;

            Assert.DoesNotContain("class Class1", sourceCode);
            Assert.Contains("class Class2", sourceCode);
            Assert.DoesNotContain("class Class3", sourceCode);
            Assert.DoesNotContain("class Service1", sourceCode);
            Assert.Contains("class Service2", sourceCode);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCodeGen)]
        public void FilterClient()
        {
            // Verify that only those types tagged with [Target("client")] are generated.

            var settings = new CodeGeneratorSettings()
            {
                SourceNamespace = typeof(Test_CustomOutput).Namespace,
                Targets         = new List<string>() { "client" }
            };

            var generator = new CodeGenerator(settings);
            var output    = generator.Generate(Assembly.GetExecutingAssembly());

            Assert.False(output.HasErrors);

            var assemblyStream = CodeGenerator.Compile(output.SourceCode, "test-assembly", references => CodeGenTestHelper.ReferenceHandler(references));
            var sourceCode     = output.SourceCode;

            Assert.Contains("class Class1", sourceCode);
            Assert.Contains("class Class2", sourceCode);
            Assert.DoesNotContain("class Class3", sourceCode);
            Assert.Contains("class Service1", sourceCode);
            Assert.DoesNotContain("class Service2", sourceCode);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCodeGen)]
        public void Filter3AndClient()
        {
            // Verify that only those types tagged with [Target("3")] and [Target("client")]
            // are generated.

            var settings = new CodeGeneratorSettings()
            {
                SourceNamespace = typeof(Test_CustomOutput).Namespace,
                Targets         = new List<string>() { "3", "client" }
            };

            var generator = new CodeGenerator(settings);
            var output    = generator.Generate(Assembly.GetExecutingAssembly());

            Assert.False(output.HasErrors);

            var assemblyStream = CodeGenerator.Compile(output.SourceCode, "test-assembly", references => CodeGenTestHelper.ReferenceHandler(references));
            var sourceCode     = output.SourceCode;

            Assert.Contains("class Class1", sourceCode);
            Assert.Contains("class Class2", sourceCode);
            Assert.Contains("class Class3", sourceCode);
            Assert.Contains("class Service1", sourceCode);
            Assert.DoesNotContain("class Service2", sourceCode);
        }

        [Fact]
        [Trait(TestCategory.CategoryTrait, TestCategory.NeonCodeGen)]
        public void Namespace()
        {
            // Verify that we can customize the namespace.

            var settings = new CodeGeneratorSettings()
            {
                SourceNamespace = typeof(Test_CustomOutput).Namespace,
                TargetNamespace = "Foo.Bar"
            };

            var generator = new CodeGenerator(settings);
            var output    = generator.Generate(Assembly.GetExecutingAssembly());

            Assert.False(output.HasErrors);

            var assemblyStream = CodeGenerator.Compile(output.SourceCode, "test-assembly", references => CodeGenTestHelper.ReferenceHandler(references));
            var sourceCode     = output.SourceCode;

            Assert.Contains("namespace Foo.Bar", sourceCode);
        }
    }
}
