package main

// Chain of preflight gates that run against `coily ops gh` argv; first non-nil wins.
func ghPreflightGate(argv []string) error {
	for _, gate := range []func([]string) error{
		ghActionsGate,
		ghIssueCreateGate,
	} {
		if err := gate(argv); err != nil {
			return err
		}
	}
	return nil
}
