class Coily < Formula
  desc "Operator CLI for Kai's homelab - audited wrapper over aws, kubectl, gh, and friends"
  homepage "https://github.com/coilysiren/coily"
  url "ssh://git@github.com/coilysiren/coily.git", tag: "v1.15.2", revision: "26c6f8ea9f95d7c8e5ebd80e62bdc60e22072840"
  license "MIT"
  head "https://github.com/coilysiren/coily.git", branch: "main"

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
    skills_src = Dir[".claude/skills/coily-*"]
    odie "no coily-* skills found under .claude/skills/" if skills_src.empty?
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
  # three home-dir-touching steps (completion, skill symlink, lockdown
  # re-baseline) are wrapped behind `coily setup` and printed as a single
  # caveat after each install/upgrade.
  def caveats
    <<~EOS
      Run once per upgrade to refresh completion, skill symlink, and lockdown:

        coily setup
    EOS
  end

  test do
    assert_match "v#{version}", shell_output("#{bin}/coily version")
    staged = Dir[pkgshare/"skills/coily-*"]
    assert !staged.empty?, "expected coily-* skills staged under #{pkgshare}/skills"
  end
end
