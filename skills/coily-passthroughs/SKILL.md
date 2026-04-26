---
name: coily-passthroughs
description: |
  Use when a shell command is denied by Claude Code's permission system
  (e.g. "Permission to use Bash with command X has been denied"), when
  reaching for aws, gh, kubectl, docker, tailscale, ssh, or scp against
  Kai's homelab, AWS account, or coilysiren resources, or when checking
  whether a privileged op has a coily wrapper. The body is a flat lookup
  table of every coily command.
---

# coily passthroughs

Auto-generated lookup table of every coily verb. Regenerate with `coily lockdown skill`.

Format: full path, one-line summary, comma-separated flag names. No flag descriptions; click into `coily <path> --help` for those.

## `coily aws route53 activate-key-signing-key`

Activates a key-signing key (KSK) so that it can be used for signing by DNSSEC.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 associate-vpc-with-hosted-zone`

Associates an Amazon VPC with a private hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --comment, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --vpc

## `coily aws route53 change-cidr-collection`

Creates, changes, or deletes CIDR blocks within a collection.

Flags: --ca-bundle, --changes, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --collection-version, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 change-resource-record-sets`

Creates, changes, or deletes a resource record set, which contains authoritative DNS information for a specified domain name or subdomain name.

Flags: --ca-bundle, --change-batch, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 change-tags-for-resource`

Adds, edits, or deletes tags for a health check or a hosted zone.

Flags: --add-tags, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --remove-tag-keys, --resource-id, --resource-type, --version

## `coily aws route53 create-cidr-collection`

Creates a CIDR collection in the current Amazon Web Services account.

Flags: --ca-bundle, --caller-reference, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 create-health-check`

Creates a new health check.

Flags: --ca-bundle, --caller-reference, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --health-check-config, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 create-hosted-zone`

Creates a new public or private hosted zone.

Flags: --ca-bundle, --caller-reference, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --delegation-set-id, --endpoint-url, --generate-cli-skeleton, --hosted-zone-config, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --vpc

## `coily aws route53 create-key-signing-key`

Creates a new key-signing key (KSK) associated with a hosted zone.

Flags: --ca-bundle, --caller-reference, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --key-management-service-arn, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --status, --version

## `coily aws route53 create-query-logging-config`

Creates a configuration for DNS query logging.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --cloud-watch-logs-log-group-arn, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 create-reusable-delegation-set`

Creates a delegation set (a group of four name servers) that can be reused by multiple hosted zones that were created by the same Amazon Web Services account.

Flags: --ca-bundle, --caller-reference, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 create-traffic-policy`

Creates a traffic policy, which you use to create multiple DNS resource record sets for one domain name (such as example.com) or one subdomain name (such as www...

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --comment, --debug, --document, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 create-traffic-policy-instance`

Creates resource record sets in a specified hosted zone based on the settings in a specified traffic policy version.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --traffic-policy-id, --traffic-policy-version, --ttl, --version

## `coily aws route53 create-traffic-policy-version`

Creates a new version of an existing traffic policy.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --comment, --debug, --document, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 create-vpc-association-authorization`

Authorizes the Amazon Web Services account that created a specified VPC to submit an AssociateVPCWithHostedZone request to associate the VPC with a specified ho...

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --vpc

## `coily aws route53 deactivate-key-signing-key`

Deactivates a key-signing key (KSK) so that it will not be used for signing by DNSSEC.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 delete-cidr-collection`

Deletes a CIDR collection in the current Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 delete-health-check`

Deletes a health check.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --health-check-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 delete-hosted-zone`

Deletes a hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 delete-key-signing-key`

Deletes a key-signing key (KSK).

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 delete-query-logging-config`

Deletes a configuration for DNS query logging.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 delete-reusable-delegation-set`

Deletes a reusable delegation set.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 delete-traffic-policy`

Deletes a traffic policy.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --traffic-policy-version, --version

## `coily aws route53 delete-traffic-policy-instance`

Deletes a traffic policy instance and all of the resource record sets that Amazon Route 53 created when you created the instance.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 delete-vpc-association-authorization`

Removes authorization to submit an AssociateVPCWithHostedZone request to associate a specified VPC with a hosted zone that was created by a different account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --vpc

## `coily aws route53 disable-hosted-zone-dnssec`

Disables DNSSEC signing in a specific hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 disassociate-vpc-from-hosted-zone`

Disassociates an Amazon Virtual Private Cloud (Amazon VPC) from an Amazon Route 53 private hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --comment, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --vpc

## `coily aws route53 enable-hosted-zone-dnssec`

Enables DNSSEC signing in a specific hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-account-limit`

Gets the specified limit for the current account, for example, the maximum number of health checks that you can create using the account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --type, --version

## `coily aws route53 get-change`

Returns the current status of a change batch request.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-checker-ip-ranges`

Route 53 does not perform authorization for this API because it retrieves information that is already available to the public.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-dnssec`

Returns information about DNSSEC for a specific hosted zone, including the key-signing keys (KSKs) in the hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-geo-location`

Gets information about whether a specified geographic location is supported for Amazon Route 53 geolocation resource record sets.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --continent-code, --country-code, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --subdivision-code, --version

## `coily aws route53 get-health-check`

Gets information about a specified health check.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --health-check-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-health-check-count`

Retrieves the number of health checks that are associated with the current Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-health-check-last-failure-reason`

Gets the reason that a specified health check failed most recently.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --health-check-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-health-check-status`

Gets status of a specified health check.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --health-check-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-hosted-zone`

Gets information about a specified hosted zone including the four name servers assigned to the hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-hosted-zone-count`

Retrieves the number of hosted zones that are associated with the current Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-hosted-zone-limit`

Gets the specified limit for a specified hosted zone, for example, the maximum number of records that you can create in the hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --type, --version

## `coily aws route53 get-query-logging-config`

Gets information about a specified configuration for DNS query logging.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-reusable-delegation-set`

Retrieves information about a specified reusable delegation set, including the four name servers that are assigned to the delegation set.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-reusable-delegation-set-limit`

Gets the maximum number of hosted zones that you can associate with the specified reusable delegation set.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --delegation-set-id, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --type, --version

## `coily aws route53 get-traffic-policy`

Gets information about a specific traffic policy version.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --traffic-policy-version, --version

## `coily aws route53 get-traffic-policy-instance`

Gets information about a specified traffic policy instance.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 get-traffic-policy-instance-count`

Gets the number of traffic policy instances that are associated with the current Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 list-cidr-blocks`

Returns a paginated list of location objects and their CIDR blocks.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --collection-id, --color, --debug, --endpoint-url, --generate-cli-skeleton, --location-name, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws route53 list-cidr-collections`

Returns a paginated list of CIDR collections in the Amazon Web Services account (metadata only).

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws route53 list-cidr-locations`

Returns a paginated list of CIDR locations for the given collection (metadata only, does not include CIDR blocks).

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --collection-id, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws route53 list-geo-locations`

Retrieves a list of supported geographic locations.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --start-continent-code, --start-country-code, --start-subdivision-code, --version

## `coily aws route53 list-health-checks`

Retrieve a list of the health checks that are associated with the current Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws route53 list-hosted-zones`

Retrieves a list of the public and private hosted zones that are associated with the current Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --delegation-set-id, --endpoint-url, --generate-cli-skeleton, --hosted-zone-type, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws route53 list-hosted-zones-by-name`

Retrieves a list of your hosted zones in lexicographic order.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --dns-name, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 list-hosted-zones-by-vpc`

Lists all the private hosted zones that a specified VPC is associated with, regardless of which Amazon Web Services account or Amazon Web Services service owns ...

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --next-token, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --vpc-id, --vpc-region

## `coily aws route53 list-query-logging-configs`

Lists the configurations for DNS query logging that are associated with the current Amazon Web Services account or the configuration that is associated with a s...

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws route53 list-resource-record-sets`

Lists the resource record sets in a specified hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws route53 list-reusable-delegation-sets`

Retrieves a list of the reusable delegation sets that are associated with the current Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --marker, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 list-tags-for-resource`

Lists tags for one health check or hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --resource-id, --resource-type, --version

## `coily aws route53 list-tags-for-resources`

Lists tags for up to 10 health checks or hosted zones.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --resource-ids, --resource-type, --version

## `coily aws route53 list-traffic-policies`

Gets information about the latest version for every traffic policy that is associated with the current Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --traffic-policy-id-marker, --version

## `coily aws route53 list-traffic-policy-instances`

Gets information about the traffic policy instances that you created by using the current Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id-marker, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --traffic-policy-instance-name-marker, --traffic-policy-instance-type-marker, --version

## `coily aws route53 list-traffic-policy-instances-by-policy`

Gets information about the traffic policy instances that you created by using a specify traffic policy version.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id-marker, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --traffic-policy-id, --traffic-policy-instance-name-marker, --traffic-policy-instance-type-marker, --traffic-policy-version, --version

## `coily aws route53 list-traffic-policy-versions`

Gets information about all of the versions for a specified traffic policy.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --traffic-policy-version-marker, --version

## `coily aws route53 list-vpc-association-authorizations`

Gets a list of the VPCs that were created by other accounts and that can be associated with a specified hosted zone because you've submitted one or more CreateV...

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --max-items, --max-results, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --starting-token, --version

## `coily aws route53 test-dns-answer`

Gets the value that Amazon Route 53 returns in response to a DNS request for a specified record name and type.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --edns0-client-subnet-ip, --edns0-client-subnet-mask, --endpoint-url, --generate-cli-skeleton, --hosted-zone-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --record-name, --record-type, --region, --resolver-ip, --version

## `coily aws route53 update-health-check`

Updates an existing health check.

Flags: --alarm-identifier, --ca-bundle, --child-health-checks, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --disabled, --enable-sni, --endpoint-url, --failure-threshold, --fully-qualified-domain-name, --generate-cli-skeleton, --health-check-id, --health-check-version, --health-threshold, --insufficient-data-health-status, --inverted, --ip-address, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --port, --profile, --query, --region, --regions, --reset-elements, --resource-path, --search-string, --version

## `coily aws route53 update-hosted-zone-comment`

Updates the comment for a specified hosted zone.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --comment, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws route53 update-traffic-policy-comment`

Updates the comment for a specified traffic policy version.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --comment, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --traffic-policy-version, --version

## `coily aws route53 update-traffic-policy-instance`

NOTE: After you submit a UpdateTrafficPolicyInstance request, there's a brief delay while Route 53 creates the resource record sets that are specified in the tr...

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --traffic-policy-id, --traffic-policy-version, --ttl, --version

## `coily aws route53 wait resource-record-sets-changed`

Wait until JMESPath query ChangeInfo.Status returns INSYNC when polling with get-change.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3 cp`

Copies a local file or S3 object to another location locally or in S3.

Flags: --acl, --ca-bundle, --cache-control, --checksum-algorithm, --checksum-mode, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --content-disposition, --content-encoding, --content-language, --content-type, --copy-props, --debug, --dryrun, --endpoint-url, --exclude, --expected-size, --expires, --follow-symlinks, --force-glacier-transfer, --grants, --ignore-glacier-warnings, --include, --metadata, --metadata-directive, --no-cli-auto-prompt, --no-cli-pager, --no-guess-mime-type, --no-paginate, --no-progress, --no-sign-request, --no-verify-ssl, --only-show-errors, --output, --page-size, --profile, --query, --quiet, --recursive, --region, --request-payer, --source-region, --sse, --sse-c, --sse-c-copy-source, --sse-c-copy-source-key, --sse-c-key, --sse-kms-key-id, --storage-class, --version, --website-redirect

## `coily aws s3 ls`

List S3 objects and common prefixes under a prefix or all S3 buckets.

Flags: --bucket-name-prefix, --bucket-region, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --endpoint-url, --human-readable, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --recursive, --region, --request-payer, --summarize, --version

## `coily aws s3 mb`

Creates an S3 bucket.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --endpoint-url, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3 mv`

Moves a local file or S3 object to another location locally or in S3.

Flags: --acl, --ca-bundle, --cache-control, --checksum-algorithm, --checksum-mode, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --content-disposition, --content-encoding, --content-language, --content-type, --copy-props, --debug, --dryrun, --endpoint-url, --exclude, --expires, --follow-symlinks, --force-glacier-transfer, --grants, --ignore-glacier-warnings, --include, --metadata, --metadata-directive, --no-cli-auto-prompt, --no-cli-pager, --no-guess-mime-type, --no-paginate, --no-progress, --no-sign-request, --no-verify-ssl, --only-show-errors, --output, --page-size, --profile, --query, --quiet, --recursive, --region, --request-payer, --source-region, --sse, --sse-c, --sse-c-copy-source, --sse-c-copy-source-key, --sse-c-key, --sse-kms-key-id, --storage-class, --validate-same-s3-paths, --version, --website-redirect

## `coily aws s3 presign`

Generate a pre-signed URL for an Amazon S3 object.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --endpoint-url, --expires-in, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3 rb`

Deletes an empty S3 bucket.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --endpoint-url, --force, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3 rm`

Deletes an S3 object.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --dryrun, --endpoint-url, --exclude, --include, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --only-show-errors, --output, --page-size, --profile, --query, --quiet, --recursive, --region, --request-payer, --version

