package main

import "github.com/urfave/cli/v3"

// claudeRemoteControlCommand wraps the `claude-remote-control` systemd unit
// on kai-server. The unit runs the Claude Code remote-control daemon
// (outbound HTTPS to Anthropic only, no listening port) so Kai can drive a
// kai-server-hosted Claude session from claude.ai/code.
//
// Lives under `coily ops` rather than `coily gaming` because it is not a
// game server. Uses the same `systemdUnitCommand` pattern as icarus:
// daemon-reload before restart (the unit gets edited when capacity/spawn
// flags change), and enable/disable on start/stop.
func (r *Runner) claudeRemoteControlCommand() *cli.Command {
	return r.systemdUnitCommand(systemdUnit{
		VerbName:            "claude-remote-control",
		UnitName:            "claude-remote-control",
		RestartDaemonReload: true,
		StartEnables:        true,
		StopDisables:        true,
	})
}
