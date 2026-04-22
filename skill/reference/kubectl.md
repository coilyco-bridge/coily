# coily kubectl - full reference

Mirrors `kubectl`.

Command shape: `coily kubectl <verb...> [flags]`. Flags match the underlying CLI.

## `coily kubectl annotate`

### `coily kubectl annotate`

Update the annotations on one or more resources.

Flags:

- `--all`
- `--all-namespaces`
- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--field-selector`
- `--filename`
- `--kustomize`
- `--list`
- `--local`
- `--output`
- `--overwrite`
- `--recursive`
- `--resource-version`
- `--selector`
- `--show-managed-fields`
- `--template`

## `coily kubectl api-resources`

### `coily kubectl api-resources`

Print the supported API resources on the server.

Flags:

- `--api-group`
- `--cached`
- `--categories`
- `--namespaced`
- `--no-headers`
- `--output`
- `--sort-by`
- `--verbs`

## `coily kubectl api-versions`

### `coily kubectl api-versions`

Print the supported API versions on the server, in the form of "group/version".

## `coily kubectl apply`

### `coily kubectl apply` (group)

Apply a configuration to a resource by file name or stdin.

Subcommands: `edit-last-applied`, `set-last-applied`, `view-last-applied`

### `coily kubectl apply edit-last-applied`

Edit the latest last-applied-configuration annotations of resources from the default editor.

Flags:

- `--allow-missing-template-keys`
- `--field-manager`
- `--filename`
- `--kustomize`
- `--output`
- `--recursive`
- `--show-managed-fields`
- `--template`
- `--validate`
- `--windows-line-endings`

### `coily kubectl apply set-last-applied`

Set the latest last-applied-configuration annotations by setting it to match the contents of a file.

Flags:

- `--allow-missing-template-keys`
- `--create-annotation`
- `--dry-run`
- `--filename`
- `--output`
- `--show-managed-fields`
- `--template`

### `coily kubectl apply view-last-applied`

View the latest last-applied-configuration annotations by type/name or file.

Flags:

- `--all`
- `--filename`
- `--kustomize`
- `--output`
- `--recursive`
- `--selector`

## `coily kubectl auth`

### `coily kubectl auth` (group)

Inspect authorization

Subcommands: `can-i`, `reconcile`, `whoami`

### `coily kubectl auth can-i`

Check whether an action is allowed.

Flags:

- `--all-namespaces`
- `--list`
- `--no-headers`
- `--quiet`
- `--subresource`

### `coily kubectl auth reconcile`

Reconciles rules for RBAC role, role binding, cluster role, and cluster role binding objects.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--filename`
- `--kustomize`
- `--output`
- `--recursive`
- `--remove-extra-permissions`
- `--remove-extra-subjects`
- `--show-managed-fields`
- `--template`

### `coily kubectl auth whoami`

Experimental: Check who you are and your attributes (groups, extra).

Flags:

- `--allow-missing-template-keys`
- `--output`
- `--show-managed-fields`
- `--template`

## `coily kubectl autoscale`

### `coily kubectl autoscale`

Creates an autoscaler that automatically chooses and sets the number of pods that run in a Kubernetes cluster.

Flags:

- `--allow-missing-template-keys`
- `--cpu-percent`
- `--dry-run`
- `--field-manager`
- `--filename`
- `--kustomize`
- `--max`
- `--min`
- `--name`
- `--output`
- `--recursive`
- `--save-config`
- `--show-managed-fields`
- `--template`

## `coily kubectl certificate`

### `coily kubectl certificate` (group)

Modify certificate resources.

Subcommands: `approve`, `deny`

### `coily kubectl certificate approve`

Approve a certificate signing request.

Flags:

- `--allow-missing-template-keys`
- `--filename`
- `--force`
- `--kustomize`
- `--output`
- `--recursive`
- `--show-managed-fields`
- `--template`

### `coily kubectl certificate deny`

Deny a certificate signing request.

Flags:

- `--allow-missing-template-keys`
- `--filename`
- `--force`
- `--kustomize`
- `--output`
- `--recursive`
- `--show-managed-fields`
- `--template`

## `coily kubectl cluster-info`

### `coily kubectl cluster-info` (group)

Display addresses of the control plane and services with label kubernetes.io/cluster-service=true.