## `coily aws s3 sync`

Syncs directories and S3 prefixes.

Flags: --acl, --ca-bundle, --cache-control, --checksum-algorithm, --checksum-mode, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --content-disposition, --content-encoding, --content-language, --content-type, --copy-props, --debug, --delete, --dryrun, --endpoint-url, --exact-timestamps, --exclude, --expires, --follow-symlinks, --force-glacier-transfer, --grants, --ignore-glacier-warnings, --include, --metadata, --metadata-directive, --no-cli-auto-prompt, --no-cli-pager, --no-guess-mime-type, --no-paginate, --no-progress, --no-sign-request, --no-verify-ssl, --only-show-errors, --output, --page-size, --profile, --query, --quiet, --region, --request-payer, --size-only, --source-region, --sse, --sse-c, --sse-c-copy-source, --sse-c-copy-source-key, --sse-c-key, --sse-kms-key-id, --storage-class, --version, --website-redirect

## `coily aws s3 website`

Set the website configuration for a bucket.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --endpoint-url, --error-document, --index-document, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api abort-multipart-upload`

This operation aborts a multipart upload.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --if-match-initiated-time, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --upload-id, --version

## `coily aws s3api complete-multipart-upload`

Completes a multipart upload by assembling previously uploaded parts.

Flags: --bucket, --ca-bundle, --checksum-crc32, --checksum-crc32-c, --checksum-crc64-nvme, --checksum-sha1, --checksum-sha256, --checksum-type, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --if-match, --if-none-match, --key, --mpu-object-size, --multipart-upload, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --upload-id, --version

## `coily aws s3api copy-object`

Creates a copy of an object that is already stored in Amazon S3.

Flags: --acl, --bucket, --bucket-key-enabled, --ca-bundle, --cache-control, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-disposition, --content-encoding, --content-language, --content-type, --copy-source, --copy-source-if-match, --copy-source-if-modified-since, --copy-source-if-none-match, --copy-source-if-unmodified-since, --copy-source-sse-customer-algorithm, --copy-source-sse-customer-key, --copy-source-sse-customer-key-md5, --debug, --endpoint-url, --expected-bucket-owner, --expected-source-bucket-owner, --expires, --generate-cli-skeleton, --grant-full-control, --grant-read, --grant-read-acp, --grant-write-acp, --key, --metadata, --metadata-directive, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --object-lock-legal-hold-status, --object-lock-mode, --object-lock-retain-until-date, --output, --profile, --query, --region, --request-payer, --server-side-encryption, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --ssekms-encryption-context, --ssekms-key-id, --storage-class, --tagging, --tagging-directive, --version, --website-redirect-location

## `coily aws s3api create-bucket`

NOTE: This action creates an Amazon S3 bucket.

Flags: --acl, --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --create-bucket-configuration, --debug, --endpoint-url, --generate-cli-skeleton, --grant-full-control, --grant-read, --grant-read-acp, --grant-write, --grant-write-acp, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --object-ownership, --output, --profile, --query, --region, --version

## `coily aws s3api create-multipart-upload`

This action initiates a multipart upload and returns an upload ID.

Flags: --acl, --bucket, --bucket-key-enabled, --ca-bundle, --cache-control, --checksum-algorithm, --checksum-type, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-disposition, --content-encoding, --content-language, --content-type, --debug, --endpoint-url, --expected-bucket-owner, --expires, --generate-cli-skeleton, --grant-full-control, --grant-read, --grant-read-acp, --grant-write-acp, --key, --metadata, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --object-lock-legal-hold-status, --object-lock-mode, --object-lock-retain-until-date, --output, --profile, --query, --region, --request-payer, --server-side-encryption, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --ssekms-encryption-context, --ssekms-key-id, --storage-class, --tagging, --version, --website-redirect-location

## `coily aws s3api create-session`

Creates a session that establishes temporary security credentials to support fast authentication and authorization for the Zonal endpoint API operations on dire...

Flags: --bucket, --bucket-key-enabled, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --server-side-encryption, --session-mode, --ssekms-encryption-context, --ssekms-key-id, --version

## `coily aws s3api delete-bucket`

Deletes the S3 bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-analytics-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-cors`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-encryption`

This implementation of the DELETE action resets the default encryption for the bucket as server-side encryption with Amazon S3 managed keys (SSE-S3).

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-inventory-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-lifecycle`

Deletes the lifecycle configuration from the specified bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-metrics-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-ownership-controls`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-policy`

Deletes the policy of a specified bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-replication`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-tagging`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-bucket-website`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api delete-object`

Removes an object from a bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --if-match, --if-match-last-modified-time, --if-match-size, --key, --mfa, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --version, --version-id

## `coily aws s3api delete-object-tagging`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --version-id

## `coily aws s3api delete-objects`

This operation enables you to delete multiple objects from a bucket using a single HTTP request.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --delete, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --mfa, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --version

## `coily aws s3api delete-public-access-block`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-accelerate-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --version

## `coily aws s3api get-bucket-acl`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-analytics-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-cors`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-encryption`

Returns the default encryption configuration for an Amazon S3 bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-inventory-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-lifecycle-configuration`

Returns the lifecycle configuration information set on the bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-location`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-logging`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-metadata-table-configuration`

Retrieves the metadata table configuration for a general purpose bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-metrics-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-notification-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-ownership-controls`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-policy`

Returns the policy of a specified bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-policy-status`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-replication`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-request-payment`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-tagging`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-versioning`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-bucket-website`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-object`

Retrieves an object from Amazon S3.

Flags: --bucket, --ca-bundle, --checksum-mode, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --if-match, --if-modified-since, --if-none-match, --if-unmodified-since, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --part-number, --profile, --query, --range, --region, --request-payer, --response-cache-control, --response-content-disposition, --response-content-encoding, --response-content-language, --response-content-type, --response-expires, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --version, --version-id

## `coily aws s3api get-object-acl`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --version, --version-id

## `coily aws s3api get-object-attributes`

Retrieves all the metadata from an object without returning the object itself.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --max-parts, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --object-attributes, --output, --part-number-marker, --profile, --query, --region, --request-payer, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --version, --version-id

## `coily aws s3api get-object-legal-hold`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --version, --version-id

## `coily aws s3api get-object-lock-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api get-object-retention`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --version, --version-id

## `coily aws s3api get-object-tagging`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --version, --version-id

## `coily aws s3api get-object-torrent`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --version

## `coily aws s3api get-public-access-block`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api head-bucket`

You can use this operation to determine if a bucket exists and if you have permission to access it.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api head-object`

The HEAD operation retrieves metadata from an object without returning the object itself.

Flags: --bucket, --ca-bundle, --checksum-mode, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --if-match, --if-modified-since, --if-none-match, --if-unmodified-since, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --part-number, --profile, --query, --range, --region, --request-payer, --response-cache-control, --response-content-disposition, --response-content-encoding, --response-content-language, --response-content-type, --response-expires, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --version, --version-id

## `coily aws s3api list-bucket-analytics-configurations`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --continuation-token, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api list-bucket-inventory-configurations`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --continuation-token, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api list-bucket-metrics-configurations`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --continuation-token, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api list-buckets`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket-region, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --prefix, --profile, --query, --region, --starting-token, --version

## `coily aws s3api list-directory-buckets`

Returns a list of all Amazon S3 directory buckets owned by the authenticated sender of the request.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws s3api list-multipart-uploads`

This operation lists in-progress multipart uploads in a bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --delimiter, --encoding-type, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --prefix, --profile, --query, --region, --request-payer, --starting-token, --version

## `coily aws s3api list-object-versions`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --delimiter, --encoding-type, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --optional-object-attributes, --output, --page-size, --prefix, --profile, --query, --region, --request-payer, --starting-token, --version

## `coily aws s3api list-objects`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --delimiter, --encoding-type, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --optional-object-attributes, --output, --page-size, --prefix, --profile, --query, --region, --request-payer, --starting-token, --version

## `coily aws s3api list-objects-v2`

Returns some or all (up to 1,000) of the objects in a bucket with each request.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --delimiter, --encoding-type, --endpoint-url, --expected-bucket-owner, --fetch-owner, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --optional-object-attributes, --output, --page-size, --prefix, --profile, --query, --region, --request-payer, --start-after, --starting-token, --version

## `coily aws s3api list-parts`

Lists the parts that have been uploaded for a specific multipart upload.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --request-payer, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --starting-token, --upload-id, --version

## `coily aws s3api put-bucket-accelerate-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --accelerate-configuration, --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api put-bucket-acl`

NOTE: This operation is not supported for directory buckets.

Flags: --access-control-policy, --acl, --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --grant-full-control, --grant-read, --grant-read-acp, --grant-write, --grant-write-acp, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api put-bucket-analytics-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --analytics-configuration, --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api put-bucket-cors`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --cors-configuration, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api put-bucket-encryption`

This operation configures default encryption and Amazon S3 Bucket Keys for an existing bucket.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --server-side-encryption-configuration, --version

## `coily aws s3api put-bucket-inventory-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --id, --inventory-configuration, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api put-bucket-lifecycle-configuration`

Creates a new lifecycle configuration for the bucket or replaces an existing lifecycle configuration.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --lifecycle-configuration, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --transition-default-minimum-object-size, --version

## `coily aws s3api put-bucket-logging`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --bucket-logging-status, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api put-bucket-metrics-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --id, --metrics-configuration, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api put-bucket-notification-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --notification-configuration, --output, --profile, --query, --region, --version

## `coily aws s3api put-bucket-ownership-controls`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --ownership-controls, --profile, --query, --region, --version

## `coily aws s3api put-bucket-policy`

Applies an Amazon S3 bucket policy to an Amazon S3 bucket.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-confirm-remove-self-bucket-access, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --policy, --profile, --query, --region, --version

## `coily aws s3api put-bucket-replication`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --replication-configuration, --token, --version

## `coily aws s3api put-bucket-request-payment`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payment-configuration, --version

## `coily aws s3api put-bucket-tagging`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --tagging, --version

## `coily aws s3api put-bucket-versioning`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --mfa, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --versioning-configuration

## `coily aws s3api put-bucket-website`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --website-configuration

## `coily aws s3api put-object`

<string>:: (ERROR/3) Anonymous hyperlink mismatch: 2 references but 0 targets.

Flags: --acl, --bucket, --bucket-key-enabled, --ca-bundle, --cache-control, --checksum-algorithm, --checksum-crc32, --checksum-crc32-c, --checksum-crc64-nvme, --checksum-sha1, --checksum-sha256, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-disposition, --content-encoding, --content-language, --content-length, --content-md5, --content-type, --debug, --endpoint-url, --expected-bucket-owner, --expires, --generate-cli-skeleton, --grant-full-control, --grant-read, --grant-read-acp, --grant-write-acp, --if-match, --if-none-match, --key, --metadata, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --object-lock-legal-hold-status, --object-lock-mode, --object-lock-retain-until-date, --output, --profile, --query, --region, --request-payer, --server-side-encryption, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --ssekms-encryption-context, --ssekms-key-id, --storage-class, --tagging, --version, --website-redirect-location, --write-offset-bytes

## `coily aws s3api put-object-acl`

NOTE: This operation is not supported for directory buckets.

Flags: --access-control-policy, --acl, --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --grant-full-control, --grant-read, --grant-read-acp, --grant-write, --grant-write-acp, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --version, --version-id

## `coily aws s3api put-object-legal-hold`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --legal-hold, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --version, --version-id

## `coily aws s3api put-object-lock-configuration`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --object-lock-configuration, --output, --profile, --query, --region, --request-payer, --token, --version

## `coily aws s3api put-object-retention`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --retention, --version, --version-id

## `coily aws s3api put-object-tagging`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --tagging, --version, --version-id

## `coily aws s3api put-public-access-block`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --public-access-block-configuration, --query, --region, --version

## `coily aws s3api restore-object`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --request-payer, --restore-request, --version, --version-id

## `coily aws s3api select-object-content`

NOTE: This operation is not supported for directory buckets.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --expression, --expression-type, --input-serialization, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --output-serialization, --profile, --query, --region, --request-progress, --scan-range, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --version

## `coily aws s3api upload-part`

Uploads a part in a multipart upload.

Flags: --bucket, --ca-bundle, --checksum-algorithm, --checksum-crc32, --checksum-crc32-c, --checksum-crc64-nvme, --checksum-sha1, --checksum-sha256, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-length, --content-md5, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --part-number, --profile, --query, --region, --request-payer, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --upload-id, --version

## `coily aws s3api upload-part-copy`

Uploads a part by copying data from an existing object as data source.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --copy-source, --copy-source-if-match, --copy-source-if-modified-since, --copy-source-if-none-match, --copy-source-if-unmodified-since, --copy-source-range, --copy-source-sse-customer-algorithm, --copy-source-sse-customer-key, --copy-source-sse-customer-key-md5, --debug, --endpoint-url, --expected-bucket-owner, --expected-source-bucket-owner, --generate-cli-skeleton, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --part-number, --profile, --query, --region, --request-payer, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --upload-id, --version

## `coily aws s3api wait bucket-exists`

Wait until 200 response is received when polling with head-bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api wait bucket-not-exists`

