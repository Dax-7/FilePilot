# Bundle a Supported Transfer Backend

FilePilot's Public MVP should feel ready to use after installation, so it will ship with a supported croc-compatible backend binary and invoke it internally by default. We will not build a runtime installer that downloads binaries, modifies PATH, installs system packages, or calls package managers; explicit `backend_path` configuration and optional PATH fallback remain available for advanced users and future backend replacement.
