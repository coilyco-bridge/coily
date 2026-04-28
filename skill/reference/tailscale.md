# coily tailscale - full reference

Mirrors `tailscale`. Underlying version at scan time: 1.96.3

Command shape: `coily tailscale <verb...> [flags]`. Flags match the underlying CLI.

## `coily tailscale cert`

### `coily tailscale cert`

Get TLS certs

Flags:

- `--cert-file`
- `--key-file`
- `--min-validity`

## `coily tailscale down`

### `coily tailscale down`

Disconnect from Tailscale

Flags:

- `--accept-risk`
- `--reason`

## `coily tailscale funnel`

### `coily tailscale funnel` (group)

Serve content and local servers on the internet

Subcommands: `status`, `reset`

### `coily tailscale funnel status`

View current funnel configuration

Flags:

- `--json`

### `coily tailscale funnel reset`

Reset current funnel config

## `coily tailscale ip`

### `coily tailscale ip`

Show Tailscale IP addresses

Flags:

- `--assert`

## `coily tailscale login`

### `coily tailscale login`

Log in to a Tailscale account

Flags:

- `--advertise-routes`
- `--advertise-tags`
- `--audience`
- `--auth-key`
- `--client-id`
- `--client-secret`
- `--exit-node`
- `--hostname`
- `--id-token`
- `--login-server`
- `--nickname`
- `--qr-format`
- `--timeout`

## `coily tailscale logout`

### `coily tailscale logout`

Disconnect from Tailscale and expire current node key

Flags:

- `--reason`

## `coily tailscale netcheck`

### `coily tailscale netcheck`

Print an analysis of local network conditions

Flags:

- `--bind-address`
- `--bind-port`
- `--every`
- `--format`

## `coily tailscale ping`

### `coily tailscale ping`

Ping a host at the Tailscale layer, see how it routed

Flags:

- `--c`
- `--size`
- `--timeout`

## `coily tailscale serve`

### `coily tailscale serve` (group)

Serve content and local servers on your tailnet

Subcommands: `status`, `reset`, `drain`, `clear`, `advertise`, `get-config`, `set-config`

### `coily tailscale serve status`

View current serve configuration

Flags:

- `--json`

### `coily tailscale serve reset`

Reset current serve config

### `coily tailscale serve drain`

Drain a service from the current node

### `coily tailscale serve clear`

Remove all config for a service

### `coily tailscale serve advertise`

Advertise this node as a service proxy to the tailnet

### `coily tailscale serve get-config`

Get service configuration to save to a file

Flags:

- `--service`

### `coily tailscale serve set-config`

Define service configuration from a file

Flags:

- `--service`

## `coily tailscale set`

### `coily tailscale set`

Change specified preferences

Flags:

- `--accept-risk`
- `--advertise-routes`
- `--exit-node`
- `--hostname`
- `--nickname`
- `--relay-server-port`
- `--relay-server-static-endpoints`

## `coily tailscale ssh`

### `coily tailscale ssh`

SSH to a Tailscale machine

## `coily tailscale status`

### `coily tailscale status`

Show state of tailscaled and its connections

Flags:

- `--listen`

## `coily tailscale switch`

### `coily tailscale switch` (group)

Switch to a different Tailscale account

Subcommands: `remove`

### `coily tailscale switch remove`

Remove a Tailscale account

## `coily tailscale up`

### `coily tailscale up`

Connect to Tailscale, logging in if needed

Flags:

- `--accept-risk`
- `--advertise-routes`
- `--advertise-tags`
- `--audience`
- `--auth-key`
- `--client-id`
- `--client-secret`
- `--exit-node`
- `--hostname`
- `--id-token`
- `--login-server`
- `--qr-format`
- `--timeout`

## `coily tailscale update`

### `coily tailscale update`

Update Tailscale to the latest/different version

Flags:

- `--track`
- `--version`

## `coily tailscale whois`

### `coily tailscale whois`

Show the machine and user associated with a Tailscale IP (v4 or v6)

Flags:

- `--proto`
