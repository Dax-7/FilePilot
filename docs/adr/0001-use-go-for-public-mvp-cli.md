# Use Go for the Public MVP CLI

FilePilot's Public MVP is a cross-platform command-line tool that must run cleanly on Linux servers and Windows/macOS receivers, manage subprocesses, package directories, and ship as a simple installable binary. We will implement the main CLI in Go rather than Python so the default user path does not depend on a local interpreter, virtual environment, or Python package installation, while preserving Python only for optional helper scripts or tests.