Wait until 404 response is received when polling with head-bucket.

Flags: --bucket, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws s3api wait object-exists`

Wait until 200 response is received when polling with head-object.

Flags: --bucket, --ca-bundle, --checksum-mode, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --if-match, --if-modified-since, --if-none-match, --if-unmodified-since, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --part-number, --profile, --query, --range, --region, --request-payer, --response-cache-control, --response-content-disposition, --response-content-encoding, --response-content-language, --response-content-type, --response-expires, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --version, --version-id

## `coily aws s3api wait object-not-exists`

Wait until 404 response is received when polling with head-object.

Flags: --bucket, --ca-bundle, --checksum-mode, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --expected-bucket-owner, --generate-cli-skeleton, --if-match, --if-modified-since, --if-none-match, --if-unmodified-since, --key, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --part-number, --profile, --query, --range, --region, --request-payer, --response-cache-control, --response-content-disposition, --response-content-encoding, --response-content-language, --response-content-type, --response-expires, --sse-customer-algorithm, --sse-customer-key, --sse-customer-key-md5, --version, --version-id

## `coily aws s3api write-get-object-response`

NOTE: This operation is not supported for directory buckets.

Flags: --accept-ranges, --bucket-key-enabled, --ca-bundle, --cache-control, --checksum-crc32, --checksum-crc32-c, --checksum-crc64-nvme, --checksum-sha1, --checksum-sha256, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content-disposition, --content-encoding, --content-language, --content-length, --content-range, --content-type, --debug, --delete-marker, --e-tag, --endpoint-url, --error-code, --error-message, --expiration, --expires, --generate-cli-skeleton, --last-modified, --metadata, --missing-meta, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --object-lock-legal-hold-status, --object-lock-mode, --object-lock-retain-until-date, --output, --parts-count, --profile, --query, --region, --replication-status, --request-charged, --request-route, --request-token, --restore, --server-side-encryption, --sse-customer-algorithm, --sse-customer-key-md5, --ssekms-key-id, --status-code, --storage-class, --tag-count, --version, --version-id

## `coily aws ssm add-tags-to-resource`

Adds or overwrites one or more tags for the specified resource.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --resource-id, --resource-type, --tags, --version

## `coily aws ssm associate-ops-item-related-item`

Associates a related item to a Systems Manager OpsCenter OpsItem.

Flags: --association-type, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --ops-item-id, --output, --profile, --query, --region, --resource-type, --resource-uri, --version

## `coily aws ssm cancel-command`

Attempts to cancel the command specified by the Command ID.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --command-id, --debug, --endpoint-url, --generate-cli-skeleton, --instance-ids, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm cancel-maintenance-window-execution`

Stops a maintenance window execution that is already in progress and cancels any tasks in the window that haven't already starting running.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --window-execution-id

## `coily aws ssm create-activation`

Generates an activation code and activation ID you can use to register your on-premises servers, edge devices, or virtual machine (VM) with Amazon Web Services ...

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --default-instance-name, --description, --endpoint-url, --expiration-date, --generate-cli-skeleton, --iam-role, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --registration-limit, --registration-metadata, --tags, --version

## `coily aws ssm create-association`

A State Manager association defines the state that you want to maintain on your managed nodes.

Flags: --alarm-configuration, --association-name, --automation-target-parameter-name, --ca-bundle, --calendar-names, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --compliance-severity, --debug, --document-version, --duration, --endpoint-url, --generate-cli-skeleton, --instance-id, --max-concurrency, --max-errors, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --output-location, --parameters, --profile, --query, --region, --schedule-expression, --schedule-offset, --sync-compliance, --tags, --target-locations, --target-maps, --targets, --version

## `coily aws ssm create-association-batch`

Associates the specified Amazon Web Services Systems Manager document (SSM document) with the specified managed nodes or targets.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --entries, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm create-document`

Creates a Amazon Web Services Systems Manager (SSM document).

Flags: --attachments, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content, --debug, --display-name, --document-format, --document-type, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --requires, --tags, --target-type, --version, --version-name

## `coily aws ssm create-maintenance-window`

Creates a new maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --client-token, --color, --cutoff, --debug, --description, --duration, --end-date, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --schedule, --schedule-offset, --schedule-timezone, --start-date, --tags, --version

## `coily aws ssm create-ops-item`

Creates a new OpsItem.

Flags: --account-id, --actual-end-time, --actual-start-time, --ca-bundle, --category, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --description, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --notifications, --operational-data, --ops-item-type, --output, --planned-end-time, --planned-start-time, --priority, --profile, --query, --region, --related-ops-items, --severity, --source, --tags, --title, --version

## `coily aws ssm create-ops-metadata`

If you create a new application in Application Manager, Amazon Web Services Systems Manager calls this API operation to specify information about the new applic...

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --metadata, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --resource-id, --tags, --version

## `coily aws ssm create-patch-baseline`

Creates a patch baseline.

Flags: --approval-rules, --approved-patches, --approved-patches-compliance-level, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --endpoint-url, --global-filters, --name, --no-approved-patches-enable-non-security, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --operating-system, --output, --profile, --query, --region, --rejected-patches, --rejected-patches-action, --version

## `coily aws ssm create-resource-data-sync`

A resource data sync helps you view data from multiple sources in a single location.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --s3-destination, --sync-name, --sync-source, --sync-type, --version

## `coily aws ssm delete-activation`

Deletes an activation.

Flags: --activation-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm delete-association`

Disassociates the specified Amazon Web Services Systems Manager document (SSM document) from the specified managed node.

Flags: --association-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --instance-id, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm delete-document`

Deletes the Amazon Web Services Systems Manager document (SSM document) and all managed node associations to the document.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --document-version, --endpoint-url, --force, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --version-name

## `coily aws ssm delete-inventory`

Delete a custom inventory type or the data associated with a custom Inventory type.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --client-token, --color, --debug, --dry-run, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --schema-delete-option, --type-name, --version

## `coily aws ssm delete-maintenance-window`

Deletes a maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --window-id

## `coily aws ssm delete-ops-item`

Delete an OpsItem.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --ops-item-id, --output, --profile, --query, --region, --version

## `coily aws ssm delete-ops-metadata`

Delete OpsMetadata related to an application.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --ops-metadata-arn, --output, --profile, --query, --region, --version

## `coily aws ssm delete-parameter`

Delete a parameter from the system.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm delete-parameters`

Delete a list of parameters.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --names, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm delete-patch-baseline`

Deletes a patch baseline.

Flags: --baseline-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm delete-resource-data-sync`

Deletes a resource data sync configuration.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --sync-name, --sync-type, --version

## `coily aws ssm delete-resource-policy`

Deletes a Systems Manager resource policy.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --policy-hash, --policy-id, --profile, --query, --region, --resource-arn, --version

## `coily aws ssm deregister-managed-instance`

Removes the server or virtual machine from the list of registered servers.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --instance-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm deregister-patch-baseline-for-patch-group`

Removes a patch group from a patch baseline.

Flags: --baseline-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --patch-group, --profile, --query, --region, --version

## `coily aws ssm deregister-target-from-maintenance-window`

Removes a target from a maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --safe, --version, --window-id, --window-target-id

## `coily aws ssm deregister-task-from-maintenance-window`

Removes a task from a maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --window-id, --window-task-id

## `coily aws ssm describe-activations`

Describes details about the activation, such as the date and time the activation was created, its expiration date, the Identity and Access Management (IAM) role...

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-association`

Describes the association for the specified target or managed node.

Flags: --association-id, --association-version, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --instance-id, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm describe-association-execution-targets`

Views information about a specific execution of a specific association.

Flags: --association-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --execution-id, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-association-executions`

Views all executions for a specific association ID.

Flags: --association-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-automation-executions`

Provides details about all active and terminated Automation executions.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-automation-step-executions`

Information about all active and terminated step executions in an Automation workflow.

Flags: --automation-execution-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --reverse-order, --starting-token, --version

## `coily aws ssm describe-available-patches`

Lists all patches eligible to be included in a patch baseline.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-document`

Describes the specified Amazon Web Services Systems Manager document (SSM document).

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --document-version, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --version-name

## `coily aws ssm describe-document-permission`

Describes the permissions for a Amazon Web Services Systems Manager document (SSM document).

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-results, --name, --next-token, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --permission-type, --profile, --query, --region, --version

## `coily aws ssm describe-effective-instance-associations`

All associations for the managed nodes.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --instance-id, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-instance-associations-status`

The status of the associations for the managed nodes.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --instance-id, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-instance-information`

Provides information about one or more of your managed nodes, including the operating system platform, SSM Agent version, association status, and IP address.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --instance-information-filter-list, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-instance-patch-states`

Retrieves the high-level patch state of one or more managed nodes.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --instance-ids, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-instance-patches`

Retrieves information about the patches on the specified managed node and their state relative to the patch baseline being used for the node.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --instance-id, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-instance-properties`

An API operation used by the Systems Manager console to display information about Systems Manager managed nodes.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters-with-operator, --generate-cli-skeleton, --instance-property-filter-list, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-inventory-deletions`

Describes a specific delete inventory operation.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --deletion-id, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-maintenance-window-executions`

Lists the executions of a maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version, --window-id

## `coily aws ssm describe-maintenance-window-schedule`

Retrieves information about upcoming executions of a maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --resource-type, --starting-token, --targets, --version, --window-id

## `coily aws ssm describe-maintenance-window-targets`

Lists the targets registered with the maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version, --window-id

## `coily aws ssm describe-maintenance-window-tasks`

Lists the tasks in a maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version, --window-id

## `coily aws ssm describe-maintenance-windows`

Retrieves the maintenance windows in an Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-maintenance-windows-for-target`

Retrieves information about the maintenance window targets or tasks that a managed node is associated with.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --resource-type, --starting-token, --targets, --version

## `coily aws ssm describe-ops-items`

Query a set of OpsItems.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --ops-item-filters, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-parameters`

Lists the parameters in your Amazon Web Services account or the parameters shared with you when you enable the Shared option.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --parameter-filters, --profile, --query, --region, --shared, --starting-token, --version

## `coily aws ssm describe-patch-baselines`

Lists the patch baselines in your Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-patch-group-state`

Returns high-level aggregated patch compliance state information for a patch group.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --patch-group, --profile, --query, --region, --version

## `coily aws ssm describe-patch-groups`

Lists all patch groups that have been registered with patch baselines.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm describe-patch-properties`

Lists the properties of available patches organized by product, product family, classification, severity, and other properties of available patches.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --operating-system, --output, --page-size, --patch-set, --profile, --property, --query, --region, --starting-token, --version

## `coily aws ssm describe-sessions`

Retrieves a list of all active sessions (both connected and disconnected) or terminated sessions from the past 30 days.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --state, --version

## `coily aws ssm disassociate-ops-item-related-item`

Deletes the association between an OpsItem and a related item.

Flags: --association-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --ops-item-id, --output, --profile, --query, --region, --version

## `coily aws ssm get-automation-execution`

Get detailed information about a particular Automation execution.

Flags: --automation-execution-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm get-calendar-state`

Gets the state of a Amazon Web Services Systems Manager change calendar at the current time or a specified time.

Flags: --at-time, --ca-bundle, --calendar-names, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm get-command-invocation`

Returns detailed information about command execution for an invocation or plugin.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --command-id, --debug, --endpoint-url, --generate-cli-skeleton, --instance-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --plugin-name, --profile, --query, --region, --version

## `coily aws ssm get-connection-status`

Retrieves the Session Manager connection status for a managed node to determine whether it is running and ready to receive Session Manager connections.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --target, --version

## `coily aws ssm get-default-patch-baseline`

Retrieves the default patch baseline.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --operating-system, --output, --profile, --query, --region, --version

## `coily aws ssm get-document`

Gets the contents of the specified Amazon Web Services Systems Manager document (SSM document).

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --document-format, --document-version, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --version-name

## `coily aws ssm get-execution-preview`

Initiates the process of retrieving an existing preview that shows the effects that running a specified Automation runbook would have on the targeted resources.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --execution-preview-id, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm get-inventory`

Query inventory information.

Flags: --aggregators, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --result-attributes, --starting-token, --version

## `coily aws ssm get-inventory-schema`

Return a list of inventory type names for the account, or return a list of attribute names for a specific Inventory item type.

Flags: --aggregator, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --sub-type, --type-name, --version

## `coily aws ssm get-maintenance-window`

Retrieves a maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --window-id

## `coily aws ssm get-maintenance-window-execution`

Retrieves details about a specific a maintenance window execution.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --window-execution-id

## `coily aws ssm get-maintenance-window-execution-task`

Retrieves the details about a specific task run as part of a maintenance window execution.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --task-id, --version, --window-execution-id

## `coily aws ssm get-maintenance-window-task`

Retrieves the details of a maintenance window task.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --window-id, --window-task-id

## `coily aws ssm get-ops-item`

Get information about an OpsItem by using the ID.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --ops-item-arn, --ops-item-id, --output, --profile, --query, --region, --version

## `coily aws ssm get-ops-metadata`

View operational metadata related to an application in Application Manager.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-results, --next-token, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --ops-metadata-arn, --output, --profile, --query, --region, --version

## `coily aws ssm get-ops-summary`

View a summary of operations metadata (OpsData) based on specified filters and aggregators.

Flags: --aggregators, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --result-attributes, --starting-token, --sync-name, --version

## `coily aws ssm get-parameter`

Get information about a single parameter by specifying the parameter name.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --with-decryption

## `coily aws ssm get-parameter-history`

Retrieves the history of all changes to a parameter.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version, --with-decryption

## `coily aws ssm get-parameters`

Get information about one or more parameters by specifying multiple parameter names.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --names, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version, --with-decryption

## `coily aws ssm get-parameters-by-path`

Retrieve information about one or more parameters under a specified level in a hierarchy.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --parameter-filters, --path, --profile, --query, --recursive, --region, --starting-token, --version, --with-decryption

## `coily aws ssm get-patch-baseline`

Retrieves information about a patch baseline.

Flags: --baseline-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm get-patch-baseline-for-patch-group`

