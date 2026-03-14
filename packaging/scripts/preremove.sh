#!/bin/sh
set -e

if [ -d /run/systemd/system ]; then
    systemctl stop crabby.service || true
    systemctl disable crabby.service || true
fi
