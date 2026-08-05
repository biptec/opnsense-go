#!/usr/bin/env python3

"""Install the exact os-bind candidate in the running OPNsense CI guest."""

import base64
import json
import os
import re
import select
import socket
import sys
import time

QEMU_GA_SOCKET = os.environ.get("QEMU_GA_SOCKET", "/tmp/qemu-virtserialport.sock")
COMMIT = os.environ.get("OPNSENSE_BIND_PLUGIN_COMMIT", "")
COMMIT_RE = re.compile(r"^[0-9a-f]{40}$")


def send(command, timeout=10):
    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
        client.settimeout(timeout)
        client.connect(QEMU_GA_SOCKET)
        client.sendall((json.dumps(command) + "\n").encode())
        response = b""
        while b"\n" not in response:
            ready, _, _ = select.select([client], [], [], timeout)
            if not ready:
                raise TimeoutError("QEMU guest agent response timed out")
            chunk = client.recv(65536)
            if not chunk:
                break
            response += chunk
    payload = json.loads(response.decode().strip())
    if "error" in payload:
        raise RuntimeError(payload["error"].get("desc", "guest agent error"))
    return payload.get("return", {})


def guest_exec(script):
    result = send({
        "execute": "guest-exec",
        "arguments": {
            "path": "/bin/sh",
            "arg": ["-c", script],
            "capture-output": True,
        },
    })
    pid = result["pid"]
    deadline = time.time() + 900
    while time.time() < deadline:
        status = send({"execute": "guest-exec-status", "arguments": {"pid": pid}})
        if status.get("exited"):
            stdout = base64.b64decode(status.get("out-data", "")).decode(errors="replace")
            stderr = base64.b64decode(status.get("err-data", "")).decode(errors="replace")
            if status.get("exitcode", 1) != 0:
                # Build output contains no credentials, but cap it to keep CI useful.
                detail = (stderr or stdout)[-8000:]
                raise RuntimeError(f"os-bind installation failed:\n{detail}")
            return stdout
        time.sleep(2)
    raise TimeoutError("os-bind installation timed out")


def main():
    if not COMMIT_RE.fullmatch(COMMIT):
        raise RuntimeError("OPNSENSE_BIND_PLUGIN_COMMIT must be a full Git commit SHA")
    script = f"""set -eu
if [ ! -d /usr/plugins/.git ]; then
    opnsense-code plugins
fi
git -C /usr/plugins fetch --depth 1 https://github.com/biptec/opnsense-plugins.git {COMMIT}
git -C /usr/plugins checkout --detach FETCH_HEAD
make -C /usr/plugins/dns/bind upgrade
pkg info -e 'os-bind-1.35'
test -f /usr/local/opnsense/mvc/app/models/OPNsense/Bind/View.xml
"""
    guest_exec(script)
    print(f"Installed os-bind candidate {COMMIT[:12]}")


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        print(f"ERROR: {error}", file=sys.stderr)
        sys.exit(1)