Subcommands: `dump`

### `coily kubectl cluster-info dump`

Dump cluster information out suitable for debugging and diagnosing cluster problems.

Flags:

- `--all-namespaces`
- `--allow-missing-template-keys`
- `--namespaces`
- `--output`
- `--output-directory`
- `--pod-running-timeout`
- `--show-managed-fields`
- `--template`

## `coily kubectl config`

### `coily kubectl config` (group)

Modify kubeconfig files using subcommands like "kubectl config set current-context my-context"

Subcommands: `current-context`, `delete-cluster`, `delete-context`, `delete-user`, `get-clusters`, `get-contexts`, `get-users`, `rename-context`, `set`, `set-cluster`, `set-context`, `set-credentials`, `unset`, `use-context`, `view`

### `coily kubectl config current-context`

help

### `coily kubectl config delete-cluster`

deleted cluster help from /Users/kai/.kube/config

### `coily kubectl config delete-context`

warning: this removed your active context, use "kubectl config use-context" to select a different one

### `coily kubectl config delete-user`

deleted user help from /Users/kai/.kube/config

### `coily kubectl config get-clusters`

NAME

### `coily kubectl config get-contexts`

Display one or many contexts from the kubeconfig file.

Flags:

- `--no-headers`
- `--output`

### `coily kubectl config get-users`

NAME

### `coily kubectl config rename-context`

Renames a context from the kubeconfig file.

### `coily kubectl config set`

Set an individual value in a kubeconfig file.

Flags:

- `--set-raw-bytes`

### `coily kubectl config set-cluster`

Cluster "help" set.

### `coily kubectl config set-context`

Context "help" created.

### `coily kubectl config set-credentials`

User "help" set.

### `coily kubectl config unset`

Unset an individual value in a kubeconfig file.

### `coily kubectl config use-context`

Switched to context "help".

### `coily kubectl config view`

Display merged kubeconfig settings or a specified kubeconfig file.

Flags:

- `--allow-missing-template-keys`
- `--flatten`
- `--merge`
- `--minify`
- `--output`
- `--raw`
- `--show-managed-fields`
- `--template`

## `coily kubectl cordon`

### `coily kubectl cordon`

Mark node as unschedulable.

Flags:

- `--dry-run`
- `--selector`

## `coily kubectl describe`

### `coily kubectl describe`

Show details of a specific resource or group of resources.

Flags:

- `--all-namespaces`
- `--chunk-size`
- `--filename`
- `--kustomize`
- `--recursive`
- `--selector`
- `--show-events`

## `coily kubectl diff`

### `coily kubectl diff`

Diff configurations specified by file name or stdin between the current online configuration, and the configuration as it would be if applied.

Flags:

- `--field-manager`
- `--filename`
- `--force-conflicts`
- `--kustomize`
- `--prune`
- `--prune-allowlist`
- `--recursive`
- `--selector`
- `--server-side`
- `--show-managed-fields`

## `coily kubectl drain`

### `coily kubectl drain`

Drain node in preparation for maintenance.

Flags:

- `--chunk-size`
- `--delete-emptydir-data`
- `--disable-eviction`
- `--dry-run`
- `--force`
- `--grace-period`
- `--ignore-daemonsets`
- `--pod-selector`
- `--selector`
- `--skip-wait-for-delete-timeout`
- `--timeout`

## `coily kubectl events`

### `coily kubectl events`

Display events

Flags:

- `--all-namespaces`
- `--allow-missing-template-keys`
- `--chunk-size`
- `--for`
- `--no-headers`
- `--output`
- `--show-managed-fields`
- `--template`
- `--types`
- `--watch`

## `coily kubectl label`

### `coily kubectl label`

Update the labels on a resource.

Flags:

- `--all`
- `--all-namespaces`
- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--field-selector`
- `--filename`
- `--kustomize`
- `--list`
- `--local`
- `--output`
- `--overwrite`
- `--recursive`
- `--resource-version`
- `--selector`
- `--show-managed-fields`
- `--template`

## `coily kubectl logs`

### `coily kubectl logs`

Print the logs for a container in a pod or specified resource.

Flags:

- `--all-containers`
- `--container`
- `--follow`
- `--ignore-errors`
- `--insecure-skip-tls-verify-backend`
- `--limit-bytes`
- `--max-log-requests`
- `--pod-running-timeout`
- `--prefix`
- `--previous`
- `--selector`
- `--since`
- `--since-time`
- `--tail`
- `--timestamps`

## `coily kubectl patch`

### `coily kubectl patch`

Update fields of a resource using strategic merge patch, a JSON merge patch, or a JSON patch.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--filename`
- `--kustomize`
- `--local`
- `--output`
- `--patch`
- `--patch-file`
- `--recursive`
- `--show-managed-fields`
- `--subresource`
- `--template`
- `--type`

## `coily kubectl rollout`

### `coily kubectl rollout` (group)

Manage the rollout of one or many resources.

Subcommands: `history`, `pause`, `restart`, `resume`, `status`, `undo`

### `coily kubectl rollout history`

View previous rollout revisions and configurations.

Flags:

- `--allow-missing-template-keys`
- `--filename`
- `--kustomize`
- `--output`
- `--recursive`
- `--revision`
- `--selector`
- `--show-managed-fields`
- `--template`

### `coily kubectl rollout pause`

Mark the provided resource as paused.

Flags:

- `--allow-missing-template-keys`
- `--field-manager`
- `--filename`
- `--kustomize`
- `--output`
- `--recursive`
- `--selector`
- `--show-managed-fields`
- `--template`

### `coily kubectl rollout restart`

Restart a resource.

Flags:

- `--allow-missing-template-keys`
- `--field-manager`
- `--filename`
- `--kustomize`
- `--output`
- `--recursive`
- `--selector`
- `--show-managed-fields`
- `--template`

### `coily kubectl rollout resume`

Resume a paused resource.

Flags:

- `--allow-missing-template-keys`
- `--field-manager`
- `--filename`
- `--kustomize`
- `--output`
- `--recursive`
- `--selector`
- `--show-managed-fields`
- `--template`

### `coily kubectl rollout status`

Show the status of the rollout.

Flags:

- `--filename`
- `--kustomize`
- `--recursive`
- `--revision`
- `--selector`
- `--timeout`
- `--watch`

### `coily kubectl rollout undo`

Roll back to a previous rollout.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--filename`
- `--kustomize`
- `--output`
- `--recursive`
- `--selector`
- `--show-managed-fields`
- `--template`
- `--to-revision`

## `coily kubectl scale`

### `coily kubectl scale`

Set a new size for a deployment, replica set, replication controller, or stateful set.

Flags:

- `--all`
- `--allow-missing-template-keys`
- `--current-replicas`
- `--dry-run`
- `--filename`
- `--kustomize`
- `--output`
- `--recursive`
- `--replicas`
- `--resource-version`
- `--selector`
- `--show-managed-fields`
- `--template`
- `--timeout`

## `coily kubectl taint`

### `coily kubectl taint`

Update the taints on one or more nodes.

Flags:

- `--all`
- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--output`
- `--overwrite`
- `--selector`
- `--show-managed-fields`
- `--template`
- `--validate`

## `coily kubectl top`

### `coily kubectl top` (group)

Display Resource (CPU/Memory) usage.

Subcommands: `node`, `pod`

### `coily kubectl top node`

Display resource (CPU/memory) usage of nodes.

Flags:

- `--no-headers`
- `--selector`
- `--show-capacity`
- `--sort-by`
- `--use-protocol-buffers`

### `coily kubectl top pod`

Display resource (CPU/memory) usage of pods.

Flags:

- `--all-namespaces`
- `--containers`
- `--field-selector`
- `--no-headers`
- `--selector`
- `--sort-by`
- `--sum`
- `--use-protocol-buffers`

## `coily kubectl uncordon`

### `coily kubectl uncordon`

Mark node as schedulable.

Flags:

- `--dry-run`
- `--selector`

## `coily kubectl wait`

### `coily kubectl wait`

Experimental: Wait for a specific condition on one or many resources.

Flags:

- `--all`
- `--all-namespaces`
- `--allow-missing-template-keys`
- `--field-selector`
- `--filename`
- `--for`
- `--local`
- `--output`
- `--recursive`
- `--selector`
- `--show-managed-fields`
- `--template`
- `--timeout`