Retrieves the patch baseline that should be used for the specified patch group.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --operating-system, --output, --patch-group, --profile, --query, --region, --version

## `coily aws ssm get-resource-policies`

Returns an array of the Policy object.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --resource-arn, --starting-token, --version

## `coily aws ssm get-service-setting`

ServiceSetting is an account-level setting for an Amazon Web Services service.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --setting-id, --version

## `coily aws ssm label-parameter-version`

A parameter label is a user-defined alias to help you manage different versions of a parameter.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --labels, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --parameter-version, --profile, --query, --region, --version

## `coily aws ssm list-association-versions`

Retrieves all versions of an association for a specific association ID.

Flags: --association-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-associations`

Returns all State Manager associations in the current Amazon Web Services account and Amazon Web Services Region.

Flags: --association-filter-list, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-command-invocations`

An invocation is copy of a command sent to a specific managed node.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --command-id, --debug, --details, --endpoint-url, --filters, --generate-cli-skeleton, --instance-id, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-commands`

Lists the commands requested by users of the Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --command-id, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --instance-id, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-compliance-items`

For a specified resource ID, this API operation returns a list of compliance statuses for different resource types.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --resource-ids, --resource-types, --starting-token, --version

## `coily aws ssm list-compliance-summaries`

Returns a summary count of compliant and non-compliant resources for a compliance type.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-document-metadata-history`

Information about approval reviews for a version of a change template in Change Manager.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --document-version, --endpoint-url, --generate-cli-skeleton, --max-results, --metadata, --name, --next-token, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm list-document-versions`

List all versions for a document.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-documents`

Returns all Systems Manager (SSM) documents in the current Amazon Web Services account and Amazon Web Services Region.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --document-filter-list, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-inventory-entries`

A list of inventory items returned by the request.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --instance-id, --max-results, --next-token, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --type-name, --version

## `coily aws ssm list-nodes`

Takes in filters and returns a list of managed nodes matching the filter criteria.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --sync-name, --version

## `coily aws ssm list-nodes-summary`

Generates a summary of managed instance/node metadata based on the filters and aggregators you specify.

Flags: --aggregators, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --sync-name, --version

## `coily aws ssm list-ops-item-events`

Returns a list of all OpsItem events in the current Amazon Web Services Region and Amazon Web Services account.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-ops-item-related-items`

Lists all related-item resources associated with a Systems Manager OpsCenter OpsItem.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --ops-item-id, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-ops-metadata`

Amazon Web Services Systems Manager calls this API operation when displaying all Application Manager OpsMetadata objects or blobs.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-resource-compliance-summaries`

Returns a resource-level summary count.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --filters, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --version

## `coily aws ssm list-resource-data-sync`

Lists your resource data sync configurations.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --max-items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --page-size, --profile, --query, --region, --starting-token, --sync-type, --version

## `coily aws ssm list-tags-for-resource`

Returns a list of the tags assigned to the specified resource.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --resource-id, --resource-type, --version

## `coily aws ssm modify-document-permission`

Shares a Amazon Web Services Systems Manager document (SSM document)publicly or privately.

Flags: --account-ids-to-add, --account-ids-to-remove, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --permission-type, --profile, --query, --region, --shared-document-version, --version

## `coily aws ssm put-compliance-items`

Registers a compliance type and other compliance details on a designated resource.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --compliance-type, --debug, --endpoint-url, --execution-summary, --generate-cli-skeleton, --item-content-hash, --items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --resource-id, --resource-type, --upload-type, --version

## `coily aws ssm put-inventory`

Bulk update custom inventory items on one or more managed nodes.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --instance-id, --items, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm put-parameter`

Create or update a parameter in Parameter Store.

Flags: --allowed-pattern, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --data-type, --debug, --description, --endpoint-url, --generate-cli-skeleton, --key-id, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --overwrite, --policies, --profile, --query, --region, --tags, --tier, --type, --value, --version

## `coily aws ssm put-resource-policy`

Creates or updates a Systems Manager resource policy.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --policy, --policy-hash, --policy-id, --profile, --query, --region, --resource-arn, --version

## `coily aws ssm register-default-patch-baseline`

Defines the default patch baseline for the relevant operating system.

Flags: --baseline-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm register-patch-baseline-for-patch-group`

Registers a patch baseline for a patch group.

Flags: --baseline-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --patch-group, --profile, --query, --region, --version

## `coily aws ssm register-target-with-maintenance-window`

Registers a target with a maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --client-token, --color, --debug, --description, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --owner-information, --profile, --query, --region, --resource-type, --targets, --version, --window-id

## `coily aws ssm register-task-with-maintenance-window`

Adds a new task to a maintenance window.

Flags: --alarm-configuration, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --client-token, --color, --cutoff-behavior, --debug, --description, --endpoint-url, --generate-cli-skeleton, --logging-info, --max-concurrency, --max-errors, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --priority, --profile, --query, --region, --service-role-arn, --targets, --task-arn, --task-invocation-parameters, --task-parameters, --task-type, --version, --window-id

## `coily aws ssm remove-tags-from-resource`

Removes tag keys from the specified resource.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --resource-id, --resource-type, --tag-keys, --version

## `coily aws ssm reset-service-setting`

ServiceSetting is an account-level setting for an Amazon Web Services service.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --setting-id, --version

## `coily aws ssm resume-session`

Reconnects a session to a managed node after it has been disconnected.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --session-id, --version

## `coily aws ssm send-automation-signal`

Sends a signal to an Automation execution to change the current behavior or status of the execution.

Flags: --automation-execution-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --payload, --profile, --query, --region, --signal-type, --version

## `coily aws ssm send-command`

Runs commands on one or more managed nodes.

Flags: --alarm-configuration, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --cloud-watch-output-config, --color, --comment, --debug, --document-hash, --document-hash-type, --document-name, --document-version, --endpoint-url, --generate-cli-skeleton, --instance-ids, --max-concurrency, --max-errors, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --notification-config, --output, --output-s3-bucket-name, --output-s3-key-prefix, --output-s3-region, --parameters, --profile, --query, --region, --service-role-arn, --targets, --timeout-seconds, --version

## `coily aws ssm start-associations-once`

Runs an association immediately and only one time.

Flags: --association-ids, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm start-automation-execution`

Initiates execution of an Automation runbook.

Flags: --alarm-configuration, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --client-token, --color, --debug, --document-name, --document-version, --endpoint-url, --generate-cli-skeleton, --max-concurrency, --max-errors, --mode, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --parameters, --profile, --query, --region, --tags, --target-locations, --target-locations-url, --target-maps, --target-parameter-name, --targets, --version

## `coily aws ssm start-change-request-execution`

Creates a change request for Change Manager.

Flags: --auto-approve, --ca-bundle, --change-details, --change-request-name, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --client-token, --color, --debug, --document-name, --document-version, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --parameters, --profile, --query, --region, --runbooks, --scheduled-end-time, --scheduled-time, --tags, --version

## `coily aws ssm start-execution-preview`

Initiates the process of creating a preview showing the effects that running a specified Automation runbook would have on the targeted resources.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --document-name, --document-version, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm start-session`

Initiates a connection to a target (for example, a managed node) for a Session Manager session.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --document-name, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --parameters, --profile, --query, --reason, --region, --target, --version

## `coily aws ssm stop-automation-execution`

Stop an Automation that is currently running.

Flags: --automation-execution-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --type, --version

## `coily aws ssm terminate-session`

Permanently ends a session and closes the data connection between the Session Manager client and SSM Agent on the managed node.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --session-id, --version

## `coily aws ssm unlabel-parameter-version`

Remove a label or labels from a parameter.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --labels, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --parameter-version, --profile, --query, --region, --version

## `coily aws ssm update-association`

Updates an association.

Flags: --alarm-configuration, --association-id, --association-name, --association-version, --automation-target-parameter-name, --ca-bundle, --calendar-names, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --compliance-severity, --debug, --document-version, --duration, --endpoint-url, --generate-cli-skeleton, --max-concurrency, --max-errors, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --output-location, --parameters, --profile, --query, --region, --schedule-expression, --schedule-offset, --sync-compliance, --target-locations, --target-maps, --targets, --version

## `coily aws ssm update-association-status`

Updates the status of the Amazon Web Services Systems Manager document (SSM document) associated with the specified managed node.

Flags: --association-status, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --instance-id, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm update-document`

Updates one or more values for an SSM document.

Flags: --attachments, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --content, --debug, --display-name, --document-format, --document-version, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --target-type, --version, --version-name

## `coily aws ssm update-document-default-version`

Set the default version of a document.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --document-version, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm update-document-metadata`

Updates information related to approval reviews for a specific version of a change template in Change Manager.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --document-reviews, --document-version, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm update-maintenance-window`

Updates an existing maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --cutoff, --debug, --description, --duration, --enabled, --end-date, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --replace, --schedule, --schedule-offset, --schedule-timezone, --start-date, --version, --window-id

## `coily aws ssm update-maintenance-window-target`

Modifies the target of an existing maintenance window.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --description, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --owner-information, --profile, --query, --region, --replace, --targets, --version, --window-id, --window-target-id

## `coily aws ssm update-maintenance-window-task`

Modifies a task assigned to a maintenance window.

Flags: --alarm-configuration, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --cutoff-behavior, --debug, --description, --endpoint-url, --generate-cli-skeleton, --logging-info, --max-concurrency, --max-errors, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --priority, --profile, --query, --region, --replace, --service-role-arn, --targets, --task-arn, --task-invocation-parameters, --task-parameters, --version, --window-id, --window-task-id

## `coily aws ssm update-managed-instance-role`

Changes the Identity and Access Management (IAM) role that is assigned to the on-premises server, edge device, or virtual machines (VM).

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --iam-role, --instance-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws ssm update-ops-item`

Edit or change an OpsItem.

Flags: --actual-end-time, --actual-start-time, --ca-bundle, --category, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --description, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --notifications, --operational-data, --operational-data-to-delete, --ops-item-arn, --ops-item-id, --output, --planned-end-time, --planned-start-time, --priority, --profile, --query, --region, --related-ops-items, --severity, --status, --title, --version

## `coily aws ssm update-ops-metadata`

Amazon Web Services Systems Manager calls this API operation when you edit OpsMetadata in Application Manager.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --keys-to-delete, --metadata-to-update, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --ops-metadata-arn, --output, --profile, --query, --region, --version

## `coily aws ssm update-patch-baseline`

Modifies an existing patch baseline.

Flags: --approval-rules, --approved-patches, --approved-patches-compliance-level, --baseline-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-read-timeout, --color, --debug, --endpoint-url, --global-filters, --name, --no-approved-patches-enable-non-security, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --rejected-patches, --rejected-patches-action, --version

## `coily aws ssm update-resource-data-sync`

Update a resource data sync.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --sync-name, --sync-source, --sync-type, --version

## `coily aws ssm update-service-setting`

ServiceSetting is an account-level setting for an Amazon Web Services service.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --setting-id, --setting-value, --version

## `coily aws ssm wait command-executed`

Wait until JMESPath query Status returns Success when polling with get-command-invocation.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --command-id, --debug, --endpoint-url, --generate-cli-skeleton, --instance-id, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --plugin-name, --profile, --query, --region, --version

## `coily aws sts assume-role`

Returns a set of temporary security credentials that you can use to access Amazon Web Services resources.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --duration-seconds, --endpoint-url, --external-id, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --policy, --policy-arns, --profile, --provided-contexts, --query, --region, --role-arn, --role-session-name, --serial-number, --source-identity, --tags, --token-code, --transitive-tag-keys, --version

## `coily aws sts assume-role-with-saml`

Returns a set of temporary security credentials for users who have been authenticated via a SAML authentication response.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --duration-seconds, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --policy, --policy-arns, --principal-arn, --profile, --query, --region, --role-arn, --saml-assertion, --version

## `coily aws sts assume-role-with-web-identity`

Returns a set of temporary security credentials for users who have been authenticated in a mobile or web application with a web identity provider.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --duration-seconds, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --policy, --policy-arns, --profile, --provider-id, --query, --region, --role-arn, --role-session-name, --version, --web-identity-token

## `coily aws sts assume-root`

Returns a set of short term credentials you can use to perform privileged tasks on a member account in your organization.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --duration-seconds, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --target-principal, --task-policy-arn, --version

## `coily aws sts decode-authorization-message`

Decodes additional information about the authorization status of a request from an encoded message returned in response to an Amazon Web Services request.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --encoded-message, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws sts get-access-key-info`

