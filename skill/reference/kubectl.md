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

Display the current-context.

### `coily kubectl config delete-cluster`

Delete the specified cluster from the kubeconfig.

### `coily kubectl config delete-context`

Delete the specified context from the kubeconfig.

### `coily kubectl config delete-user`

Delete the specified user from the kubeconfig.

### `coily kubectl config get-clusters`

Display clusters defined in the kubeconfig.

### `coily kubectl config get-contexts`

Display one or many contexts from the kubeconfig file.

Flags:

- `--no-headers`
- `--output`

### `coily kubectl config get-users`

Display users defined in the kubeconfig.

### `coily kubectl config rename-context`

Renames a context from the kubeconfig file.

### `coily kubectl config set`

Set an individual value in a kubeconfig file.

Flags:

- `--set-raw-bytes`

### `coily kubectl config set-cluster`

Set a cluster entry in kubeconfig.

Flags:

- `--certificate-authority`
- `--embed-certs`
- `--insecure-skip-tls-verify`
- `--proxy-url`
- `--server`
- `--tls-server-name`

### `coily kubectl config set-context`

Set a context entry in kubeconfig.

Flags:

- `--cluster`
- `--current`
- `--namespace`
- `--user`

### `coily kubectl config set-credentials`

Set a user entry in kubeconfig.

Flags:

- `--client-certificate`
- `--token`
- `--username`
- `--auth-provider`
- `--auth-provider-arg`
- `--client-key`
- `--embed-certs`
- `--exec-api-version`
- `--exec-arg`
- `--exec-command`
- `--exec-env`
- `--password`

### `coily kubectl config unset`

Unset an individual value in a kubeconfig file.

### `coily kubectl config use-context`

Set the current-context in a kubeconfig file.

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

## `coily kubectl create`

### `coily kubectl create` (group)

Create a resource from a file or from stdin.

Subcommands: `clusterrole`, `clusterrolebinding`, `configmap`, `cronjob`, `deployment`, `ingress`, `job`, `namespace`, `poddisruptionbudget`, `priorityclass`, `quota`, `role`, `rolebinding`, `secret`, `service`, `serviceaccount`, `token`

### `coily kubectl create clusterrole`

Create a cluster role.

Flags:

- `--aggregation-rule`
- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--non-resource-url`
- `--output`
- `--resource`
- `--resource-name`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`
- `--verb`

### `coily kubectl create clusterrolebinding`

Create a cluster role binding for a particular cluster role.

Flags:

- `--allow-missing-template-keys`
- `--clusterrole`
- `--dry-run`
- `--field-manager`
- `--group`
- `--output`
- `--save-config`
- `--serviceaccount`
- `--show-managed-fields`
- `--template`
- `--user`
- `--validate`

### `coily kubectl create configmap`

Create a config map based on a file, directory, or specified literal value.

Flags:

- `--allow-missing-template-keys`
- `--append-hash`
- `--dry-run`
- `--field-manager`
- `--from-env-file`
- `--from-file`
- `--from-literal`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create cronjob`

Create a cron job with the specified name.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--image`
- `--output`
- `--restart`
- `--save-config`
- `--schedule`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create deployment`

Create a deployment with the specified name.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--image`
- `--output`
- `--port`
- `--replicas`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create ingress`

Create an ingress with the specified name.

Flags:

- `--annotation`
- `--rule`
- `--default-backend`
- `--allow-missing-template-keys`
- `--class`
- `--dry-run`
- `--field-manager`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create job`

Create a job with the specified name.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--from`
- `--image`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create namespace`

Create a namespace with the specified name.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create poddisruptionbudget`

Create a pod disruption budget with the specified name, selector, and desired minimum available pods.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--max-unavailable`
- `--min-available`
- `--output`
- `--save-config`
- `--selector`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create priorityclass`

Create a priority class with the specified name, value, globalDefault and description.

Flags:

- `--allow-missing-template-keys`
- `--description`
- `--dry-run`
- `--field-manager`
- `--global-default`
- `--output`
- `--preemption-policy`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`
- `--value`

### `coily kubectl create quota`

Create a resource quota with the specified name, hard limits, and optional scopes.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--hard`
- `--output`
- `--save-config`
- `--scopes`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create role`

Create a role with single rule.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--output`
- `--resource`
- `--resource-name`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`
- `--verb`

### `coily kubectl create rolebinding`

Create a role binding for a particular role or cluster role.

Flags:

- `--allow-missing-template-keys`
- `--clusterrole`
- `--dry-run`
- `--field-manager`
- `--group`
- `--output`
- `--role`
- `--save-config`
- `--serviceaccount`
- `--show-managed-fields`
- `--template`
- `--user`
- `--validate`

### `coily kubectl create secret` (group)

Create a secret using specified subcommand.

Subcommands: `docker-registry`, `generic`, `tls`

### `coily kubectl create secret docker-registry`

Create a new secret for use with Docker registries.

Flags:

- `--allow-missing-template-keys`
- `--append-hash`
- `--docker-email`
- `--docker-password`
- `--docker-server`
- `--docker-username`
- `--dry-run`
- `--field-manager`
- `--from-file`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create secret generic`

Create a secret based on a file, directory, or specified literal value.

Flags:

- `--allow-missing-template-keys`
- `--append-hash`
- `--dry-run`
- `--field-manager`
- `--from-env-file`
- `--from-file`
- `--from-literal`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--type`
- `--validate`

### `coily kubectl create secret tls`

Create a TLS secret from the given public/private key pair.

