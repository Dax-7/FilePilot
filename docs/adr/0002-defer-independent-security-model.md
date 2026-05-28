# Defer an Independent FilePilot Security Model

FilePilot's Public MVP will not implement its own encryption protocol, account system, device pairing model, or credential mapping service. Transfer confidentiality, integrity, and backend session authentication are delegated to the configured Transfer Backend, while FilePilot is responsible for generating usable session codes, warning users that those codes are sensitive, redacting persistent logs, and hiding backend-specific commands and terminology from the normal workflow.