Returns the account identifier for the specified access key ID.

Flags: --access-key-id, --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws sts get-caller-identity`

Returns details about the IAM user or role whose credentials are used to call the operation.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --version

## `coily aws sts get-federation-token`

Returns a set of temporary security credentials (consisting of an access key ID, a secret access key, and a security token) for a user.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --duration-seconds, --endpoint-url, --generate-cli-skeleton, --name, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --policy, --policy-arns, --profile, --query, --region, --tags, --version

## `coily aws sts get-session-token`

Returns a set of temporary credentials for an Amazon Web Services account or IAM user.

Flags: --ca-bundle, --cli-auto-prompt, --cli-binary-format, --cli-connect-timeout, --cli-input-json, --cli-read-timeout, --color, --debug, --duration-seconds, --endpoint-url, --generate-cli-skeleton, --no-cli-auto-prompt, --no-cli-pager, --no-paginate, --no-sign-request, --no-verify-ssl, --output, --profile, --query, --region, --serial-number, --token-code, --version

## `coily core-keeper restart`

Restart the core-keeper-server unit.

## `coily core-keeper start`

Start the core-keeper-server unit.

## `coily core-keeper status`

Print systemctl status core-keeper-server.

## `coily core-keeper stop`

Stop the core-keeper-server unit.

## `coily core-keeper tail`

Tail core-keeper-server journal logs (journalctl -u core-keeper-server -f).

Flags: --follow, --lines

## `coily docker build`

Start a build

Flags: --add-host, --allow, --annotation, --attest, --build-arg, --build-context, --builder, --cache-from, --cache-to, --call, --cgroup-parent, --check, --debug, --file, --iidfile, --label, --load, --metadata-file, --network, --no-cache, --no-cache-filter, --output, --platform, --progress, --provenance, --pull, --push, --quiet, --sbom, --secret, --shm-size, --ssh, --tag, --target, --ulimit

## `coily docker commit`

Usage:  docker commit [OPTIONS] CONTAINER [REPOSITORY[:TAG]]

Flags: --author, --change, --message, --pause

## `coily docker container attach`

Usage:  docker container attach [OPTIONS] CONTAINER

Flags: --detach-keys, --no-stdin, --sig-proxy

## `coily docker container commit`

Usage:  docker container commit [OPTIONS] CONTAINER [REPOSITORY[:TAG]]

Flags: --author, --change, --message, --pause

## `coily docker container cp`

Usage:  docker container cp [OPTIONS] CONTAINER:SRC_PATH DEST_PATH|-

Flags: --archive, --follow-link, --quiet

## `coily docker container create`

Usage:  docker container create [OPTIONS] IMAGE [COMMAND] [ARG...]

Flags: --add-host, --annotation, --attach, --blkio-weight, --blkio-weight-device, --cap-add, --cap-drop, --cgroup-parent, --cgroupns, --cidfile, --cpu-count, --cpu-percent, --cpu-period, --cpu-quota, --cpu-rt-period, --cpu-rt-runtime, --cpu-shares, --cpus, --cpuset-cpus, --cpuset-mems, --device, --device-cgroup-rule, --device-read-bps, --device-read-iops, --device-write-bps, --device-write-iops, --disable-content-trust, --dns, --dns-option, --dns-search, --domainname, --entrypoint, --env, --env-file, --expose, --gpus, --group-add, --health-cmd, --health-interval, --health-retries, --health-start-interval, --health-start-period, --health-timeout, --help, --hostname, --init, --interactive, --io-maxbandwidth, --io-maxiops, --ip, --ip6, --ipc, --isolation, --kernel-memory, --label, --label-file, --link, --link-local-ip, --log-driver, --log-opt, --mac-address, --memory, --memory-reservation, --memory-swap, --memory-swappiness, --mount, --name, --network, --network-alias, --no-healthcheck, --oom-kill-disable, --oom-score-adj, --pid, --pids-limit, --platform, --privileged, --publish, --publish-all, --pull, --quiet, --read-only, --restart, --rm, --runtime, --security-opt, --shm-size, --stop-signal, --stop-timeout, --storage-opt, --sysctl, --tmpfs, --tty, --ulimit, --user, --userns, --uts, --volume, --volume-driver, --volumes-from, --workdir

## `coily docker container diff`

Usage:  docker container diff CONTAINER

## `coily docker container exec`

Usage:  docker container exec [OPTIONS] CONTAINER COMMAND [ARG...]

Flags: --detach, --detach-keys, --env, --env-file, --interactive, --privileged, --tty, --user, --workdir

## `coily docker container export`

Usage:  docker container export [OPTIONS] CONTAINER

Flags: --output

## `coily docker container inspect`

Usage:  docker container inspect [OPTIONS] CONTAINER [CONTAINER...]

Flags: --format, --size

## `coily docker container kill`

Usage:  docker container kill [OPTIONS] CONTAINER [CONTAINER...]

Flags: --signal

## `coily docker container logs`

Usage:  docker container logs [OPTIONS] CONTAINER

Flags: --details, --follow, --since, --tail, --timestamps, --until

## `coily docker container ls`

Usage:  docker container ls [OPTIONS]

Flags: --all, --filter, --format, --last, --latest, --no-trunc, --quiet, --size

## `coily docker container pause`

Usage:  docker container pause CONTAINER [CONTAINER...]

## `coily docker container port`

Usage:  docker container port CONTAINER [PRIVATE_PORT[/PROTO]]

## `coily docker container prune`

Usage:  docker container prune [OPTIONS]

Flags: --filter, --force

## `coily docker container rename`

Usage:  docker container rename CONTAINER NEW_NAME

## `coily docker container restart`

Usage:  docker container restart [OPTIONS] CONTAINER [CONTAINER...]

Flags: --signal, --timeout

## `coily docker container rm`

Usage:  docker container rm [OPTIONS] CONTAINER [CONTAINER...]

Flags: --force, --link, --volumes

## `coily docker container run`

Usage:  docker container run [OPTIONS] IMAGE [COMMAND] [ARG...]

Flags: --add-host, --annotation, --attach, --blkio-weight, --blkio-weight-device, --cap-add, --cap-drop, --cgroup-parent, --cgroupns, --cidfile, --cpu-count, --cpu-percent, --cpu-period, --cpu-quota, --cpu-rt-period, --cpu-rt-runtime, --cpu-shares, --cpus, --cpuset-cpus, --cpuset-mems, --detach, --detach-keys, --device, --device-cgroup-rule, --device-read-bps, --device-read-iops, --device-write-bps, --device-write-iops, --disable-content-trust, --dns, --dns-option, --dns-search, --domainname, --entrypoint, --env, --env-file, --expose, --gpus, --group-add, --health-cmd, --health-interval, --health-retries, --health-start-interval, --health-start-period, --health-timeout, --help, --hostname, --init, --interactive, --io-maxbandwidth, --io-maxiops, --ip, --ip6, --ipc, --isolation, --kernel-memory, --label, --label-file, --link, --link-local-ip, --log-driver, --log-opt, --mac-address, --memory, --memory-reservation, --memory-swap, --memory-swappiness, --mount, --name, --network, --network-alias, --no-healthcheck, --oom-kill-disable, --oom-score-adj, --pid, --pids-limit, --platform, --privileged, --publish, --publish-all, --pull, --quiet, --read-only, --restart, --rm, --runtime, --security-opt, --shm-size, --sig-proxy, --stop-signal, --stop-timeout, --storage-opt, --sysctl, --tmpfs, --tty, --ulimit, --user, --userns, --uts, --volume, --volume-driver, --volumes-from, --workdir

## `coily docker container start`

Usage:  docker container start [OPTIONS] CONTAINER [CONTAINER...]

Flags: --attach, --checkpoint, --checkpoint-dir, --detach-keys, --interactive

## `coily docker container stats`

Usage:  docker container stats [OPTIONS] [CONTAINER...]

Flags: --all, --format, --no-stream, --no-trunc

## `coily docker container stop`

Usage:  docker container stop [OPTIONS] CONTAINER [CONTAINER...]

Flags: --signal, --timeout

## `coily docker container top`

Usage:  docker container top CONTAINER [ps OPTIONS]

## `coily docker container unpause`

Usage:  docker container unpause CONTAINER [CONTAINER...]

## `coily docker container update`

Usage:  docker container update [OPTIONS] CONTAINER [CONTAINER...]

Flags: --blkio-weight, --cpu-period, --cpu-quota, --cpu-rt-period, --cpu-rt-runtime, --cpu-shares, --cpus, --cpuset-cpus, --cpuset-mems, --memory, --memory-reservation, --memory-swap, --pids-limit, --restart

## `coily docker container wait`

Usage:  docker container wait CONTAINER [CONTAINER...]

## `coily docker cp`

Usage:  docker cp [OPTIONS] CONTAINER:SRC_PATH DEST_PATH|-

Flags: --archive, --follow-link, --quiet

## `coily docker exec`

Usage:  docker exec [OPTIONS] CONTAINER COMMAND [ARG...]

Flags: --detach, --detach-keys, --env, --env-file, --interactive, --privileged, --tty, --user, --workdir

## `coily docker image build`

Start a build

Flags: --add-host, --allow, --annotation, --attest, --build-arg, --build-context, --builder, --cache-from, --cache-to, --call, --cgroup-parent, --check, --debug, --file, --iidfile, --label, --load, --metadata-file, --network, --no-cache, --no-cache-filter, --output, --platform, --progress, --provenance, --pull, --push, --quiet, --sbom, --secret, --shm-size, --ssh, --tag, --target, --ulimit

## `coily docker image history`

Usage:  docker image history [OPTIONS] IMAGE

Flags: --format, --human, --no-trunc, --platform, --quiet

## `coily docker image import`

Usage:  docker image import [OPTIONS] file|URL|- [REPOSITORY[:TAG]]

Flags: --change, --message, --platform

## `coily docker image inspect`

Usage:  docker image inspect [OPTIONS] IMAGE [IMAGE...]

Flags: --format

## `coily docker image load`

Usage:  docker image load [OPTIONS]

Flags: --input, --platform, --quiet

## `coily docker image ls`

Usage:  docker image ls [OPTIONS] [REPOSITORY[:TAG]]

Flags: --all, --digests, --filter, --format, --no-trunc, --quiet, --tree

## `coily docker image prune`

Usage:  docker image prune [OPTIONS]

Flags: --all, --filter, --force

## `coily docker image pull`

Usage:  docker image pull [OPTIONS] NAME[:TAG|@DIGEST]

Flags: --all-tags, --disable-content-trust, --platform, --quiet

## `coily docker image push`

Usage:  docker image push [OPTIONS] NAME[:TAG]

Flags: --all-tags, --disable-content-trust, --platform, --quiet

## `coily docker image rm`

Usage:  docker image rm [OPTIONS] IMAGE [IMAGE...]

Flags: --force, --no-prune

## `coily docker image save`

Usage:  docker image save [OPTIONS] IMAGE [IMAGE...]

Flags: --output, --platform

## `coily docker image tag`

Usage:  docker image tag SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]

## `coily docker images`

Usage:  docker images [OPTIONS] [REPOSITORY[:TAG]]

Flags: --all, --digests, --filter, --format, --no-trunc, --quiet, --tree

## `coily docker info`

Usage:  docker info [OPTIONS]

Flags: --format

## `coily docker inspect`

Usage:  docker inspect [OPTIONS] NAME|ID [NAME|ID...]

Flags: --format, --size, --type

## `coily docker kill`

Usage:  docker kill [OPTIONS] CONTAINER [CONTAINER...]

Flags: --signal

## `coily docker login`

Usage:  docker login [OPTIONS] [SERVER]

Flags: --password, --password-stdin, --username

## `coily docker logout`

Usage:  docker logout [SERVER]

## `coily docker logs`

Usage:  docker logs [OPTIONS] CONTAINER

Flags: --details, --follow, --since, --tail, --timestamps, --until

## `coily docker manifest annotate`

Usage:  docker manifest annotate [OPTIONS] MANIFEST_LIST MANIFEST

Flags: --arch, --os, --os-features, --os-version, --variant

## `coily docker manifest create`

Usage:  docker manifest create MANIFEST_LIST MANIFEST [MANIFEST...]

Flags: --amend, --insecure

## `coily docker manifest inspect`

Usage:  docker manifest inspect [OPTIONS] [MANIFEST_LIST] MANIFEST

Flags: --insecure, --verbose

## `coily docker manifest push`

Usage:  docker manifest push [OPTIONS] MANIFEST_LIST

Flags: --insecure, --purge

## `coily docker manifest rm`

Usage:  docker manifest rm MANIFEST_LIST [MANIFEST_LIST...]

## `coily docker network connect`

Usage:  docker network connect [OPTIONS] NETWORK CONTAINER

Flags: --alias, --driver-opt, --gw-priority, --ip, --ip6, --link, --link-local-ip

## `coily docker network create`

Usage:  docker network create [OPTIONS] NETWORK

Flags: --attachable, --aux-address, --config-from, --config-only, --driver, --gateway, --ingress, --internal, --ip-range, --ipam-driver, --ipam-opt, --ipv4, --ipv6, --label, --opt, --scope, --subnet

## `coily docker network disconnect`

Usage:  docker network disconnect [OPTIONS] NETWORK CONTAINER

Flags: --force

## `coily docker network inspect`

Usage:  docker network inspect [OPTIONS] NETWORK [NETWORK...]

Flags: --format, --verbose

## `coily docker network ls`

Usage:  docker network ls [OPTIONS]

Flags: --filter, --format, --no-trunc, --quiet

## `coily docker network prune`

Usage:  docker network prune [OPTIONS]

Flags: --filter, --force

## `coily docker network rm`

Usage:  docker network rm NETWORK [NETWORK...]

Flags: --force

## `coily docker ps`

Usage:  docker ps [OPTIONS]

Flags: --all, --filter, --format, --last, --latest, --no-trunc, --quiet, --size

## `coily docker pull`

Usage:  docker pull [OPTIONS] NAME[:TAG|@DIGEST]

Flags: --all-tags, --disable-content-trust, --platform, --quiet

## `coily docker push`

Usage:  docker push [OPTIONS] NAME[:TAG]

Flags: --all-tags, --disable-content-trust, --platform, --quiet

## `coily docker restart`

Usage:  docker restart [OPTIONS] CONTAINER [CONTAINER...]

Flags: --signal, --timeout

## `coily docker rm`

Usage:  docker rm [OPTIONS] CONTAINER [CONTAINER...]

Flags: --force, --link, --volumes

## `coily docker rmi`

Usage:  docker rmi [OPTIONS] IMAGE [IMAGE...]

Flags: --force, --no-prune

## `coily docker run`

Usage:  docker run [OPTIONS] IMAGE [COMMAND] [ARG...]

Flags: --add-host, --annotation, --attach, --blkio-weight, --blkio-weight-device, --cap-add, --cap-drop, --cgroup-parent, --cgroupns, --cidfile, --cpu-count, --cpu-percent, --cpu-period, --cpu-quota, --cpu-rt-period, --cpu-rt-runtime, --cpu-shares, --cpus, --cpuset-cpus, --cpuset-mems, --detach, --detach-keys, --device, --device-cgroup-rule, --device-read-bps, --device-read-iops, --device-write-bps, --device-write-iops, --disable-content-trust, --dns, --dns-option, --dns-search, --domainname, --entrypoint, --env, --env-file, --expose, --gpus, --group-add, --health-cmd, --health-interval, --health-retries, --health-start-interval, --health-start-period, --health-timeout, --help, --hostname, --init, --interactive, --io-maxbandwidth, --io-maxiops, --ip, --ip6, --ipc, --isolation, --kernel-memory, --label, --label-file, --link, --link-local-ip, --log-driver, --log-opt, --mac-address, --memory, --memory-reservation, --memory-swap, --memory-swappiness, --mount, --name, --network, --network-alias, --no-healthcheck, --oom-kill-disable, --oom-score-adj, --pid, --pids-limit, --platform, --privileged, --publish, --publish-all, --pull, --quiet, --read-only, --restart, --rm, --runtime, --security-opt, --shm-size, --sig-proxy, --stop-signal, --stop-timeout, --storage-opt, --sysctl, --tmpfs, --tty, --ulimit, --user, --userns, --uts, --volume, --volume-driver, --volumes-from, --workdir

## `coily docker search`

Usage:  docker search [OPTIONS] TERM

Flags: --filter, --format, --limit, --no-trunc

## `coily docker start`

Usage:  docker start [OPTIONS] CONTAINER [CONTAINER...]

Flags: --attach, --checkpoint, --checkpoint-dir, --detach-keys, --interactive

## `coily docker stop`

Usage:  docker stop [OPTIONS] CONTAINER [CONTAINER...]

Flags: --signal, --timeout

## `coily docker system df`

Usage:  docker system df [OPTIONS]

Flags: --format, --verbose

## `coily docker system events`

Usage:  docker system events [OPTIONS]

Flags: --filter, --format, --since, --until

## `coily docker system info`

Usage:  docker system info [OPTIONS]

Flags: --format

## `coily docker system prune`

Usage:  docker system prune [OPTIONS]

Flags: --all, --filter, --force, --volumes

## `coily docker tag`

Usage:  docker tag SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]

## `coily docker volume create`

Usage:  docker volume create [OPTIONS] [VOLUME]

Flags: --availability, --driver, --group, --label, --limit-bytes, --opt, --required-bytes, --scope, --secret, --sharing, --topology-preferred, --topology-required, --type

## `coily docker volume inspect`

Usage:  docker volume inspect [OPTIONS] VOLUME [VOLUME...]

Flags: --format

## `coily docker volume ls`

Usage:  docker volume ls [OPTIONS]

Flags: --cluster, --filter, --format, --quiet

## `coily docker volume prune`

Usage:  docker volume prune [OPTIONS]

Flags: --all, --filter, --force

## `coily docker volume rm`

Usage:  docker volume rm [OPTIONS] VOLUME [VOLUME...]

Flags: --force

## `coily docker volume update`

Usage:  docker volume update [OPTIONS] [VOLUME]

Flags: --availability

## `coily eco mod push`

scp a .zip to <server_dir> on kai-server and unzip -o it.

Flags: --keep-remote, --server-dir, --src

## `coily eco restart`

Restart the eco-server systemd unit.

## `coily eco start`

Start the eco-server systemd unit.

## `coily eco status`

Print systemctl status eco-server.

## `coily eco stop`

Stop the eco-server systemd unit.

## `coily eco tail`

Tail eco-server journal logs (journalctl -u eco-server -f).

Flags: --follow, --lines

## `coily eco world get-seed`

Print the current Seed from Configs/WorldGenerator.eco.

Flags: --configs-dir

## `coily eco world randomize`

Generate a random seed and write it to Configs/WorldGenerator.eco.

Flags: --configs-dir

## `coily eco world set-seed`

Write a specific Seed into Configs/WorldGenerator.eco.

Flags: --configs-dir, --seed

## `coily eco world snapshot`

Copy Configs/WorldGenerator.eco to --target.

Flags: --configs-dir, --target

## `coily gh api`

Makes an authenticated HTTP request to the GitHub API and prints the response.

Flags: --cache, --field, --header, --help, --hostname, --include, --input, --jq, --method, --paginate, --preview, --raw-field, --silent, --slurp, --template, --verbose

## `coily gh issue close`

Close issue

Flags: --comment, --help, --reason, --repo

## `coily gh issue comment`

Add a comment to a GitHub issue.

Flags: --body, --body-file, --edit-last, --editor, --help, --repo, --web

## `coily gh issue create`

Create an issue on GitHub.

Flags: --assignee, --body, --body-file, --editor, --help, --label, --milestone, --project, --recover, --repo, --template, --title, --web

## `coily gh issue delete`

Delete issue

Flags: --help, --repo, --yes

## `coily gh issue develop`

Manage linked branches for an issue.

Flags: --base, --branch-repo, --checkout, --help, --list, --name, --repo

## `coily gh issue edit`

Edit one or more issues within the same repository.

Flags: --add-assignee, --add-label, --add-project, --body, --body-file, --help, --milestone, --remove-assignee, --remove-label, --remove-milestone, --remove-project, --repo, --title

## `coily gh issue list`

List issues in a GitHub repository.

Flags: --app, --assignee, --author, --help, --jq, --json, --label, --limit, --mention, --milestone, --repo, --search, --state, --template, --web

## `coily gh issue lock`

Lock issue conversation

Flags: --help, --reason, --repo

## `coily gh issue pin`

Pin an issue to a repository.

Flags: --help, --repo

## `coily gh issue reopen`

Reopen issue

Flags: --comment, --help, --repo

## `coily gh issue status`

Show status of relevant issues

Flags: --help, --jq, --json, --repo, --template

## `coily gh issue transfer`

Transfer issue to another repository

Flags: --help, --repo

## `coily gh issue unlock`

Unlock issue conversation

Flags: --help, --repo

## `coily gh issue unpin`

Unpin an issue from a repository.

Flags: --help, --repo

## `coily gh issue view`

Display the title, body, and other information about an issue.

Flags: --comments, --help, --jq, --json, --repo, --template, --web

## `coily gh pr checkout`

Check out a pull request in git

Flags: --branch, --detach, --force, --help, --recurse-submodules, --repo

## `coily gh pr checks`

Show CI status for a single pull request.

Flags: --fail-fast, --help, --jq, --json, --repo, --required, --template, --watch, --web

## `coily gh pr close`

Close a pull request

Flags: --comment, --delete-branch, --help, --repo

## `coily gh pr comment`

Add a comment to a GitHub pull request.

Flags: --body, --body-file, --edit-last, --editor, --help, --repo, --web

## `coily gh pr create`

Create a pull request on GitHub.

Flags: --assignee, --base, --body, --body-file, --draft, --dry-run, --editor, --fill, --fill-first, --fill-verbose, --head, --help, --label, --milestone, --no-maintainer-edit, --project, --recover, --repo, --reviewer, --template, --title, --web

## `coily gh pr diff`

View changes in a pull request.

Flags: --color, --help, --name-only, --patch, --repo, --web

## `coily gh pr edit`

Edit a pull request.

Flags: --add-assignee, --add-label, --add-project, --add-reviewer, --base, --body, --body-file, --help, --milestone, --remove-assignee, --remove-label, --remove-milestone, --remove-project, --remove-reviewer, --repo, --title

## `coily gh pr list`

List pull requests in a GitHub repository.

Flags: --app, --assignee, --author, --base, --draft, --head, --help, --jq, --json, --label, --limit, --repo, --search, --state, --template, --web

## `coily gh pr lock`

Lock pull request conversation

Flags: --help, --reason, --repo

## `coily gh pr merge`

Merge a pull request on GitHub.

Flags: --admin, --author-email, --auto, --body, --body-file, --delete-branch, --disable-auto, --help, --match-head-commit, --merge, --rebase, --repo, --squash, --subject

## `coily gh pr ready`

Mark a pull request as ready for review.

Flags: --help, --repo, --undo

## `coily gh pr reopen`

Reopen a pull request

Flags: --comment, --help, --repo

## `coily gh pr review`

Add a review to a pull request.

Flags: --approve, --body, --body-file, --comment, --help, --repo, --request-changes

## `coily gh pr status`

Show status of relevant pull requests

Flags: --conflict-status, --help, --jq, --json, --repo, --template

## `coily gh pr unlock`

Unlock pull request conversation

Flags: --help, --repo

## `coily gh pr view`

Display the title, body, and other information about a pull request.

Flags: --comments, --help, --jq, --json, --repo, --template, --web

## `coily gh release create`

Create a new GitHub Release for a repository.

Flags: --discussion-category, --draft, --generate-notes, --help, --latest, --notes, --notes-file, --notes-from-tag, --notes-start-tag, --prerelease, --repo, --target, --title, --verify-tag

## `coily gh release delete`

Delete a release

Flags: --cleanup-tag, --help, --repo, --yes

## `coily gh release download`

Download assets from a GitHub release.

Flags: --archive, --clobber, --dir, --help, --output, --pattern, --repo, --skip-existing

## `coily gh release edit`

Edit a release

Flags: --discussion-category, --draft, --help, --latest, --notes, --notes-file, --prerelease, --repo, --tag, --target, --title, --verify-tag

## `coily gh release list`

List releases in a repository

Flags: --exclude-drafts, --exclude-pre-releases, --help, --jq, --json, --limit, --order, --repo, --template

## `coily gh release upload`

Upload asset files to a GitHub Release.

Flags: --clobber, --help, --repo

## `coily gh release view`

View information about a GitHub Release.

Flags: --help, --jq, --json, --repo, --template, --web

## `coily gh repo archive`

Archive a GitHub repository.

Flags: --help, --yes

## `coily gh repo autolink create`

Create a new autolink reference for a repository.

Flags: --help, --numeric, --repo

## `coily gh repo autolink list`

Gets all autolink references that are configured for a repository.

Flags: --help, --jq, --json, --repo, --template, --web

## `coily gh repo autolink view`

View an autolink reference for a repository.

Flags: --help, --jq, --json, --repo, --template

## `coily gh repo clone`

Clone a GitHub repository locally.

Flags: --help, --upstream-remote-name

## `coily gh repo create`

Create a new GitHub repository.

Flags: --add-readme, --clone, --description, --disable-issues, --disable-wiki, --gitignore, --help, --homepage, --include-all-branches, --internal, --license, --private, --public, --push, --remote, --source, --team, --template

## `coily gh repo delete`

Delete a GitHub repository.

Flags: --help, --yes

## `coily gh repo deploy-key add`

Add a deploy key to a GitHub repository.

Flags: --allow-write, --help, --repo, --title

## `coily gh repo deploy-key delete`

Delete a deploy key from a GitHub repository

Flags: --help, --repo

## `coily gh repo deploy-key list`

List deploy keys in a GitHub repository

Flags: --help, --jq, --json, --repo, --template

## `coily gh repo edit`

Edit repository settings.

Flags: --accept-visibility-change-consequences, --add-topic, --allow-forking, --allow-update-branch, --default-branch, --delete-branch-on-merge, --description, --enable-advanced-security, --enable-auto-merge, --enable-discussions, --enable-issues, --enable-merge-commit, --enable-projects, --enable-rebase-merge, --enable-secret-scanning, --enable-secret-scanning-push-protection, --enable-squash-merge, --enable-wiki, --help, --homepage, --remove-topic, --template, --visibility

## `coily gh repo fork`

Create a fork of a repository.

Flags: --clone, --default-branch-only, --fork-name, --help, --org, --remote, --remote-name

## `coily gh repo gitignore list`

List available repository gitignore templates

Flags: --help

## `coily gh repo gitignore view`

View an available repository '.gitignore' template.

Flags: --help

## `coily gh repo license list`

List common repository licenses.

Flags: --help

## `coily gh repo license view`

View a specific repository license by license key or SPDX ID.

Flags: --help, --web

## `coily gh repo list`

List repositories owned by a user or organization.

Flags: --archived, --fork, --help, --jq, --json, --language, --limit, --no-archived, --source, --template, --topic, --visibility

## `coily gh repo rename`

Rename a GitHub repository.

Flags: --help, --repo, --yes

## `coily gh repo sync`

Sync destination repository from source repository.

Flags: --branch, --force, --help, --source

## `coily gh repo unarchive`

Unarchive a GitHub repository.

Flags: --help, --yes

## `coily gh repo view`

Display the description and the README of a GitHub repository.

Flags: --branch, --help, --jq, --json, --template, --web

## `coily gh run cancel`

Cancel a workflow run

Flags: --help, --repo

## `coily gh run delete`

Delete a workflow run

Flags: --help, --repo

## `coily gh run download`

Download artifacts generated by a GitHub Actions workflow run.

Flags: --dir, --help, --name, --pattern, --repo

## `coily gh run list`

List recent workflow runs.

Flags: --all, --branch, --commit, --created, --event, --help, --jq, --json, --limit, --repo, --status, --template, --user, --workflow

## `coily gh run rerun`

Rerun an entire run, only failed jobs, or a specific job from a run.

Flags: --debug, --failed, --help, --job, --repo

## `coily gh run view`

View a summary of a workflow run.

Flags: --attempt, --exit-status, --help, --job, --jq, --json, --log, --log-failed, --repo, --template, --verbose, --web

## `coily gh run watch`

Watch a run until it completes, showing its progress.

Flags: --exit-status, --help, --interval, --repo

## `coily gh search code`

Search within code in GitHub repositories.

Flags: --extension, --filename, --help, --jq, --json, --language, --limit, --match, --owner, --repo, --size, --template, --web

## `coily gh search commits`

Search for commits on GitHub.

Flags: --author, --author-date, --author-email, --author-name, --committer, --committer-date, --committer-email, --committer-name, --hash, --help, --jq, --json, --limit, --merge, --order, --owner, --parent, --repo, --sort, --template, --tree, --visibility, --web

## `coily gh search issues`

Search for issues on GitHub.

Flags: --app, --archived, --assignee, --author, --closed, --commenter, --comments, --created, --help, --include-prs, --interactions, --involves, --jq, --json, --label, --language, --limit, --locked, --match, --mentions, --milestone, --no-assignee, --no-label, --no-milestone, --no-project, --order, --owner, --project, --reactions, --repo, --sort, --state, --team-mentions, --template, --updated, --visibility, --web

## `coily gh search prs`

Search for pull requests on GitHub.

Flags: --app, --archived, --assignee, --author, --base, --checks, --closed, --commenter, --comments, --created, --draft, --head, --help, --interactions, --involves, --jq, --json, --label, --language, --limit, --locked, --match, --mentions, --merged, --merged-at, --milestone, --no-assignee, --no-label, --no-milestone, --no-project, --order, --owner, --project, --reactions, --repo, --review, --review-requested, --reviewed-by, --sort, --state, --team-mentions, --template, --updated, --visibility, --web

## `coily gh search repos`

Search for repositories on GitHub.

Flags: --archived, --created, --followers, --forks, --good-first-issues, --help, --help-wanted-issues, --include-forks, --jq, --json, --language, --license, --limit, --match, --number-topics, --order, --owner, --size, --sort, --stars, --template, --topic, --updated, --visibility, --web

## `coily gh secret delete`

Delete a secret on one of the following levels:

Flags: --app, --env, --help, --org, --repo, --user

## `coily gh secret list`

List secrets on one of the following levels:

Flags: --app, --env, --help, --jq, --json, --org, --repo, --template, --user

## `coily gh secret set`

Set a value for a secret on one of the following levels:

Flags: --app, --body, --env, --env-file, --help, --no-store, --org, --repo, --repos, --user, --visibility

## `coily gh workflow disable`

Disable a workflow, preventing it from running or showing up when listing workflows.

Flags: --help, --repo

## `coily gh workflow enable`

Enable a workflow, allowing it to be run and show up when listing workflows.

Flags: --help, --repo

## `coily gh workflow list`

List workflow files, hiding disabled workflows by default.

Flags: --all, --help, --jq, --json, --limit, --repo, --template

## `coily gh workflow run`

Create a 'workflow_dispatch' event for a given workflow.

Flags: --field, --help, --json, --raw-field, --ref, --repo

## `coily gh workflow view`

View the summary of a workflow

Flags: --help, --ref, --repo, --web, --yaml

## `coily icarus restart`

Restart the icarus-server unit.

## `coily icarus start`

Start the icarus-server unit.

## `coily icarus status`

Print systemctl status icarus-server.

## `coily icarus stop`

Stop the icarus-server unit.

## `coily icarus tail`

Tail icarus-server journal logs (journalctl -u icarus-server -f).

Flags: --follow, --lines

## `coily install-completion`

Install shell tab-completion for coily.

Flags: --dry-run, --shell

## `coily kubectl annotate`

Update the annotations on one or more resources.

Flags: --all, --all-namespaces, --allow-missing-template-keys, --dry-run, --field-manager, --field-selector, --filename, --kustomize, --list, --local, --output, --overwrite, --recursive, --resource-version, --selector, --show-managed-fields, --template

## `coily kubectl api-resources`

Print the supported API resources on the server.

Flags: --api-group, --cached, --categories, --namespaced, --no-headers, --output, --sort-by, --verbs

## `coily kubectl api-versions`

Print the supported API versions on the server, in the form of 'group/version'.

## `coily kubectl apply edit-last-applied`

Edit the latest last-applied-configuration annotations of resources from the default editor.

Flags: --allow-missing-template-keys, --field-manager, --filename, --kustomize, --output, --recursive, --show-managed-fields, --template, --validate, --windows-line-endings

## `coily kubectl apply set-last-applied`

Set the latest last-applied-configuration annotations by setting it to match the contents of a file.

Flags: --allow-missing-template-keys, --create-annotation, --dry-run, --filename, --output, --show-managed-fields, --template

## `coily kubectl apply view-last-applied`

View the latest last-applied-configuration annotations by type/name or file.

Flags: --all, --filename, --kustomize, --output, --recursive, --selector

## `coily kubectl auth can-i`

Check whether an action is allowed.

Flags: --all-namespaces, --list, --no-headers, --quiet, --subresource

## `coily kubectl auth reconcile`

Reconciles rules for RBAC role, role binding, cluster role, and cluster role binding objects.

Flags: --allow-missing-template-keys, --dry-run, --filename, --kustomize, --output, --recursive, --remove-extra-permissions, --remove-extra-subjects, --show-managed-fields, --template

## `coily kubectl auth whoami`

Experimental: Check who you are and your attributes (groups, extra).

Flags: --allow-missing-template-keys, --output, --show-managed-fields, --template

## `coily kubectl autoscale`

Creates an autoscaler that automatically chooses and sets the number of pods that run in a Kubernetes cluster.

Flags: --allow-missing-template-keys, --cpu-percent, --dry-run, --field-manager, --filename, --kustomize, --max, --min, --name, --output, --recursive, --save-config, --show-managed-fields, --template

## `coily kubectl certificate approve`

Approve a certificate signing request.

Flags: --allow-missing-template-keys, --filename, --force, --kustomize, --output, --recursive, --show-managed-fields, --template

## `coily kubectl certificate deny`

Deny a certificate signing request.

Flags: --allow-missing-template-keys, --filename, --force, --kustomize, --output, --recursive, --show-managed-fields, --template

## `coily kubectl cluster-info dump`

Dump cluster information out suitable for debugging and diagnosing cluster problems.

Flags: --all-namespaces, --allow-missing-template-keys, --namespaces, --output, --output-directory, --pod-running-timeout, --show-managed-fields, --template

## `coily kubectl config current-context`

Display the current-context.

## `coily kubectl config delete-cluster`

Delete the specified cluster from the kubeconfig.

## `coily kubectl config delete-context`

Delete the specified context from the kubeconfig.

## `coily kubectl config delete-user`

Delete the specified user from the kubeconfig.

## `coily kubectl config get-clusters`

Display clusters defined in the kubeconfig.

## `coily kubectl config get-contexts`

Display one or many contexts from the kubeconfig file.

Flags: --no-headers, --output

## `coily kubectl config get-users`

Display users defined in the kubeconfig.

## `coily kubectl config rename-context`

Renames a context from the kubeconfig file.

## `coily kubectl config set`

Set an individual value in a kubeconfig file.

Flags: --set-raw-bytes

## `coily kubectl config set-cluster`

Set a cluster entry in kubeconfig.

Flags: --certificate-authority, --embed-certs, --insecure-skip-tls-verify, --proxy-url, --server, --tls-server-name

## `coily kubectl config set-context`

Set a context entry in kubeconfig.

Flags: --cluster, --current, --namespace, --user

## `coily kubectl config set-credentials`

Set a user entry in kubeconfig.

Flags: --auth-provider, --auth-provider-arg, --client-certificate, --client-key, --embed-certs, --exec-api-version, --exec-arg, --exec-command, --exec-env, --password, --token, --username

## `coily kubectl config unset`

Unset an individual value in a kubeconfig file.

## `coily kubectl config use-context`

Set the current-context in a kubeconfig file.

## `coily kubectl config view`

Display merged kubeconfig settings or a specified kubeconfig file.

Flags: --allow-missing-template-keys, --flatten, --merge, --minify, --output, --raw, --show-managed-fields, --template

## `coily kubectl cordon`

Mark node as unschedulable.

Flags: --dry-run, --selector

## `coily kubectl create clusterrole`

Create a cluster role.

Flags: --aggregation-rule, --allow-missing-template-keys, --dry-run, --field-manager, --non-resource-url, --output, --resource, --resource-name, --save-config, --show-managed-fields, --template, --validate, --verb

## `coily kubectl create clusterrolebinding`

Create a cluster role binding for a particular cluster role.

Flags: --allow-missing-template-keys, --clusterrole, --dry-run, --field-manager, --group, --output, --save-config, --serviceaccount, --show-managed-fields, --template, --user, --validate

## `coily kubectl create configmap`

Create a config map based on a file, directory, or specified literal value.

Flags: --allow-missing-template-keys, --append-hash, --dry-run, --field-manager, --from-env-file, --from-file, --from-literal, --output, --save-config, --show-managed-fields, --template, --validate

## `coily kubectl create cronjob`

Create a cron job with the specified name.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --image, --output, --restart, --save-config, --schedule, --show-managed-fields, --template, --validate

## `coily kubectl create deployment`

Create a deployment with the specified name.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --image, --output, --port, --replicas, --save-config, --show-managed-fields, --template, --validate

## `coily kubectl create ingress`

Create an ingress with the specified name.

Flags: --allow-missing-template-keys, --annotation, --class, --default-backend, --dry-run, --field-manager, --output, --rule, --save-config, --show-managed-fields, --template, --validate

## `coily kubectl create job`

Create a job with the specified name.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --from, --image, --output, --save-config, --show-managed-fields, --template, --validate

## `coily kubectl create namespace`

Create a namespace with the specified name.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --output, --save-config, --show-managed-fields, --template, --validate

## `coily kubectl create poddisruptionbudget`

Create a pod disruption budget with the specified name, selector, and desired minimum available pods.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --max-unavailable, --min-available, --output, --save-config, --selector, --show-managed-fields, --template, --validate

## `coily kubectl create priorityclass`

Create a priority class with the specified name, value, globalDefault and description.

Flags: --allow-missing-template-keys, --description, --dry-run, --field-manager, --global-default, --output, --preemption-policy, --save-config, --show-managed-fields, --template, --validate, --value

## `coily kubectl create quota`

Create a resource quota with the specified name, hard limits, and optional scopes.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --hard, --output, --save-config, --scopes, --show-managed-fields, --template, --validate

## `coily kubectl create role`

Create a role with single rule.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --output, --resource, --resource-name, --save-config, --show-managed-fields, --template, --validate, --verb

## `coily kubectl create rolebinding`

Create a role binding for a particular role or cluster role.

Flags: --allow-missing-template-keys, --clusterrole, --dry-run, --field-manager, --group, --output, --role, --save-config, --serviceaccount, --show-managed-fields, --template, --user, --validate

## `coily kubectl create secret docker-registry`

Create a new secret for use with Docker registries.

Flags: --allow-missing-template-keys, --append-hash, --docker-email, --docker-password, --docker-server, --docker-username, --dry-run, --field-manager, --from-file, --output, --save-config, --show-managed-fields, --template, --validate

## `coily kubectl create secret generic`

Create a secret based on a file, directory, or specified literal value.

Flags: --allow-missing-template-keys, --append-hash, --dry-run, --field-manager, --from-env-file, --from-file, --from-literal, --output, --save-config, --show-managed-fields, --template, --type, --validate

## `coily kubectl create secret tls`

Create a TLS secret from the given public/private key pair.

Flags: --allow-missing-template-keys, --append-hash, --cert, --dry-run, --field-manager, --key, --output, --save-config, --show-managed-fields, --template, --validate

## `coily kubectl create service clusterip`

Create a ClusterIP service with the specified name.

Flags: --allow-missing-template-keys, --clusterip, --dry-run, --field-manager, --output, --save-config, --show-managed-fields, --tcp, --template, --validate

## `coily kubectl create service externalname`

Create an ExternalName service with the specified name.

Flags: --allow-missing-template-keys, --dry-run, --external-name, --field-manager, --output, --save-config, --show-managed-fields, --tcp, --template, --validate

## `coily kubectl create service loadbalancer`

Create a LoadBalancer service with the specified name.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --output, --save-config, --show-managed-fields, --tcp, --template, --validate

## `coily kubectl create service nodeport`

Create a NodePort service with the specified name.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --node-port, --output, --save-config, --show-managed-fields, --tcp, --template, --validate

## `coily kubectl create serviceaccount`

Create a service account with the specified name.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --output, --save-config, --show-managed-fields, --template, --validate

## `coily kubectl create token`

Request a service account token.

Flags: --allow-missing-template-keys, --audience, --bound-object-kind, --bound-object-name, --bound-object-uid, --duration, --output, --show-managed-fields, --template

## `coily kubectl delete`

Delete resources by file names, stdin, resources and names, or by resources and label selector.

Flags: --all, --all-namespaces, --cascade, --dry-run, --field-selector, --filename, --force, --grace-period, --ignore-not-found, --kustomize, --now, --output, --raw, --recursive, --selector, --timeout, --wait

## `coily kubectl describe`

Show details of a specific resource or group of resources.

Flags: --all-namespaces, --chunk-size, --filename, --kustomize, --recursive, --selector, --show-events

## `coily kubectl diff`

Diff configurations specified by file name or stdin between the current online configuration, and the configuration as it would be if applied.

Flags: --field-manager, --filename, --force-conflicts, --kustomize, --prune, --prune-allowlist, --recursive, --selector, --server-side, --show-managed-fields

## `coily kubectl drain`

Drain node in preparation for maintenance.

Flags: --chunk-size, --delete-emptydir-data, --disable-eviction, --dry-run, --force, --grace-period, --ignore-daemonsets, --pod-selector, --selector, --skip-wait-for-delete-timeout, --timeout

## `coily kubectl events`

Display events

Flags: --all-namespaces, --allow-missing-template-keys, --chunk-size, --for, --no-headers, --output, --show-managed-fields, --template, --types, --watch

## `coily kubectl explain`

List the fields for supported resources.

Flags: --api-version, --output, --recursive

## `coily kubectl expose`

Expose a resource as a new Kubernetes service.

Flags: --allow-missing-template-keys, --cluster-ip, --dry-run, --external-ip, --field-manager, --filename, --kustomize, --labels, --load-balancer-ip, --name, --output, --override-type, --overrides, --port, --protocol, --recursive, --save-config, --selector, --session-affinity, --show-managed-fields, --target-port, --template, --type

## `coily kubectl get`

Display one or many resources.

Flags: --all-namespaces, --allow-missing-template-keys, --chunk-size, --field-selector, --filename, --ignore-not-found, --kustomize, --label-columns, --no-headers, --output, --output-watch-events, --raw, --recursive, --selector, --server-print, --show-kind, --show-labels, --show-managed-fields, --sort-by, --subresource, --template, --watch, --watch-only

## `coily kubectl label`

Update the labels on a resource.

Flags: --all, --all-namespaces, --allow-missing-template-keys, --dry-run, --field-manager, --field-selector, --filename, --kustomize, --list, --local, --output, --overwrite, --recursive, --resource-version, --selector, --show-managed-fields, --template

## `coily kubectl logs`

Print the logs for a container in a pod or specified resource.

Flags: --all-containers, --container, --follow, --ignore-errors, --insecure-skip-tls-verify-backend, --limit-bytes, --max-log-requests, --pod-running-timeout, --prefix, --previous, --selector, --since, --since-time, --tail, --timestamps

## `coily kubectl patch`

Update fields of a resource using strategic merge patch, a JSON merge patch, or a JSON patch.

Flags: --allow-missing-template-keys, --dry-run, --field-manager, --filename, --kustomize, --local, --output, --patch, --patch-file, --recursive, --show-managed-fields, --subresource, --template, --type

## `coily kubectl rollout history`

View previous rollout revisions and configurations.

Flags: --allow-missing-template-keys, --filename, --kustomize, --output, --recursive, --revision, --selector, --show-managed-fields, --template

## `coily kubectl rollout pause`

Mark the provided resource as paused.

Flags: --allow-missing-template-keys, --field-manager, --filename, --kustomize, --output, --recursive, --selector, --show-managed-fields, --template

## `coily kubectl rollout restart`

Restart a resource.

Flags: --allow-missing-template-keys, --field-manager, --filename, --kustomize, --output, --recursive, --selector, --show-managed-fields, --template

## `coily kubectl rollout resume`

Resume a paused resource.

Flags: --allow-missing-template-keys, --field-manager, --filename, --kustomize, --output, --recursive, --selector, --show-managed-fields, --template

## `coily kubectl rollout status`

Show the status of the rollout.

Flags: --filename, --kustomize, --recursive, --revision, --selector, --timeout, --watch

## `coily kubectl rollout undo`

Roll back to a previous rollout.

Flags: --allow-missing-template-keys, --dry-run, --filename, --kustomize, --output, --recursive, --selector, --show-managed-fields, --template, --to-revision

## `coily kubectl scale`

Set a new size for a deployment, replica set, replication controller, or stateful set.

Flags: --all, --allow-missing-template-keys, --current-replicas, --dry-run, --filename, --kustomize, --output, --recursive, --replicas, --resource-version, --selector, --show-managed-fields, --template, --timeout

## `coily kubectl set env`

Update environment variables on a pod template.

Flags: --all, --allow-missing-template-keys, --containers, --dry-run, --env, --field-manager, --filename, --from, --keys, --kustomize, --list, --local, --output, --overwrite, --prefix, --recursive, --resolve, --selector, --show-managed-fields, --template

## `coily kubectl set image`

Update existing container image(s) of resources.

Flags: --all, --allow-missing-template-keys, --dry-run, --field-manager, --filename, --kustomize, --local, --output, --recursive, --selector, --show-managed-fields, --template

## `coily kubectl set resources`

Specify compute resource requirements (CPU, memory) for any resource that defines a pod template.

Flags: --all, --allow-missing-template-keys, --containers, --dry-run, --field-manager, --filename, --kustomize, --limits, --local, --output, --recursive, --requests, --selector, --show-managed-fields, --template

## `coily kubectl set selector`

Set the selector on a resource.

Flags: --all, --allow-missing-template-keys, --dry-run, --field-manager, --filename, --local, --output, --recursive, --resource-version, --show-managed-fields, --template

## `coily kubectl set serviceaccount`

Update the service account of pod template resources.

Flags: --all, --allow-missing-template-keys, --dry-run, --field-manager, --filename, --kustomize, --local, --output, --recursive, --show-managed-fields, --template

## `coily kubectl set subject`

Update the user, group, or service account in a role binding or cluster role binding.

Flags: --all, --allow-missing-template-keys, --dry-run, --field-manager, --filename, --group, --kustomize, --local, --output, --recursive, --selector, --serviceaccount, --show-managed-fields, --template, --user

## `coily kubectl taint`

Update the taints on one or more nodes.

Flags: --all, --allow-missing-template-keys, --dry-run, --field-manager, --output, --overwrite, --selector, --show-managed-fields, --template, --validate

## `coily kubectl top node`

Display resource (CPU/memory) usage of nodes.

Flags: --no-headers, --selector, --show-capacity, --sort-by, --use-protocol-buffers

## `coily kubectl top pod`

Display resource (CPU/memory) usage of pods.

Flags: --all-namespaces, --containers, --field-selector, --no-headers, --selector, --sort-by, --sum, --use-protocol-buffers

## `coily kubectl uncordon`

Mark node as schedulable.

Flags: --dry-run, --selector

## `coily kubectl wait`

Experimental: Wait for a specific condition on one or many resources.

Flags: --all, --all-namespaces, --allow-missing-template-keys, --field-selector, --filename, --for, --local, --output, --recursive, --selector, --show-managed-fields, --template, --timeout

## `coily lockdown skill`

Regenerate skills/coily-passthroughs/SKILL.md from the in-process command tree.

Flags: --out

## `coily modio mods comments`

GET /games/{game-id}/mods/{mod-id}/comments

## `coily modio mods files`

GET /games/{game-id}/mods/{mod-id}/files

## `coily modio mods get`

GET /games/{game-id}/mods/{mod-id}

## `coily modio mods list`

GET /games/{game-id}/mods

Flags: --limit, --offset

## `coily ssh copy`

Upload a local file to the remote via sftp.

Flags: --host, --user

## `coily ssh rm-unit`

Remove /etc/systemd/system/<unit>.service and reload systemd.

Flags: --host, --user

## `coily ssh systemctl daemon-reload`

Run systemctl daemon-reload.

Flags: --host, --user

## `coily ssh systemctl disable`

Disable <unit>.

Flags: --host, --user

## `coily ssh systemctl enable`

Enable <unit>.

Flags: --host, --user

## `coily ssh systemctl restart`

Restart <unit>.

Flags: --host, --user

## `coily ssh systemctl start`

Start <unit>.

Flags: --host, --user

## `coily ssh systemctl status`

Print systemctl status of <unit>.

Flags: --host, --user

## `coily ssh systemctl stop`

Stop <unit>.

Flags: --host, --user

## `coily tailscale cert`

Get TLS certs

Flags: --cert-file, --key-file, --min-validity

## `coily tailscale down`

Disconnect from Tailscale

Flags: --accept-risk, --reason

## `coily tailscale funnel reset`

Reset current funnel config

## `coily tailscale funnel status`

View current funnel configuration

Flags: --json

## `coily tailscale ip`

Show Tailscale IP addresses

Flags: --assert

## `coily tailscale login`

Log in to a Tailscale account

Flags: --advertise-routes, --advertise-tags, --audience, --auth-key, --client-id, --client-secret, --exit-node, --hostname, --id-token, --login-server, --nickname, --qr-format, --timeout

## `coily tailscale logout`

Disconnect from Tailscale and expire current node key

Flags: --reason

## `coily tailscale netcheck`

Print an analysis of local network conditions

Flags: --bind-address, --bind-port, --every, --format

## `coily tailscale ping`

Ping a host at the Tailscale layer, see how it routed

Flags: --c, --size, --timeout

## `coily tailscale serve advertise`

Advertise this node as a service proxy to the tailnet

## `coily tailscale serve clear`

Remove all config for a service

## `coily tailscale serve drain`

Drain a service from the current node

## `coily tailscale serve get-config`

Get service configuration to save to a file

Flags: --service

## `coily tailscale serve reset`

Reset current serve config

## `coily tailscale serve set-config`

Define service configuration from a file

Flags: --service

## `coily tailscale serve status`

View current serve configuration

Flags: --json

## `coily tailscale set`

Change specified preferences

Flags: --accept-risk, --advertise-routes, --exit-node, --hostname, --nickname, --relay-server-port, --relay-server-static-endpoints

## `coily tailscale ssh`

SSH to a Tailscale machine

## `coily tailscale status`

Show state of tailscaled and its connections

Flags: --listen

## `coily tailscale switch remove`

Remove a Tailscale account

## `coily tailscale up`

Connect to Tailscale, logging in if needed

Flags: --accept-risk, --advertise-routes, --advertise-tags, --audience, --auth-key, --client-id, --client-secret, --exit-node, --hostname, --id-token, --login-server, --qr-format, --timeout

## `coily tailscale update`

Update Tailscale to the latest/different version

Flags: --track, --version

## `coily tailscale whois`

Show the machine and user associated with a Tailscale IP (v4 or v6)

Flags: --proto

## `coily version`

Print the build version and exit.

## `coily whoami`

Print the authenticated identity coily sees across aws, kubectl, and gh.
