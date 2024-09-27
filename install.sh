#!/bin/bash

   ARCH=$(uname -m)
   if [ "$ARCH" = "x86_64" ]; then
       ARCH="amd64"
   elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
       ARCH="arm64"
   else
       echo "Unsupported architecture: $ARCH"
       exit 1
   fi

   DOWNLOAD_URL="https://github.com/Merbeleue/gostat/releases/latest/download/gostat_Linux_${ARCH}.tar.gz"

   curl -L "$DOWNLOAD_URL" | tar xz
   sudo mv gostat /usr/local/bin/

   echo "gostat has been installed to /usr/local/bin/gostat"
   echo "You can now run it by typing 'gostat' in your terminal."
