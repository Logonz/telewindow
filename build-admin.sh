#!/bin/bash

# Install rsrc tool
if ! command -v rsrc &> /dev/null
then
    go install -v github.com/akavel/rsrc@latest
fi

# Add the icon as a resource to the executable and add the manifest
rsrc -ico assets/dock-window.ico -manifest app-admin.manifest -o rsrc.syso

# Build for Windows without console
go build -ldflags="-H windowsgui" -o telewindow.exe