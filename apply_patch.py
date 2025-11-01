#!/usr/bin/env python3
"""
apply_patch.py

Usage:
  python apply_patch.py patch_file.txt

This script parses a patch file in the custom format:
- File blocks start with: *** Add File: path/to/file
- Content lines are prefixed with '+'
- File blocks end with: *** End Patch

The script writes each file to disk, creating directories as needed.
For shell scripts with shebangs, it sets executable permissions on Unix systems.
"""
import sys
import os
import stat
import re

def write_file(path, lines):
    """Write content to file, creating directories as needed."""
    # Normalize path separators for current OS
    path = path.replace('/', os.sep).replace('\\', os.sep)
    
    # Create directory if needed
    directory = os.path.dirname(path)
    if directory and not os.path.exists(directory):
        os.makedirs(directory, exist_ok=True)
    
    # Join lines and write content
    content = "".join(lines)
    
    try:
        with open(path, "w", newline="\n", encoding="utf-8") as f:
            f.write(content)
        
        # Set executable permission for scripts with shebang on Unix
        if os.name != "nt" and lines and lines[0].startswith("#!"):
            st = os.stat(path)
            os.chmod(path, st.st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
            print(f"✓ Wrote (executable): {path}")
        else:
            print(f"✓ Wrote: {path}")
    except Exception as e:
        print(f"✗ Error writing {path}: {e}")

def parse_patch(patch_path):
    """Parse the patch file and extract all files."""
    if not os.path.exists(patch_path):
        print(f"Error: Patch file not found: {patch_path}")
        sys.exit(1)

    current_file = None
    current_lines = []
    in_file = False
    files_written = 0

    print(f"Reading patch file: {patch_path}\n")

    with open(patch_path, "r", encoding="utf-8", errors="replace") as f:
        for line_num, raw_line in enumerate(f, 1):
            line = raw_line.rstrip("\n")
            
            # Detect "*** Add File:" marker
            if line.startswith("*** Add File:"):
                # Flush previous file if any
                if in_file and current_file:
                    write_file(current_file, current_lines)
                    files_written += 1
                
                # Extract new file path
                path = line.split(":", 1)[1].strip()
                current_file = path
                current_lines = []
                in_file = True
                continue
            
            # Detect "*** End Patch" marker
            if line.startswith("*** End Patch"):
                if in_file and current_file:
                    write_file(current_file, current_lines)
                    files_written += 1
                current_file = None
                current_lines = []
                in_file = False
                continue
            
            # Inside a file block: collect content
            if in_file and current_file:
                # Strip leading '+' if present (patch format)
                if line.startswith("+"):
                    content_line = line[1:] + "\n"
                else:
                    # Empty lines or lines without '+'
                    content_line = line + "\n"
                current_lines.append(content_line)

    # Flush last file if still open
    if in_file and current_file:
        write_file(current_file, current_lines)
        files_written += 1

    print(f"\n{'='*60}")
    print(f"✓ Successfully created {files_written} files")
    print(f"{'='*60}")

def check_missing_files():
    """Check for critical missing files and warn user."""
    critical_files = [
        "frontend/src/index.js",
        "frontend/src/index.html",
        "frontend/public/index.html",
        "backend/main.go",
        "backend/store.go",
        "backend/handlers.go",
        "backend/auth.go",
        "backend/email.go"
    ]
    
    missing = []
    for file_path in critical_files:
        if not os.path.exists(file_path):
            missing.append(file_path)
    
    if missing:
        print("\n⚠️  WARNING: Critical files missing!")
        print("="*60)
        print("The patch only contains NEW files for this feature.")
        print("You need the following base files to run the app:\n")
        for f in missing:
            print(f"   ✗ {f}")
        print("\n💡 This patch is incremental and requires:")
        print("   - Existing backend Go files (main.go, store.go, etc.)")
        print("   - Existing frontend React files (index.js, App.js, etc.)")
        print("   - Base database schema")
        return True
    return False

def print_next_steps():
    """Print next steps for the user."""
    print("\n📋 Next Steps:")
    print("   1. Review the created files")
    print("   2. Stage changes: git add .")
    print("   3. Commit: git commit -m \"feat: admin security, sessions, and media export\"")
    print("   4. Push: git push origin feature/admin-security-export")
    print("   5. Create PR: gh pr create --title \"Admin Security & Export Features\"")
    print("\n💡 Don't forget to:")
    print("   - Update .env with required variables")
    print("   - Run database migrations: make migrate")
    print("   - Test the new features locally")
    print("   - Review security settings before deploying")

def main():
    """Main entry point."""
    if len(sys.argv) < 2:
        print("Usage: python apply_patch.py <patch-file>")
        print("\nExample:")
        print("  python apply_patch.py admin-security-patch.txt")
        sys.exit(1)
    
    patch_path = sys.argv[1]
    
    # Confirm action
    print("🔧 Restaurant Site Patch Applier")
    print("="*60)
    print(f"This will create/overwrite files from: {patch_path}")
    
    response = input("\nContinue? (y/n): ").lower().strip()
    if response != 'y':
        print("Aborted.")
        sys.exit(0)
    
    print()
    parse_patch(patch_path)
    print_next_steps()

if __name__ == "__main__":
    main()