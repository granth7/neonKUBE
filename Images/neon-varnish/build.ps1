﻿#------------------------------------------------------------------------------
# FILE:         build.ps1
# CONTRIBUTOR:  Jeff Lill
# COPYRIGHT:    Copyright (c) 2016-2018 by neonFORGE, LLC.  All rights reserved.
#
# Builds the neonHIVE Varnish base images.

param 
(
	[parameter(Mandatory=$True,Position=1)][string] $registry,
	[parameter(Mandatory=$True,Position=2)][string] $varnishVersion,    # Specific Varish version (like "6.0.0")
	[parameter(Mandatory=$True,Position=3)][string] $tag
)

"   "
"======================================="
"* NEON-VARNISH:" + $tag
"======================================="

$appname = "neon-varnish"

# Build and publish the app to a local [bin] folder.

if (Test-Path bin)
{
	rm -r bin
}

Exec { mkdir bin }
Exec { dotnet publish "$src_services_path\\$appname\\$appname.csproj" -c Release -o "$pwd\bin" }

# Split the build binaries into [__app] application and [__dep] dependency subfolders
# so we can tune the image layers.

Exec { core-layers $appname "$pwd\bin" }

# Build the image.

Exec { docker build -t "${registry}:$tag" --build-arg "FAMILY=$varnishFamily" --build-arg "BRANCH=$branch" --build-arg "VERSION=$varnishVersion" --build-arg "APPNAME=$appname" . }

# Clean up

Exec { rm -r bin }