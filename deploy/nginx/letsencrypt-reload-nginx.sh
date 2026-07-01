#!/bin/sh
# Reload nginx after any certificate renewal so new certs take effect.
systemctl reload nginx 2>/dev/null || nginx -s reload 2>/dev/null || true
