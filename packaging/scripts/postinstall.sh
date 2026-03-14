#!/bin/sh
set -e

if command -v systemd-sysusers >/dev/null 2>&1; then
    systemd-sysusers
else
    if ! getent group crabby >/dev/null; then
        groupadd --system crabby
    fi
    if ! getent passwd crabby >/dev/null; then
        useradd --system --gid crabby --shell /usr/sbin/nologin --home / crabby
    fi
fi

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload
    systemctl enable crabby.service
fi
