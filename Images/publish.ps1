﻿#------------------------------------------------------------------------------
# FILE:         publish.ps1
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

# Builds and publishes all of the Neon Docker images.
#
# NOTE: You must be logged into Docker Hub.
#
# Usage: powershell -file ./publish-all.ps1 [-all]

param 
(
	[switch]$all         = $false,        # Rebuild all images
	[switch]$base        = $false,        # Rebuild base images
	[switch]$dotnetBase  = $false,        # Rebuild base .NET images
	[switch]$dotnet      = $false,        # Rebuild .NET based images
	[switch]$other       = $false,        # Rebuild all other images (usually script based)
    [switch]$nopush      = $false,        # Don't push to the registry
    [switch]$noprune     = $false,        # Don't prune the local Docker state
	[switch]$allVersions = $false         # Rebuild all image versions
)

#----------------------------------------------------------
# Global includes
$image_root = "$env:NF_ROOT\\Images"
. $image_root/includes.ps1
#----------------------------------------------------------

# Take care to ensure that you order the image builds such that
# dependant images are built before any dependancies.

function Publish
{
    [CmdletBinding()]
    param (
        [Parameter(Position=0, Mandatory=1)]
        [string]$Path
    )

	cd "$Path"

	if ($allVersions)
	{
		if ($nopush)
		{
			./publish.ps1 -all -nopush
		}
		else
		{
			./publish.ps1 -all
		}
	}
	else
	{
		if ($nopush)
		{
			./publish.ps1 -nopush
		}
		else
		{
			./publish.ps1
		}
	}
}

# Handle the command line arguments.

if ($all)
{
	$base       = $true
	$dotnetBase = $true
	$dotnet     = $true
	$other      = $true
}
elseif ((-not $base) -and (-not $dotnet) -and (-not $other))
{
	# Build .NET and other images, but not base images, 
	# by default.

	$dotnet = $true
	$other  = $true
}

# Purge any local Docker images as well as the image build cache.
# This also purges everything else Docker as a side effect.  We
# need to do this to ensure that we get a clean build.

if (-not $noprune)
{
	docker system prune -af
}

# NOTE: 
#
# The build order below is important since later images
# may depend on earlier ones.

if ($base)
{
	$dotnetBase = $true

	# Base OS images:

	Publish "$image_root\\alpine"
	Publish "$image_root\\ubuntu-16.04"

	# Other base images:

	Publish "$image_root\\haproxy"
}

if ($dotnetBase)
{
	Publish "$image_root\\dotnet"
	Publish "$image_root\\aspnet"
	Publish "$image_root\\ubuntu-16.04-dotnet"
	Publish "$image_root\\ubuntu-16.04-aspnet"
}

if ($other)
{
	Publish "$image_root\\golang"
    Publish "$image_root\\nats"
    Publish "$image_root\\nats-streaming"
	Publish "$image_root\\cadence-test"
	Publish "$image_root\\couchbase-test"
	Publish "$image_root\\test"
}
