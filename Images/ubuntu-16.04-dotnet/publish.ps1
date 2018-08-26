#------------------------------------------------------------------------------
# FILE:         publish.ps1
# CONTRIBUTOR:  Jeff Lill
# COPYRIGHT:    Copyright (c) 2016-2018 by neonFORGE, LLC.  All rights reserved.
#
# Builds all of the supported Ubuntu/.NET Core images and pushes them to Docker Hub.
#
# NOTE: You must be logged into Docker Hub.
#
# Usage: powershell -file ./publish.ps1 [-all]

param 
(
	[switch]$all = $False,
    [switch]$nopush = $False
)

#----------------------------------------------------------
# Global Includes
$image_root = "$env:NF_ROOT\\Images"
. $image_root/includes.ps1
#----------------------------------------------------------

function Build
{
	param
	(
		[parameter(Mandatory=$True, Position=1)][string] $dotnetVersion,
		[switch]$latest = $False
	)

	$registry = "nhive/ubuntu-16.04-dotnet"
	$date     = UtcDate
	$branch   = GitBranch
	$tag      = "${dotnetVersion}-${date}"

	# Build and publish the images.

	. ./build.ps1 -registry $registry -tag $tag -version $dotnetVersion
	PushImage "${registry}:$tag"

	if (IsProd)
	{
		Exec { docker tag "${registry}:$tag" "${registry}:$dotnetVersion" }
		PushImage "${registry}:$dotnetVersion"
	}

	if ($latest)
	{
		if (IsProd)
		{
			Exec { docker tag "${registry}:$tag" "${registry}:latest" }
			PushImage "${registry}:latest"
		}
		else
		{
			Exec { docker tag "${registry}:$tag" "${registry}:${branch}-latest" }
			PushImage "${registry}:${branch}-latest"
		}
	}
}

$noImagePush = $nopush

if ($all)
{
    # I'm not sure if these older .NET Core 2.0.x builds will work anymore
    # after we upgraded to 2.1.  There probably isn't a reason to rebuild
    # these again though, because neonHIVE was never released to the public
    # on .NET Core 2.1.
    #
	# Build 2.0.3
	# Build 2.0.4
    # Build 2.0.5
}

Build 2.1 -latest
