#!/bin/bash

# Add the icon as a resource to the executable and add the manifest
rsrc -ico assets/dock-window.ico -manifest app.manifest -o rsrc.syso

# Build for Windows without console
go build -ldflags="-H windowsgui" -o telewindow.exe