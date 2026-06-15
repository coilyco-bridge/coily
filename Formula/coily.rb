class Coily < Formula
  desc "Operator CLI for Kai's homelab - audited wrapper over aws, kubectl, gh, and friends"
  homepage "https://forgejo.coilysiren.me/coilysiren/coily"
  url "https://forgejo.coilysiren.me/coilyco-bridge/coily.git", tag: "v2.55.0", revision: "f74f2fb2dd1010570764fc7be47dee392889079e"
  license "MIT"
  head "https://forgejo.coilysiren.me/coilysiren/coily.git", branch: "main"

  depends_on "go" => :build

  def install
    # config.yaml is no longer //go:embed'd: coily/2b27fd3 dropped pkg/config
    # and switched the defaults to Go literals in cli-guard/config. Nothing
    # in the source tree needs staging before the go build runs.

    ldflags = "-s -w -X main.Version=v#{version}"
    system "go", "build", "-tags", "prod", "-trimpath",
           "-ldflags", ldflags,
           "-o", bin/"coily",
           "./cmd/coily"

    # Stage every coily-* skill directory into a brew-managed prefix so
    # `coily setup`'s symlink loop can wire each one into ~/.claude/skills/.
    # Keeps skill freshness tied to brew upgrades instead of requiring a
    # coily checkout. Source binary `coily install-skill` stays dev-only on
    # purpose - the threat model is about runtime self-steering, not the
    # brew install path.
    skills_src = Dir[".agents/skills/coily-*"]
    odie "no coily-* skills found under .agents/skills/" if skills_src.empty?
    (pkgshare/"skills").install skills_src
  end

  # Regenerate ~/.coily/dashboard.html every 5 minutes by shelling out to
  # `coily audit dashboard --since 7d`. The Mac launchd plist that used to
  # carry this lived at infrastructure/scripts/launchd/me.coilysiren.coily-
  # audit-dashboard.plist and is being retired now that the brew formula is
  # the install path. The Linux side stays on systemd (different output
  # path, /var/lib/coily/dashboard.html, for Caddy serving).
  service do
    run [opt_bin/"coily", "audit", "dashboard", "--since", "7d"]
    run_type :interval
    interval 300
    log_path var/"log/coily-audit-dashboard.log"
    error_log_path var/"log/coily-audit-dashboard.log"
  end

  # Brew's post_install sandbox blocks writes outside the cellar, so the
  # home-dir-touching steps (completion, skill symlink, host-bootstrap) are
  # wrapped behind `coily setup` and printed as a single caveat after each
  # install/upgrade. Lockdown is deliberately NOT a setup step: fleet-wide
  # lockdown convergence is an ansible rollout (infrastructure/ansible), per
  # the authoring-vs-rollout rule. Brew installs the binary and stops.
  def caveats
    <<~EOS
      Run once per upgrade to refresh completion and skill symlinks:

        coily setup

      Lockdown is converged by ansible (`coily ansible-freshen`) on the fleet,
      not by `coily setup`. For a one-off, run `coily lockdown` by hand.
    EOS
  end

  test do
    assert_match "v#{version}", shell_output("#{bin}/coily version")
    staged = Dir[pkgshare/"skills/coily-*"]
    assert !staged.empty?, "expected coily-* skills staged under #{pkgshare}/skills"
  end
end
