#!/usr/bin/env python3

import base64
import json
import os
import select
import socket
import sys
import time

QEMU_GA_SOCKET = os.environ.get("QEMU_GA_SOCKET", "/tmp/qemu-virtserialport.sock")
READY_TIMEOUT = int(os.environ.get("QEMU_GA_READY_TIMEOUT", "120"))
COMMAND_TIMEOUT = int(os.environ.get("QEMU_GA_COMMAND_TIMEOUT", "30"))


def send_qemu_command(command, socket_path=QEMU_GA_SOCKET, timeout=5):
    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
        client.settimeout(timeout)
        client.connect(socket_path)
        client.sendall((json.dumps(command) + "\n").encode("utf-8"))
        response = b""
        deadline = time.monotonic() + timeout
        while b"\n" not in response:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise TimeoutError(f"No guest-agent response within {timeout} seconds")
            ready, _, _ = select.select([client], [], [], remaining)
            if not ready:
                raise TimeoutError(f"No guest-agent response within {timeout} seconds")
            chunk = client.recv(4096)
            if not chunk:
                break
            response += chunk
    if not response:
        raise RuntimeError("guest agent returned an empty response")
    payload = json.loads(response.splitlines()[0])
    if "error" in payload:
        desc = payload["error"].get("desc", payload["error"])
        raise RuntimeError(f"guest agent error: {desc}")
    return payload.get("return")


def wait_guest_agent(timeout=READY_TIMEOUT):
    deadline = time.monotonic() + timeout
    last_error = None
    while time.monotonic() < deadline:
        try:
            send_qemu_command({"execute": "guest-ping"})
            return
        except (OSError, TimeoutError, ValueError, RuntimeError) as exc:
            last_error = exc
            time.sleep(2)
    raise RuntimeError(f"guest agent did not become ready within {timeout} seconds: {last_error}")


def run_guest_exec(path, args, timeout=COMMAND_TIMEOUT):
    result = send_qemu_command({
        "execute": "guest-exec",
        "arguments": {"path": path, "arg": args, "capture-output": True},
    })
    pid = (result or {}).get("pid")
    if pid is None:
        raise RuntimeError(f"guest-exec returned no pid: {result}")

    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        status = send_qemu_command({"execute": "guest-exec-status", "arguments": {"pid": pid}}) or {}
        if status.get("exited"):
            stdout = base64.b64decode(status.get("out-data", "")).decode("utf-8", errors="replace")
            stderr = base64.b64decode(status.get("err-data", "")).decode("utf-8", errors="replace")
            if status.get("exitcode", 0) != 0:
                raise RuntimeError(f"{path} failed with exit code {status.get('exitcode')}: {stderr.strip()}")
            return stdout
        time.sleep(1)
    raise RuntimeError(f"guest-exec pid {pid} did not finish within {timeout} seconds")


def parse_credentials(output):
    credentials = {}
    for line in output.splitlines():
        key, separator, value = line.partition("=")
        if separator:
            credentials[key.strip()] = value.strip()
    api_key = credentials.get("key", "")
    api_secret = credentials.get("secret", "")
    if not api_key or not api_secret or "\n" in api_key or "\n" in api_secret:
        raise RuntimeError("opn-apikey returned invalid credentials")
    return api_key, api_secret


def write_outputs(api_key, api_secret):
    print(f"::add-mask::{api_key}")
    print(f"::add-mask::{api_secret}")
    output_path = os.environ.get("GITHUB_OUTPUT")
    if not output_path:
        raise RuntimeError("GITHUB_OUTPUT is not set")
    with open(output_path, "a", encoding="utf-8") as output_file:
        output_file.write(f"key={api_key}\n")
        output_file.write(f"secret={api_secret}\n")


def main():
    wait_guest_agent()
    stdout = run_guest_exec("/usr/local/bin/opn-apikey", ["-u", "root", "create"])
    api_key, api_secret = parse_credentials(stdout)
    write_outputs(api_key, api_secret)
    print("OPNsense API credentials created successfully")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(1)