Flags:

- `--allow-missing-template-keys`
- `--append-hash`
- `--cert`
- `--dry-run`
- `--field-manager`
- `--key`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create service` (group)

Create a service using a specified subcommand.

Subcommands: `clusterip`, `externalname`, `loadbalancer`, `nodeport`

### `coily kubectl create service clusterip`

Create a ClusterIP service with the specified name.

Flags:

- `--allow-missing-template-keys`
- `--clusterip`
- `--dry-run`
- `--field-manager`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--tcp`
- `--template`
- `--validate`

### `coily kubectl create service externalname`

Create an ExternalName service with the specified name.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--external-name`
- `--field-manager`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--tcp`
- `--template`
- `--validate`

### `coily kubectl create service loadbalancer`

Create a LoadBalancer service with the specified name.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--tcp`
- `--template`
- `--validate`

### `coily kubectl create service nodeport`

Create a NodePort service with the specified name.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--node-port`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--tcp`
- `--template`
- `--validate`

### `coily kubectl create serviceaccount`

Create a service account with the specified name.

Flags:

- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--output`
- `--save-config`
- `--show-managed-fields`
- `--template`
- `--validate`

### `coily kubectl create token`

Request a service account token.

Flags:

- `--allow-missing-template-keys`
- `--audience`
- `--bound-object-kind`
- `--bound-object-name`
- `--bound-object-uid`
- `--duration`
- `--output`
- `--show-managed-fields`
- `--template`

## `coily kubectl delete`

### `coily kubectl delete`

Delete resources by file names, stdin, resources and names, or by resources and label selector.

Flags:

- `--all`
- `--all-namespaces`
- `--cascade`
- `--dry-run`
- `--field-selector`
- `--filename`
- `--force`
- `--grace-period`
- `--ignore-not-found`
- `--kustomize`
- `--now`
- `--output`
- `--raw`
- `--recursive`
- `--selector`
- `--timeout`
- `--wait`

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

## `coily kubectl explain`

### `coily kubectl explain`

List the fields for supported resources.

Flags:

- `--api-version`
- `--output`
- `--recursive`

## `coily kubectl expose`

### `coily kubectl expose`

Expose a resource as a new Kubernetes service.

Flags:

- `--allow-missing-template-keys`
- `--cluster-ip`
- `--dry-run`
- `--external-ip`
- `--field-manager`
- `--filename`
- `--kustomize`
- `--labels`
- `--load-balancer-ip`
- `--name`
- `--output`
- `--override-type`
- `--overrides`
- `--port`
- `--protocol`
- `--recursive`
- `--save-config`
- `--selector`
- `--session-affinity`
- `--show-managed-fields`
- `--target-port`
- `--template`
- `--type`

## `coily kubectl get`

### `coily kubectl get`

Display one or many resources.

Flags:

- `--all-namespaces`
- `--allow-missing-template-keys`
- `--chunk-size`
- `--field-selector`
- `--filename`
- `--ignore-not-found`
- `--kustomize`
- `--label-columns`
- `--no-headers`
- `--output`
- `--output-watch-events`
- `--raw`
- `--recursive`
- `--selector`
- `--server-print`
- `--show-kind`
- `--show-labels`
- `--show-managed-fields`
- `--sort-by`
- `--subresource`
- `--template`
- `--watch`
- `--watch-only`

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

## `coily kubectl set`

### `coily kubectl set` (group)

Configure application resources.

Subcommands: `env`, `image`, `resources`, `selector`, `serviceaccount`, `subject`

### `coily kubectl set env`

Update environment variables on a pod template.

Flags:

- `--all`
- `--allow-missing-template-keys`
- `--containers`
- `--dry-run`
- `--env`
- `--field-manager`
- `--filename`
- `--from`
- `--keys`
- `--kustomize`
- `--list`
- `--local`
- `--output`
- `--overwrite`
- `--prefix`
- `--recursive`
- `--resolve`
- `--selector`
- `--show-managed-fields`
- `--template`

### `coily kubectl set image`

Update existing container image(s) of resources.

Flags:

- `--all`
- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--filename`
- `--kustomize`
- `--local`
- `--output`
- `--recursive`
- `--selector`
- `--show-managed-fields`
- `--template`

### `coily kubectl set resources`

Specify compute resource requirements (CPU, memory) for any resource that defines a pod template.

Flags:

- `--all`
- `--allow-missing-template-keys`
- `--containers`
- `--dry-run`
- `--field-manager`
- `--filename`
- `--kustomize`
- `--limits`
- `--local`
- `--output`
- `--recursive`
- `--requests`
- `--selector`
- `--show-managed-fields`
- `--template`

### `coily kubectl set selector`

Set the selector on a resource.

Flags:

- `--all`
- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--filename`
- `--local`
- `--output`
- `--recursive`
- `--resource-version`
- `--show-managed-fields`
- `--template`

### `coily kubectl set serviceaccount`

Update the service account of pod template resources.

Flags:

- `--all`
- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--filename`
- `--kustomize`
- `--local`
- `--output`
- `--recursive`
- `--show-managed-fields`
- `--template`

### `coily kubectl set subject`

Update the user, group, or service account in a role binding or cluster role binding.

Flags:

- `--all`
- `--allow-missing-template-keys`
- `--dry-run`
- `--field-manager`
- `--filename`
- `--group`
- `--kustomize`
- `--local`
- `--output`
- `--recursive`
- `--selector`
- `--serviceaccount`
- `--show-managed-fields`
- `--template`
- `--user`

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
