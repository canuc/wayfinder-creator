#!/usr/bin/python3
"""
Test script for debugging clawdbot onboard expect responses.
Run as the clawdbot user:
    python3 /home/clawdbot/test_onboard.py
"""
import pexpect
import sys
import os

cmd = os.path.expanduser("~/.local/bin/clawdbot onboard --install-daemon")

print(f"Running: {cmd}")
print("=" * 60)

child = pexpect.spawn("/bin/bash", ["-c", cmd], encoding="utf-8", timeout=300)
child.logfile_read = sys.stdout

responses = [
    ("Continue\\?",              "y",          "confirm continue"),
    ("Onboarding mode",          "\r",         "select QuickStart (default)"),
    ("Model/auth provider",      "\x1b[B\r",   "arrow down to Anthropic + enter"),
    ("Anthropic auth method",    "\x1b[B\r",   "arrow down to API key + enter"),
    ("Use existing ANTHROPIC_API_KEY", "y",    "confirm existing key"),
    ("Default model",            "\r",         "select default model"),
    ("Select channel",           "\x1b[A\r",   "arrow up + enter"),
    ("Enable hooks\\?",          " \r",        "space to toggle + enter"),
    ("Configure skills now\\?",  "n",          "skip skills config"),
    ("Install daemon",           "y",          "install daemon"),
    ("Start daemon",             "y",          "start daemon"),
]

for pattern, response, desc in responses:
    print(f"\n>>> Waiting for: {pattern} ({desc})")
    try:
        child.expect(pattern, timeout=120)
        print(f">>> MATCHED! Sending: {repr(response)}")
        child.send(response)
    except pexpect.TIMEOUT:
        print(f">>> TIMEOUT waiting for: {pattern}")
        print(f">>> Buffer contents: {child.before}")
        sys.exit(1)
    except pexpect.EOF:
        print(f">>> EOF before matching: {pattern}")
        print(f">>> Buffer contents: {child.before}")
        sys.exit(1)

print("\n>>> Waiting for process to finish...")
child.expect(pexpect.EOF, timeout=120)
print(f"\n>>> Exit status: {child.exitstatus}")
print("Done!")
