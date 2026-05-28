# FilePilot

FilePilot is a cross-platform file transfer orchestration tool for developer machines, lab servers, edge devices, and agent workflows. This context defines the product language used when discussing requirements and implementation boundaries.

## Language

**FilePilot**:
A tool that orchestrates file transfer sessions across two endpoints without requiring direct IP reachability between them.
_Avoid_: croc wrapper, file transfer protocol, cloud drive

**Transfer Session**:
A short-lived pairing between one sender and one receiver for transferring one file payload or one packaged directory payload.
_Avoid_: job, upload, sync task

**Transfer Attempt**:
One local invocation of send or receive that ends in success, failure, or cancellation and may be recorded in transfer history.
_Avoid_: successful transfer only, audit record

**File Payload**:
A single file selected by the sender and delivered to the receiver as that file, even if its extension looks like an archive.
_Avoid_: directory package

**Directory Package**:
A FilePilot-created archive representing a source directory, eligible for automatic unpacking on receive.
_Avoid_: arbitrary archive, compressed file

**Pack Command**:
The public auxiliary command that creates a Directory Package without starting a Transfer Session.
_Avoid_: required send step, manual precondition

**Clean Command**:
The maintenance command that removes FilePilot-owned temporary files from FilePilot cache locations.
_Avoid_: download cleanup, project cleanup, history deletion

**FilePilot Session ID**:
The user-visible code for a Transfer Session, used by the receiver to join without seeing backend commands or backend-specific terminology. In the Public MVP, it is also the controlled passphrase supplied to the Transfer Backend.
_Avoid_: code phrase, password, backend key

**Sensitive Session Code**:
A FilePilot Session ID during its valid lifetime, because anyone who obtains it may be able to join the corresponding Transfer Session.
_Avoid_: public ID, harmless label

**Local Session Timeout**:
A sender-side timeout that cancels the running Transfer Session process when the receiver has not completed in time.
_Avoid_: server-side expiry, remote revocation

**Backend Credential**:
A short-lived secret or command token produced by a Transfer Backend and consumed internally by FilePilot to connect the sender and receiver.
_Avoid_: session ID, user password, public token

**Transfer Backend**:
The replaceable lower-level mechanism that performs the actual byte transfer after FilePilot has prepared and paired a Transfer Session.
_Avoid_: FilePilot core, protocol, session service

**CrocBackend**:
The only actual Transfer Backend required for the Public MVP, wrapping croc behind FilePilot's stable CLI, logs, status model, and session language.
_Avoid_: default product identity, only possible backend

**Bundled Backend**:
A supported backend-compatible executable shipped with the FilePilot installation and invoked internally by FilePilot.
_Avoid_: user-installed dependency, runtime download, system package

**Rendezvous Control Layer**:
A lightweight coordination service that stores short-lived Transfer Session metadata and Backend Credentials, but never stores file contents.
_Avoid_: relay, file server, cloud storage

**Backend Rendezvous**:
The rendezvous capability supplied by a Transfer Backend, used by FilePilot's default path to pair endpoints without operating a FilePilot-hosted coordination service.
_Avoid_: FilePilot service, official hosted service

**Relay**:
A network service used by a Transfer Backend to move file data when endpoints cannot connect directly.
_Avoid_: rendezvous service, session service

**Public MVP**:
The first externally demonstrable version of FilePilot, where users interact only with FilePilot Session IDs and do not need to copy Backend Credentials.
_Avoid_: Phase 1 prototype

**Internal Validation Stage**:
An engineering-only milestone used to validate packaging, diagnostics, logging, and backend invocation before the Public MVP boundary is reached.
_Avoid_: MVP, release

**Blocking Send**:
The Public MVP send behavior where the sender process stays alive until the receiver joins and the transfer completes or fails.
_Avoid_: background job, upload ticket, fire-and-forget send

**Transfer State**:
A stable FilePilot-level phase reported during send or receive, such as preparing, packing, waiting for receiver, transferring, completed, or failed.
_Avoid_: backend progress line, exact percentage

**Agent API**:
The stable structured CLI contract exposed through JSON output so an automation agent can make decisions without parsing human text.
_Avoid_: pretty JSON, debug output, human output wrapper

**Interactive Receive**:
The human-only receive flow where the CLI prompts for a FilePilot Session ID when none was provided.
_Avoid_: session discovery, automatic nearby session list

**Local Diagnostics**:
Checks that describe whether the current machine is ready to run FilePilot and its configured Transfer Backend.
_Avoid_: end-to-end guarantee, network certification

**Advanced Self-Hosted Services**:
Optional user-managed rendezvous or relay services configured for controlled environments, not required for the default MVP experience.
_Avoid_: default path, required deployment

## Flagged Ambiguities

**MVP vs. Phase 1 prototype**:
Resolved: the Public MVP must include FilePilot Session IDs backed by the selected backend's rendezvous capability, without exposing raw backend receive commands. A stage that exposes backend receive commands is only an Internal Validation Stage.

**MVP rendezvous ownership**:
Resolved: the Public MVP does not provide an official hosted FilePilot rendezvous service and does not require users to deploy a separate service. The default MVP path reuses Backend Rendezvous and Relay capabilities from the selected Transfer Backend.

