#!/usr/bin/env python3
"""
apply_patch.py

Usage:
  python apply_patch.py feature-admin-security-export.patch

This script parses a patch file in the custom format produced earlier:
- File blocks start with a line like: *** Add File: path/to/file
- File contents lines are prefixed with '+'
- File blocks end with: *** End Patch

The script writes each file to disk (creating directories as needed) and
preserves content. For shell scripts it will also set executable permission
on non-Windows platforms when a shebang is present.
"""
import sys
import os
import stat

def write_file(path, lines):
    # Ensure directory exists
    d = os.path.dirname(path)
    if d and not os.path.exists(d):
        os.makedirs(d, exist_ok=True)
    content = "".join(lines)
    with open(path, "w", newline="\n", encoding="utf-8") as f:
        f.write(content)
    # If file appears to be a script with shebang, mark executable on POSIX
    if os.name != "nt":
        if len(lines) > 0 and lines[0].startswith("#!"):
            st = os.stat(path)
            os.chmod(path, st.st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)

def parse_patch(patch_path):
    if not os.path.exists(patch_path):
        print("Patch file not found:", patch_path)
        sys.exit(1)

    current_file = None
    current_lines = []
    in_file = False

    with open(patch_path, "r", encoding="utf-8", errors="replace") as f:
        for raw in f:
            line = raw.rstrip("\n")
            # Detect Add File marker
            if line.startswith("*** Add File:"):
                # If we were writing a previous file, flush it
                if in_file and current_file:
                    write_file(current_file, current_lines)
                    print("Wrote:", current_file)
                # start new file
                path = line.split(":", 1)[1].strip()
                # normalize path separators for current OS
                path = path.replace("/", os.sep)
                current_file = path
                current_lines = []
                in_file = True
                continue
            # Detect End Patch marker - end of current file block
            if line.startswith("*** End Patch"):
                if in_file and current_file:
                    write_file(current_file, current_lines)
                    print("Wrote:", current_file)
                current_file = None
                current_lines = []
                in_file = False
                continue
            # Inside a file block: content lines usually start with '+'
            if in_file and current_file:
                # If line starts with '+' (as in the produced patch), strip it
                if line.startswith("+"):
                    content_line = line[1:] + "\n"
                else:
                    # if it's an empty line or doesn't have a '+' prefix, include it as-is
                    content_line = line + "\n"
                current_lines.append(content_line)
            # else ignore other lines
    # At EOF ensure last file flushed
    if in_file and current_file:
        write_file(current_file, current_lines)
        print("Wrote:", current_file)

def main():
    if len(sys.argv) < 2:
        print("Usage: python apply_patch.py <patch-file>")
        sys.exit(1)
    patch_path = sys.argv[1]
    parse_patch(patch_path)
    print("\nAll files extracted. Next steps:")
    print("  git add .")
    print('  git commit -m "feat: admin onboarding, secure sessions, export media + S3"')
    print("  git push origin feature/admin-security-export")
    print("\nThen open a Pull Request on GitHub or use 'gh pr create' to create the PR.")

if __name__ == "__main__":
    main()