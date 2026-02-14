# rai

Run multiple shell commands in parallel from a single invocation.

## Install

```bash
go install github.com/phillip-england/rai@latest
```

Or build from source:

```bash
go build -o rai .
```

## Usage

```bash
rai 'cmd1 args...' 'cmd2 args...' 'cmd3 args...'
```

Each quoted argument is a separate command run in parallel. Commands are executed via `sh -c`, so pipes, env vars, and shell features work as expected.

```bash
rai 'server --port 8080' 'client --host localhost' 'watcher -d ./src'
```

Output from each command is prefixed with the command name and color-coded for readability:

```
[server] listening on :8080
[client] connected
[watcher] watching ./src for changes
```

## Behavior

- If any command exits non-zero, all other commands are terminated and `rai` exits with the failed command's exit code.
- If `rai` receives SIGINT or SIGTERM, it forwards the signal to all child processes.
- No external dependencies — stdlib only.