**MVP backend scope**:
Resolved: the Public MVP implements only CrocBackend as an actual backend while preserving a Transfer Backend abstraction for later replacement.

**MVP backend availability**:
Resolved: FilePilot should be usable without asking ordinary users to manually install croc. The Public MVP uses a Bundled Backend by default, allows an explicit configured backend path, may optionally fall back to PATH, and fails with `BACKEND_NOT_FOUND` if no usable backend is available.

**Runtime backend installation**:
Resolved: the Public MVP does not download backend binaries at runtime, modify PATH, install system packages, or call platform package managers.

**MVP send lifecycle**:
Resolved: `filepilot send <path>` uses Blocking Send by default. Background sending, daemonized transfer management, and detached sessions are later capabilities, not MVP requirements.

**MVP unpacking rule**:
Resolved: automatic unpacking applies only to Directory Packages created by FilePilot. File Payloads are saved as-is, including user-supplied archive files.

**MVP pack command**:
Resolved: `filepilot pack <dir>` is part of the Public MVP as an auxiliary command. `filepilot send <dir>` still packs automatically, and generated Directory Packages default to FilePilot's cache rather than the source directory.

**MVP clean command**:
Resolved: `filepilot clean` only removes FilePilot-owned temporary files under FilePilot cache locations by default. It does not delete downloads, source files, arbitrary archives, or transfer history.

**Public MVP command boundary**:
Resolved: the Public MVP includes `send`, `receive`, `pack`, `doctor`, `clean`, `config show`, and `config set`. It excludes `history`, `serve`, `daemon`, `gui`, `sync`, `resume`, and `login`.

**MVP progress reporting**:
Resolved: the Public MVP requires stable Transfer States and final structured results, not precise percentages or real-time transfer speeds parsed from backend output.

**MVP JSON contract**:
Resolved: `--json` is the Agent API and must use stable fields and error codes. Human-readable output may change more freely.

**MVP security boundary**:
Resolved: FilePilot does not implement an independent encryption, authentication, account, or device-pairing security model in the Public MVP. Transfer security comes from the configured Transfer Backend; FilePilot is responsible for safe session code generation, user guidance, log redaction, optional expiry handling, and hiding backend-specific commands and terminology.

**Session code visibility and logging**:
Resolved: the FilePilot Session ID is intentionally visible to the sender and shareable with the receiver through a trusted channel, but it is still a Sensitive Session Code. Persistent logs, crash reports, and transfer history must not record it in full by default.

**MVP transfer history**:
Resolved: FilePilot records each send or receive Transfer Attempt, including failures and cancellations, while excluding full session codes, backend raw commands, and backend credentials.

**MVP session expiry**:
Resolved: because the default Public MVP path does not use a FilePilot-hosted rendezvous service, FilePilot does not promise server-side expiry or remote revocation. It may provide Local Session Timeout and cancellation, while backend-level validity is controlled by the Transfer Backend.

**MVP receive interaction**:
Resolved: `filepilot receive` may prompt a human for a FilePilot Session ID, but it does not discover or list available sessions. Agent API mode must not prompt; `filepilot receive --json` without a session ID returns a structured error.

**Receive command naming**:
Resolved: the canonical receive command is `filepilot receive`. Short aliases such as `recv` are not part of the Public MVP requirement.

**MVP doctor boundary**:
Resolved: `filepilot doctor` performs Local Diagnostics only. Warnings such as proxy variables or suspected Fake-IP do not cause a non-zero exit code; fatal local blockers such as a missing backend or unwritable required directory do.

**Rendezvous Control Layer vs. Backend Rendezvous vs. Relay**:
Resolved: a FilePilot Rendezvous Control Layer is optional future or advanced infrastructure; Backend Rendezvous is the default MVP pairing mechanism supplied by the backend; the Relay carries transfer data for the Transfer Backend.

**FilePilot Session ID vs. Backend Credential without FilePilot-hosted rendezvous**:
Resolved: in the Public MVP, the FilePilot Session ID is a FilePilot-generated controlled passphrase that is supplied to the Transfer Backend. FilePilot hides backend commands and terminology, not the existence of a user-visible join code.

## Example Dialogue

Developer: "Can the MVP just print the backend receive command?"

Domain expert: "No. That is only acceptable in the Internal Validation Stage. The Public MVP must give the receiver a FilePilot Session ID."

Developer: "Is the FilePilot Session ID separate from the backend passphrase?"

Domain expert: "Not in the Public MVP. FilePilot generates the visible session code and supplies it to the backend as the passphrase, while keeping backend commands and backend-specific language out of the user workflow."

Developer: "Can we store the full FilePilot Session ID in transfer history?"

Domain expert: "No. The user can see and share it during the transfer, but persistent records should redact it because it is a sensitive one-time join code while valid."

Developer: "Does the Rendezvous Control Layer store the uploaded file?"

Domain expert: "No. In the default MVP path FilePilot does not operate that layer at all; it reuses the Backend Rendezvous and Relay. If a FilePilot-controlled layer is configured later, it still must not store file contents."
