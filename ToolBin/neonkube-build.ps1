#------------------------------------------------------------------------------
# FILE:         neonkube-build.ps1
# CONTRIBUTOR:  Jeff Lill
# COPYRIGHT:    Copyright (c) 2016-2019 by neonFORGE, LLC.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Performs a clean build of the neonKUBE solution and publishes binaries
# to the [$/build] folder.  This can also optionally build the neonKUBE
# Desktop installer.
#
# USAGE: powershell -file ./build-neonkube.ps1 [OPTIONS]
#
# OPTIONS:
#
#       -debug      - Builds the DEBUG version (this is the default)
#       -release    - Builds the RELEASE version
#       -nobuild    - Don't build the solution; just publish
#       -installer  - Builds installer binaries

param 
(
	[switch]$debug     = $False,
	[switch]$release   = $False,
	[switch]$nobuild   = $False,
	[switch]$installer = $False
)

$msbuild = "C:\Program Files (x86)\Microsoft Visual Studio\2019\Preview\MSBuild\Current\Bin\amd64\MSBuild.exe"
$nfRoot  = "$env:NF_ROOT"
$nfBuild = "$env:NF_BUILD"

if (-not $debug -and -not $release)
{
    $debug = $true
}

if ($debug)
{
    $config      = "Debug"
    $buildConfig = "-p:Configuration=Debug"
}
else
{
    $config      = "Release"
    $buildConfig = "-p:Configuration=Release"
}

function PublishCore
{
    param (
        [Parameter(Position=0, Mandatory=1)]
        [string]$projectPath,           # The relative project folder PATH

        [Parameter(Position=1, Mandatory=1)]
        [string]$targetName             # Name of the target executable
    )

    # Ensure that the NF_BUILD folder exists:

    [System.IO.Directory]::CreateDirectory($nfBuild)

    # Build the [pubcore] arguments:

    $projectPath = [System.IO.Path]::Combine($nfRoot, $projectPath)
    $targetPath  = [System.IO.Path]::Combine($nfRoot, [System.IO.Path]::GetDirectoryName($projectPath), "bin", $config, "netcoreapp3.0", "$targetName.dll")

    # Publish the binaries.

    ""
    "**********************************************************"
    "*** PUBLISH: $targetName"
    "**********************************************************"
    ""
    "pubcore ""$projectPath"" ""$targetName"" ""$config"" ""$targetPath"" ""$nfBuild"" win10-x64"
    ""

    pubcore "$projectPath" "$targetName" "$config" "$targetPath" "$nfBuild" win10-x64

    if (-not $?)
    {
        ""
        "*** PUBLICATION FAILED ***"
        ""
        exit 1
    }
}

$originalDir = $pwd
cd $nfRoot

if (-not $nobuild)
{
    # Clear the NF_BUILD folder:

    & rm -r "$nfBuild/*"

    # Build the solution.

    ""
    "**********************************************************"
    "***                   CLEAN SOLUTION                   ***"
    "**********************************************************"
    ""

    & "$msbuild" $buildConfig "-t:Clean"

    if (-not $?)
    {
        ""
        "*** CLEAN FAILED ***"
        ""
        exit 1
    }

    ""
    "**********************************************************"
    "***                  RESTORE PACKAGES                  ***"
    "**********************************************************"
    ""

    & "$msbuild" /restore

    if (-not $?)
    {
        ""
        "*** RESTORE FAILED ***"
        ""
        exit 1
    }

    ""
    "**********************************************************"
    "***                   BUILD SOLUTION                   ***"
    "**********************************************************"
    ""

    & "$msbuild" $buildConfig "-t:Compile"

    if (-not $?)
    {
        ""
        "*** BUILD FAILED ***"
        ""
        exit 1
    }
}

# Publish the .NET Core binaries.

PublishCore "Tools\neon-cli\neon-cli.csproj"         "neon"
PublishCore "Tools\neon-install\neon-install.csproj" "neon-install"
PublishCore "Tools\nshell\nshell.csproj"             "nshell"

# $todo(jeff.lill):
#
# These two apps don't publish for some reason so I'll comment
# them out for now (we don't really need them anyway).

# PublishCore "Tools\entity-gen\entity-gen.csproj"     "entity-gen"
# PublishCore "Tools\text\text.csproj"                 "text"

if ($installer)
{
    ""
    "**********************************************************"
    "***            WINDOWS DESKTOP INSTALLER               ***"
    "**********************************************************"
    ""

    & neon-install build-installer windows

    if (-not $?)
    {
        ""
        "*** Windows Installer: Build failed ***"
        ""
        exit 1
    }

    "Generating windows installer sha512..."
    & cat "$nfBuild\neonKUBE-setup.exe" | openssl dgst -sha512 > "$nfBuild\neonKUBE-setup.exe.sha512"

    if (-not $?)
    {
        ""
        "*** Windows Installer: Build SHA512 failed ***"
        ""
        exit 1
    }
}

cd $originalDir
