# 5. kubectl context "help" is real

Category: Known bugs and rough edges

`kubectl config current-context` on Kai's laptop returns the word "help"
because there is a context literally named "help" in his kubeconfig.
Probably a leftover from `kubectl config use-context help` somewhere.
Worth cleaning up with `kubectl config delete-context help` once Kai is
sure it is unused.

# Decision

Yes clean up.
