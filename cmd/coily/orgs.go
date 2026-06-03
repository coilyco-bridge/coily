package main

// orgs.go centralizes the fleet's primary-org set. Pre org-split everything
// hardcoded "coilysiren"; the fleet now spans coilysiren, coilyco-bridge, and
// coilyco-flight-deck (coilyco-bridge/coily#162). Every "is this one of ours"
// check (dispatch trust, brew/scoop tap scope) reads the configured set
// through here instead of a literal, so adding the next org is a config edit.

// defaultPrimaryOrgs is the built-in primary-org set, used when config does
// not override primary_orgs. Order is historical-first so existing on-disk
// layouts resolve before newer ones.
func defaultPrimaryOrgs() []string {
	return []string{"coilysiren", "coilyco-bridge", "coilyco-flight-deck"}
}

// primaryOrgs returns the configured primary-org set, falling back to the
// built-in default for configs that predate the primary_orgs field.
func (r *Runner) primaryOrgs() []string {
	if len(r.Cfg.PrimaryOrgs) > 0 {
		return r.Cfg.PrimaryOrgs
	}
	return defaultPrimaryOrgs()
}

// isPrimaryOrg reports whether owner is in the primary-org set.
func isPrimaryOrg(orgs []string, owner string) bool {
	for _, o := range orgs {
		if o == owner {
			return true
		}
	}
	return false
}
