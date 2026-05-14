package main

import "github.com/urfave/cli/v3"

// personalDashboardCommand wraps the `personal-dashboard` systemd unit on
// kai-server. The unit runs the personal-dashboard daemon bound to
// 127.0.0.1:31337; `tailscale serve` proxies it onto the tailnet.
//
// Lives under `coily ops` next to claudeRemoteControlCommand. Uses the
// same systemdUnitCommand pattern: daemon-reload before restart (the unit
// gets edited when the EnvironmentFile or --addr flag changes), and
// enable/disable on start/stop so reboot persistence stays in lockstep
// with operator intent.
func (r *Runner) personalDashboardCommand() *cli.Command {
	return r.systemdUnitCommand(systemdUnit{
		VerbName:            "personal-dashboard",
		UnitName:            "personal-dashboard",
		RestartDaemonReload: true,
		StartEnables:        true,
		StopDisables:        true,
	})
}
