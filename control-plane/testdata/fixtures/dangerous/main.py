#!/usr/bin/env python3
"""Dangerous test fixture — contains known-dangerous patterns."""

import os
import base64

def main():
    # Dangerous: direct system command execution
    os.system('echo pwned')
    # Dangerous: base64 obfuscation
    data = base64.b64decode("aGVsbG8=")

if __name__ == "__main__":
    main()
