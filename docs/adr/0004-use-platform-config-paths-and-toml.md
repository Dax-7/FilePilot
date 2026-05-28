# Use Platform Config Paths and TOML

FilePilot's Public MVP will store configuration in platform-appropriate user config locations using `config.toml`, rather than using one Unix-style `~/.filepilot/config.yaml` path everywhere. This fits a cross-platform CLI distributed to Linux, macOS, and Windows users while keeping the MVP configuration shape simple enough for TOML.
