# coily aws - full reference

Mirrors `aws`. Underlying version at scan time: aws-cli/2.26.5 Python/3.13.3 Darwin/24.6.0 source/arm64

Command shape: `coily aws <verb...> [flags]`. Flags match the underlying CLI.

## `coily aws route53`

### `coily aws route53` (group)

Amazon Route 53 is a highly available and scalable Domain Name System (DNS) web service.

Subcommands: `activate-key-signing-key`, `associate-vpc-with-hosted-zone`, `change-cidr-collection`, `change-resource-record-sets`, `change-tags-for-resource`, `create-cidr-collection`, `create-health-check`, `create-hosted-zone`, `create-key-signing-key`, `create-query-logging-config`, `create-reusable-delegation-set`, `create-traffic-policy`, `create-traffic-policy-instance`, `create-traffic-policy-version`, `create-vpc-association-authorization`, `deactivate-key-signing-key`, `delete-cidr-collection`, `delete-health-check`, `delete-hosted-zone`, `delete-key-signing-key`, `delete-query-logging-config`, `delete-reusable-delegation-set`, `delete-traffic-policy`, `delete-traffic-policy-instance`, `delete-vpc-association-authorization`, `disable-hosted-zone-dnssec`, `disassociate-vpc-from-hosted-zone`, `enable-hosted-zone-dnssec`, `get-account-limit`, `get-change`, `get-checker-ip-ranges`, `get-dnssec`, `get-geo-location`, `get-health-check`, `get-health-check-count`, `get-health-check-last-failure-reason`, `get-health-check-status`, `get-hosted-zone`, `get-hosted-zone-count`, `get-hosted-zone-limit`, `get-query-logging-config`, `get-reusable-delegation-set`, `get-reusable-delegation-set-limit`, `get-traffic-policy`, `get-traffic-policy-instance`, `get-traffic-policy-instance-count`, `list-cidr-blocks`, `list-cidr-collections`, `list-cidr-locations`, `list-geo-locations`, `list-health-checks`, `list-hosted-zones`, `list-hosted-zones-by-name`, `list-hosted-zones-by-vpc`, `list-query-logging-configs`, `list-resource-record-sets`, `list-reusable-delegation-sets`, `list-tags-for-resource`, `list-tags-for-resources`, `list-traffic-policies`, `list-traffic-policy-instances`, `list-traffic-policy-instances-by-policy`, `list-traffic-policy-versions`, `list-vpc-association-authorizations`, `test-dns-answer`, `update-health-check`, `update-hosted-zone-comment`, `update-traffic-policy-comment`, `update-traffic-policy-instance`, `wait`

### `coily aws route53 activate-key-signing-key`

Activates a key-signing key (KSK) so that it can be used for signing by DNSSEC.

### `coily aws route53 associate-vpc-with-hosted-zone`

Associates an Amazon VPC with a private hosted zone.

### `coily aws route53 change-cidr-collection`

Creates, changes, or deletes CIDR blocks within a collection.

### `coily aws route53 change-resource-record-sets`

Creates, changes, or deletes a resource record set, which contains authoritative DNS information for a specified domain name or subdomain name.

### `coily aws route53 change-tags-for-resource`

Adds, edits, or deletes tags for a health check or a hosted zone.

### `coily aws route53 create-cidr-collection`

Creates a CIDR collection in the current Amazon Web Services account.

### `coily aws route53 create-health-check`

Creates a new health check.

### `coily aws route53 create-hosted-zone`

Creates a new public or private hosted zone.

### `coily aws route53 create-key-signing-key`

Creates a new key-signing key (KSK) associated with a hosted zone.

### `coily aws route53 create-query-logging-config`

Creates a configuration for DNS query logging.

### `coily aws route53 create-reusable-delegation-set`

Creates a delegation set (a group of four name servers) that can be reused by multiple hosted zones that were created by the same Amazon Web Services account.

### `coily aws route53 create-traffic-policy`

Creates a traffic policy, which you use to create multiple DNS resource record sets for one domain name (such as example.com) or one subdomain name (such as www.example.com).

### `coily aws route53 create-traffic-policy-instance`

Creates resource record sets in a specified hosted zone based on the settings in a specified traffic policy version.

### `coily aws route53 create-traffic-policy-version`

Creates a new version of an existing traffic policy.

### `coily aws route53 create-vpc-association-authorization`

Authorizes the Amazon Web Services account that created a specified VPC to submit an AssociateVPCWithHostedZone request to associate the VPC with a specified hosted zone that was created by a different account.

### `coily aws route53 deactivate-key-signing-key`

Deactivates a key-signing key (KSK) so that it will not be used for signing by DNSSEC.

### `coily aws route53 delete-cidr-collection`

Deletes a CIDR collection in the current Amazon Web Services account.

### `coily aws route53 delete-health-check`

Deletes a health check.

### `coily aws route53 delete-hosted-zone`

Deletes a hosted zone.

### `coily aws route53 delete-key-signing-key`

Deletes a key-signing key (KSK).

### `coily aws route53 delete-query-logging-config`

Deletes a configuration for DNS query logging.

### `coily aws route53 delete-reusable-delegation-set`

Deletes a reusable delegation set.

### `coily aws route53 delete-traffic-policy`

Deletes a traffic policy.

### `coily aws route53 delete-traffic-policy-instance`

Deletes a traffic policy instance and all of the resource record sets that Amazon Route 53 created when you created the instance.

### `coily aws route53 delete-vpc-association-authorization`

Removes authorization to submit an AssociateVPCWithHostedZone request to associate a specified VPC with a hosted zone that was created by a different account.

### `coily aws route53 disable-hosted-zone-dnssec`

Disables DNSSEC signing in a specific hosted zone.

### `coily aws route53 disassociate-vpc-from-hosted-zone`

Disassociates an Amazon Virtual Private Cloud (Amazon VPC) from an Amazon Route 53 private hosted zone.

### `coily aws route53 enable-hosted-zone-dnssec`

Enables DNSSEC signing in a specific hosted zone.

### `coily aws route53 get-account-limit`

Gets the specified limit for the current account, for example, the maximum number of health checks that you can create using the account.

### `coily aws route53 get-change`

Returns the current status of a change batch request.

### `coily aws route53 get-checker-ip-ranges`

Route 53 does not perform authorization for this API because it retrieves information that is already available to the public.

### `coily aws route53 get-dnssec`

Returns information about DNSSEC for a specific hosted zone, including the key-signing keys (KSKs) in the hosted zone.

### `coily aws route53 get-geo-location`

Gets information about whether a specified geographic location is supported for Amazon Route 53 geolocation resource record sets.

### `coily aws route53 get-health-check`

Gets information about a specified health check.

### `coily aws route53 get-health-check-count`

Retrieves the number of health checks that are associated with the current Amazon Web Services account.

### `coily aws route53 get-health-check-last-failure-reason`

Gets the reason that a specified health check failed most recently.

### `coily aws route53 get-health-check-status`

Gets status of a specified health check.

### `coily aws route53 get-hosted-zone`

Gets information about a specified hosted zone including the four name servers assigned to the hosted zone.

### `coily aws route53 get-hosted-zone-count`

Retrieves the number of hosted zones that are associated with the current Amazon Web Services account.

### `coily aws route53 get-hosted-zone-limit`

Gets the specified limit for a specified hosted zone, for example, the maximum number of records that you can create in the hosted zone.

### `coily aws route53 get-query-logging-config`

Gets information about a specified configuration for DNS query logging.

### `coily aws route53 get-reusable-delegation-set`

Retrieves information about a specified reusable delegation set, including the four name servers that are assigned to the delegation set.

### `coily aws route53 get-reusable-delegation-set-limit`

Gets the maximum number of hosted zones that you can associate with the specified reusable delegation set.

### `coily aws route53 get-traffic-policy`

Gets information about a specific traffic policy version.

### `coily aws route53 get-traffic-policy-instance`

Gets information about a specified traffic policy instance.

### `coily aws route53 get-traffic-policy-instance-count`

Gets the number of traffic policy instances that are associated with the current Amazon Web Services account.

### `coily aws route53 list-cidr-blocks`

Returns a paginated list of location objects and their CIDR blocks.

### `coily aws route53 list-cidr-collections`

Returns a paginated list of CIDR collections in the Amazon Web Services account (metadata only).

### `coily aws route53 list-cidr-locations`

Returns a paginated list of CIDR locations for the given collection (metadata only, does not include CIDR blocks).

### `coily aws route53 list-geo-locations`

Retrieves a list of supported geographic locations.

### `coily aws route53 list-health-checks`

Retrieve a list of the health checks that are associated with the current Amazon Web Services account.

### `coily aws route53 list-hosted-zones`

Retrieves a list of the public and private hosted zones that are associated with the current Amazon Web Services account.

### `coily aws route53 list-hosted-zones-by-name`

Retrieves a list of your hosted zones in lexicographic order.

### `coily aws route53 list-hosted-zones-by-vpc`

Lists all the private hosted zones that a specified VPC is associated with, regardless of which Amazon Web Services account or Amazon Web Services service owns the hosted zones.

### `coily aws route53 list-query-logging-configs`

Lists the configurations for DNS query logging that are associated with the current Amazon Web Services account or the configuration that is associated with a specified hosted zone.

### `coily aws route53 list-resource-record-sets`

Lists the resource record sets in a specified hosted zone.

### `coily aws route53 list-reusable-delegation-sets`

Retrieves a list of the reusable delegation sets that are associated with the current Amazon Web Services account.

### `coily aws route53 list-tags-for-resource`

Lists tags for one health check or hosted zone.

### `coily aws route53 list-tags-for-resources`

Lists tags for up to 10 health checks or hosted zones.

### `coily aws route53 list-traffic-policies`

Gets information about the latest version for every traffic policy that is associated with the current Amazon Web Services account.

### `coily aws route53 list-traffic-policy-instances`

Gets information about the traffic policy instances that you created by using the current Amazon Web Services account.

### `coily aws route53 list-traffic-policy-instances-by-policy`

Gets information about the traffic policy instances that you created by using a specify traffic policy version.

### `coily aws route53 list-traffic-policy-versions`

Gets information about all of the versions for a specified traffic policy.

### `coily aws route53 list-vpc-association-authorizations`

Gets a list of the VPCs that were created by other accounts and that can be associated with a specified hosted zone because you've submitted one or more CreateVPCAssociationAuthorization requests.

### `coily aws route53 test-dns-answer`

Gets the value that Amazon Route 53 returns in response to a DNS request for a specified record name and type.

### `coily aws route53 update-health-check`

Updates an existing health check.

### `coily aws route53 update-hosted-zone-comment`

Updates the comment for a specified hosted zone.

### `coily aws route53 update-traffic-policy-comment`

Updates the comment for a specified traffic policy version.

### `coily aws route53 update-traffic-policy-instance`

NOTE: After you submit a UpdateTrafficPolicyInstance request, there's a brief delay while Route 53 creates the resource record sets that are specified in the traffic policy definition.

### `coily aws route53 wait` (group)

Wait until a particular condition is satisfied.

Subcommands: `resource-record-sets-changed`

### `coily aws route53 wait resource-record-sets-changed`

Wait until JMESPath query ChangeInfo.Status returns INSYNC when polling with get-change.

## `coily aws s3`

### `coily aws s3` (group)

This section explains prominent concepts and notations in the set of high-level S3 commands provided.

Subcommands: `cp`, `ls`, `mb`, `mv`, `presign`, `rb`, `rm`, `sync`, `website`

### `coily aws s3 cp`

Copies a local file or S3 object to another location locally or in S3.

### `coily aws s3 ls`

List S3 objects and common prefixes under a prefix or all S3 buckets.

### `coily aws s3 mb`

Creates an S3 bucket.

### `coily aws s3 mv`

Moves a local file or S3 object to another location locally or in S3.

### `coily aws s3 presign`

Generate a pre-signed URL for an Amazon S3 object.

### `coily aws s3 rb`

Deletes an empty S3 bucket.

### `coily aws s3 rm`

Deletes an S3 object.

### `coily aws s3 sync`

Syncs directories and S3 prefixes.

### `coily aws s3 website`

Set the website configuration for a bucket.

## `coily aws s3api`

### `coily aws s3api` (group)

S3API()                                                                S3API()

Subcommands: `abort-multipart-upload`, `complete-multipart-upload`, `copy-object`, `create-bucket`, `create-multipart-upload`, `create-session`, `delete-bucket`, `delete-bucket-analytics-configuration`, `delete-bucket-cors`, `delete-bucket-encryption`, `delete-bucket-inventory-configuration`, `delete-bucket-lifecycle`, `delete-bucket-metrics-configuration`, `delete-bucket-ownership-controls`, `delete-bucket-policy`, `delete-bucket-replication`, `delete-bucket-tagging`, `delete-bucket-website`, `delete-object`, `delete-object-tagging`, `delete-objects`, `delete-public-access-block`, `get-bucket-accelerate-configuration`, `get-bucket-acl`, `get-bucket-analytics-configuration`, `get-bucket-cors`, `get-bucket-encryption`, `get-bucket-inventory-configuration`, `get-bucket-lifecycle-configuration`, `get-bucket-location`, `get-bucket-logging`, `get-bucket-metadata-table-configuration`, `get-bucket-metrics-configuration`, `get-bucket-notification-configuration`, `get-bucket-ownership-controls`, `get-bucket-policy`, `get-bucket-policy-status`, `get-bucket-replication`, `get-bucket-request-payment`, `get-bucket-tagging`, `get-bucket-versioning`, `get-bucket-website`, `get-object`, `get-object-acl`, `get-object-attributes`, `get-object-legal-hold`, `get-object-lock-configuration`, `get-object-retention`, `get-object-tagging`, `get-object-torrent`, `get-public-access-block`, `head-bucket`, `head-object`, `list-bucket-analytics-configurations`, `list-bucket-inventory-configurations`, `list-bucket-metrics-configurations`, `list-buckets`, `list-directory-buckets`, `list-multipart-uploads`, `list-object-versions`, `list-objects`, `list-objects-v2`, `list-parts`, `put-bucket-accelerate-configuration`, `put-bucket-acl`, `put-bucket-analytics-configuration`, `put-bucket-cors`, `put-bucket-encryption`, `put-bucket-inventory-configuration`, `put-bucket-lifecycle-configuration`, `put-bucket-logging`, `put-bucket-metrics-configuration`, `put-bucket-notification-configuration`, `put-bucket-ownership-controls`, `put-bucket-policy`, `put-bucket-replication`, `put-bucket-request-payment`, `put-bucket-tagging`, `put-bucket-versioning`, `put-bucket-website`, `put-object`, `put-object-acl`, `put-object-legal-hold`, `put-object-lock-configuration`, `put-object-retention`, `put-object-tagging`, `put-public-access-block`, `restore-object`, `select-object-content`, `upload-part`, `upload-part-copy`, `wait`, `write-get-object-response`

### `coily aws s3api abort-multipart-upload`

This operation aborts a multipart upload.

### `coily aws s3api complete-multipart-upload`

Completes a multipart upload by assembling previously uploaded parts.

### `coily aws s3api copy-object`

Creates a copy of an object that is already stored in Amazon S3.

### `coily aws s3api create-bucket`

NOTE: This action creates an Amazon S3 bucket.

### `coily aws s3api create-multipart-upload`

This action initiates a multipart upload and returns an upload ID.

### `coily aws s3api create-session`

Creates a session that establishes temporary security credentials to support fast authentication and authorization for the Zonal endpoint API operations on directory buckets.

### `coily aws s3api delete-bucket`

Deletes the S3 bucket.

### `coily aws s3api delete-bucket-analytics-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api delete-bucket-cors`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api delete-bucket-encryption`

This implementation of the DELETE action resets the default encryption for the bucket as server-side encryption with Amazon S3 managed keys (SSE-S3).

### `coily aws s3api delete-bucket-inventory-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api delete-bucket-lifecycle`

Deletes the lifecycle configuration from the specified bucket.

### `coily aws s3api delete-bucket-metrics-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api delete-bucket-ownership-controls`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api delete-bucket-policy`

Deletes the policy of a specified bucket.

### `coily aws s3api delete-bucket-replication`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api delete-bucket-tagging`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api delete-bucket-website`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api delete-object`

Removes an object from a bucket.

### `coily aws s3api delete-object-tagging`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api delete-objects`

This operation enables you to delete multiple objects from a bucket using a single HTTP request.

### `coily aws s3api delete-public-access-block`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-accelerate-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-acl`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-analytics-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-cors`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-encryption`

Returns the default encryption configuration for an Amazon S3 bucket.

### `coily aws s3api get-bucket-inventory-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-lifecycle-configuration`

Returns the lifecycle configuration information set on the bucket.

### `coily aws s3api get-bucket-location`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-logging`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-metadata-table-configuration`

Retrieves the metadata table configuration for a general purpose bucket.

### `coily aws s3api get-bucket-metrics-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-notification-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-ownership-controls`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-policy`

Returns the policy of a specified bucket.

### `coily aws s3api get-bucket-policy-status`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-replication`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-request-payment`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-tagging`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-versioning`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-bucket-website`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-object`

Retrieves an object from Amazon S3.

### `coily aws s3api get-object-acl`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-object-attributes`

Retrieves all the metadata from an object without returning the object itself.

### `coily aws s3api get-object-legal-hold`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-object-lock-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-object-retention`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-object-tagging`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-object-torrent`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api get-public-access-block`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api head-bucket`

You can use this operation to determine if a bucket exists and if you have permission to access it.

### `coily aws s3api head-object`

The HEAD operation retrieves metadata from an object without returning the object itself.

### `coily aws s3api list-bucket-analytics-configurations`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api list-bucket-inventory-configurations`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api list-bucket-metrics-configurations`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api list-buckets`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api list-directory-buckets`

Returns a list of all Amazon S3 directory buckets owned by the authenticated sender of the request.

### `coily aws s3api list-multipart-uploads`

This operation lists in-progress multipart uploads in a bucket.

### `coily aws s3api list-object-versions`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api list-objects`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api list-objects-v2`

Returns some or all (up to 1,000) of the objects in a bucket with each request.

### `coily aws s3api list-parts`

Lists the parts that have been uploaded for a specific multipart upload.

### `coily aws s3api put-bucket-accelerate-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-acl`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-analytics-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-cors`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-encryption`

This operation configures default encryption and Amazon S3 Bucket Keys for an existing bucket.

### `coily aws s3api put-bucket-inventory-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-lifecycle-configuration`

Creates a new lifecycle configuration for the bucket or replaces an existing lifecycle configuration.

### `coily aws s3api put-bucket-logging`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-metrics-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-notification-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-ownership-controls`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-policy`

Applies an Amazon S3 bucket policy to an Amazon S3 bucket.

### `coily aws s3api put-bucket-replication`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-request-payment`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-tagging`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-versioning`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-bucket-website`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-object`

<string>:: (ERROR/3) Anonymous hyperlink mismatch: 2 references but 0 targets.

### `coily aws s3api put-object-acl`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-object-legal-hold`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-object-lock-configuration`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-object-retention`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-object-tagging`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api put-public-access-block`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api restore-object`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api select-object-content`

NOTE: This operation is not supported for directory buckets.

### `coily aws s3api upload-part`

Uploads a part in a multipart upload.

### `coily aws s3api upload-part-copy`

Uploads a part by copying data from an existing object as data source.

### `coily aws s3api wait` (group)

Wait until a particular condition is satisfied.

Subcommands: `bucket-exists`, `bucket-not-exists`, `object-exists`, `object-not-exists`

### `coily aws s3api wait bucket-exists`

Wait until 200 response is received when polling with head-bucket.

### `coily aws s3api wait bucket-not-exists`

Wait until 404 response is received when polling with head-bucket.

### `coily aws s3api wait object-exists`

Wait until 200 response is received when polling with head-object.

### `coily aws s3api wait object-not-exists`

Wait until 404 response is received when polling with head-object.

### `coily aws s3api write-get-object-response`

NOTE: This operation is not supported for directory buckets.

## `coily aws ssm`

### `coily aws ssm` (group)

Amazon Web Services Systems Manager is the operations hub for your Amazon Web Services applications and resources and a secure end-to-end management solution for hybrid cloud environments that enables safe and secure operations at scale.

Subcommands: `add-tags-to-resource`, `associate-ops-item-related-item`, `cancel-command`, `cancel-maintenance-window-execution`, `create-activation`, `create-association`, `create-association-batch`, `create-document`, `create-maintenance-window`, `create-ops-item`, `create-ops-metadata`, `create-patch-baseline`, `create-resource-data-sync`, `delete-activation`, `delete-association`, `delete-document`, `delete-inventory`, `delete-maintenance-window`, `delete-ops-item`, `delete-ops-metadata`, `delete-parameter`, `delete-parameters`, `delete-patch-baseline`, `delete-resource-data-sync`, `delete-resource-policy`, `deregister-managed-instance`, `deregister-patch-baseline-for-patch-group`, `deregister-target-from-maintenance-window`, `deregister-task-from-maintenance-window`, `describe-activations`, `describe-association`, `describe-association-execution-targets`, `describe-association-executions`, `describe-automation-executions`, `describe-automation-step-executions`, `describe-available-patches`, `describe-document`, `describe-document-permission`, `describe-effective-instance-associations`, `describe-instance-associations-status`, `describe-instance-information`, `describe-instance-patch-states`, `describe-instance-patches`, `describe-instance-properties`, `describe-inventory-deletions`, `describe-maintenance-window-executions`, `describe-maintenance-window-schedule`, `describe-maintenance-window-targets`, `describe-maintenance-window-tasks`, `describe-maintenance-windows`, `describe-maintenance-windows-for-target`, `describe-ops-items`, `describe-parameters`, `describe-patch-baselines`, `describe-patch-group-state`, `describe-patch-groups`, `describe-patch-properties`, `describe-sessions`, `disassociate-ops-item-related-item`, `get-automation-execution`, `get-calendar-state`, `get-command-invocation`, `get-connection-status`, `get-default-patch-baseline`, `get-document`, `get-execution-preview`, `get-inventory`, `get-inventory-schema`, `get-maintenance-window`, `get-maintenance-window-execution`, `get-maintenance-window-execution-task`, `get-maintenance-window-task`, `get-ops-item`, `get-ops-metadata`, `get-ops-summary`, `get-parameter`, `get-parameter-history`, `get-parameters`, `get-parameters-by-path`, `get-patch-baseline`, `get-patch-baseline-for-patch-group`, `get-resource-policies`, `get-service-setting`, `label-parameter-version`, `list-association-versions`, `list-associations`, `list-command-invocations`, `list-compliance-items`, `list-compliance-summaries`, `list-document-metadata-history`, `list-document-versions`, `list-documents`, `list-inventory-entries`, `list-nodes`, `list-nodes-summary`, `list-ops-item-events`, `list-ops-item-related-items`, `list-ops-metadata`, `list-resource-compliance-summaries`, `list-resource-data-sync`, `list-tags-for-resource`, `modify-document-permission`, `put-compliance-items`, `put-inventory`, `put-parameter`, `put-resource-policy`, `register-default-patch-baseline`, `register-patch-baseline-for-patch-group`, `register-target-with-maintenance-window`, `register-task-with-maintenance-window`, `remove-tags-from-resource`, `reset-service-setting`, `resume-session`, `send-automation-signal`, `send-command`, `start-associations-once`, `start-automation-execution`, `start-change-request-execution`, `start-execution-preview`, `start-session`, `stop-automation-execution`, `terminate-session`, `unlabel-parameter-version`, `update-association`, `update-association-status`, `update-document`, `update-document-default-version`, `update-document-metadata`, `update-maintenance-window`, `update-maintenance-window-target`, `update-maintenance-window-task`, `update-managed-instance-role`, `update-ops-item`, `update-ops-metadata`, `update-patch-baseline`, `update-resource-data-sync`, `update-service-setting`, `wait`

### `coily aws ssm add-tags-to-resource`

Adds or overwrites one or more tags for the specified resource.

### `coily aws ssm associate-ops-item-related-item`

Associates a related item to a Systems Manager OpsCenter OpsItem.

### `coily aws ssm cancel-command`

Attempts to cancel the command specified by the Command ID.

### `coily aws ssm cancel-maintenance-window-execution`

Stops a maintenance window execution that is already in progress and cancels any tasks in the window that haven't already starting running.

### `coily aws ssm create-activation`

Generates an activation code and activation ID you can use to register your on-premises servers, edge devices, or virtual machine (VM) with Amazon Web Services Systems Manager.

### `coily aws ssm create-association`

A State Manager association defines the state that you want to maintain on your managed nodes.

### `coily aws ssm create-association-batch`

Associates the specified Amazon Web Services Systems Manager document (SSM document) with the specified managed nodes or targets.

### `coily aws ssm create-document`

Creates a Amazon Web Services Systems Manager (SSM document).

### `coily aws ssm create-maintenance-window`

Creates a new maintenance window.

### `coily aws ssm create-ops-item`

Creates a new OpsItem.

### `coily aws ssm create-ops-metadata`

If you create a new application in Application Manager, Amazon Web Services Systems Manager calls this API operation to specify information about the new application, including the application type.

### `coily aws ssm create-patch-baseline`

Creates a patch baseline.

### `coily aws ssm create-resource-data-sync`

A resource data sync helps you view data from multiple sources in a single location.

### `coily aws ssm delete-activation`

Deletes an activation.

### `coily aws ssm delete-association`

Disassociates the specified Amazon Web Services Systems Manager document (SSM document) from the specified managed node.

### `coily aws ssm delete-document`

Deletes the Amazon Web Services Systems Manager document (SSM document) and all managed node associations to the document.

### `coily aws ssm delete-inventory`

Delete a custom inventory type or the data associated with a custom Inventory type.

### `coily aws ssm delete-maintenance-window`

Deletes a maintenance window.

### `coily aws ssm delete-ops-item`

Delete an OpsItem.

### `coily aws ssm delete-ops-metadata`

Delete OpsMetadata related to an application.

### `coily aws ssm delete-parameter`

Delete a parameter from the system.

### `coily aws ssm delete-parameters`

Delete a list of parameters.

### `coily aws ssm delete-patch-baseline`

Deletes a patch baseline.

### `coily aws ssm delete-resource-data-sync`

Deletes a resource data sync configuration.

### `coily aws ssm delete-resource-policy`

Deletes a Systems Manager resource policy.

### `coily aws ssm deregister-managed-instance`

Removes the server or virtual machine from the list of registered servers.

### `coily aws ssm deregister-patch-baseline-for-patch-group`

Removes a patch group from a patch baseline.

### `coily aws ssm deregister-target-from-maintenance-window`

Removes a target from a maintenance window.

### `coily aws ssm deregister-task-from-maintenance-window`

Removes a task from a maintenance window.

### `coily aws ssm describe-activations`

Describes details about the activation, such as the date and time the activation was created, its expiration date, the Identity and Access Management (IAM) role assigned to the managed nodes in the activation, and the number of nodes registered by using this activation.

### `coily aws ssm describe-association`

Describes the association for the specified target or managed node.

### `coily aws ssm describe-association-execution-targets`

Views information about a specific execution of a specific association.

### `coily aws ssm describe-association-executions`

Views all executions for a specific association ID.

### `coily aws ssm describe-automation-executions`

Provides details about all active and terminated Automation executions.

### `coily aws ssm describe-automation-step-executions`

Information about all active and terminated step executions in an Automation workflow.

### `coily aws ssm describe-available-patches`

Lists all patches eligible to be included in a patch baseline.

### `coily aws ssm describe-document`

Describes the specified Amazon Web Services Systems Manager document (SSM document).

### `coily aws ssm describe-document-permission`

Describes the permissions for a Amazon Web Services Systems Manager document (SSM document).

### `coily aws ssm describe-effective-instance-associations`

All associations for the managed nodes.

### `coily aws ssm describe-instance-associations-status`

The status of the associations for the managed nodes.

### `coily aws ssm describe-instance-information`

Provides information about one or more of your managed nodes, including the operating system platform, SSM Agent version, association status, and IP address.

### `coily aws ssm describe-instance-patch-states`

Retrieves the high-level patch state of one or more managed nodes.

### `coily aws ssm describe-instance-patches`

Retrieves information about the patches on the specified managed node and their state relative to the patch baseline being used for the node.

### `coily aws ssm describe-instance-properties`

An API operation used by the Systems Manager console to display information about Systems Manager managed nodes.

### `coily aws ssm describe-inventory-deletions`

Describes a specific delete inventory operation.

### `coily aws ssm describe-maintenance-window-executions`

Lists the executions of a maintenance window.

### `coily aws ssm describe-maintenance-window-schedule`

Retrieves information about upcoming executions of a maintenance window.

### `coily aws ssm describe-maintenance-window-targets`

Lists the targets registered with the maintenance window.

### `coily aws ssm describe-maintenance-window-tasks`

Lists the tasks in a maintenance window.

### `coily aws ssm describe-maintenance-windows`

Retrieves the maintenance windows in an Amazon Web Services account.

### `coily aws ssm describe-maintenance-windows-for-target`

Retrieves information about the maintenance window targets or tasks that a managed node is associated with.

### `coily aws ssm describe-ops-items`

Query a set of OpsItems.

### `coily aws ssm describe-parameters`

Lists the parameters in your Amazon Web Services account or the parameters shared with you when you enable the Shared option.

### `coily aws ssm describe-patch-baselines`

Lists the patch baselines in your Amazon Web Services account.

### `coily aws ssm describe-patch-group-state`

Returns high-level aggregated patch compliance state information for a patch group.

### `coily aws ssm describe-patch-groups`

Lists all patch groups that have been registered with patch baselines.

### `coily aws ssm describe-patch-properties`

Lists the properties of available patches organized by product, product family, classification, severity, and other properties of available patches.

### `coily aws ssm describe-sessions`

Retrieves a list of all active sessions (both connected and disconnected) or terminated sessions from the past 30 days.

### `coily aws ssm disassociate-ops-item-related-item`

Deletes the association between an OpsItem and a related item.

### `coily aws ssm get-automation-execution`

Get detailed information about a particular Automation execution.

### `coily aws ssm get-calendar-state`

Gets the state of a Amazon Web Services Systems Manager change calendar at the current time or a specified time.

### `coily aws ssm get-command-invocation`

Returns detailed information about command execution for an invocation or plugin.

### `coily aws ssm get-connection-status`

Retrieves the Session Manager connection status for a managed node to determine whether it is running and ready to receive Session Manager connections.

### `coily aws ssm get-default-patch-baseline`

Retrieves the default patch baseline.

### `coily aws ssm get-document`

Gets the contents of the specified Amazon Web Services Systems Manager document (SSM document).

### `coily aws ssm get-execution-preview`

Initiates the process of retrieving an existing preview that shows the effects that running a specified Automation runbook would have on the targeted resources.

### `coily aws ssm get-inventory`

Query inventory information.

### `coily aws ssm get-inventory-schema`

Return a list of inventory type names for the account, or return a list of attribute names for a specific Inventory item type.

### `coily aws ssm get-maintenance-window`

Retrieves a maintenance window.

### `coily aws ssm get-maintenance-window-execution`

Retrieves details about a specific a maintenance window execution.

### `coily aws ssm get-maintenance-window-execution-task`

Retrieves the details about a specific task run as part of a maintenance window execution.

### `coily aws ssm get-maintenance-window-task`

Retrieves the details of a maintenance window task.

### `coily aws ssm get-ops-item`

Get information about an OpsItem by using the ID.

### `coily aws ssm get-ops-metadata`

View operational metadata related to an application in Application Manager.

### `coily aws ssm get-ops-summary`

View a summary of operations metadata (OpsData) based on specified filters and aggregators.

### `coily aws ssm get-parameter`

Get information about a single parameter by specifying the parameter name.

### `coily aws ssm get-parameter-history`

Retrieves the history of all changes to a parameter.

### `coily aws ssm get-parameters`

Get information about one or more parameters by specifying multiple parameter names.

### `coily aws ssm get-parameters-by-path`

Retrieve information about one or more parameters under a specified level in a hierarchy.

### `coily aws ssm get-patch-baseline`

Retrieves information about a patch baseline.

### `coily aws ssm get-patch-baseline-for-patch-group`

Retrieves the patch baseline that should be used for the specified patch group.

### `coily aws ssm get-resource-policies`

Returns an array of the Policy object.

### `coily aws ssm get-service-setting`

ServiceSetting is an account-level setting for an Amazon Web Services service.

### `coily aws ssm label-parameter-version`

A parameter label is a user-defined alias to help you manage different versions of a parameter.

### `coily aws ssm list-association-versions`

Retrieves all versions of an association for a specific association ID.

### `coily aws ssm list-associations`

Returns all State Manager associations in the current Amazon Web Services account and Amazon Web Services Region.

### `coily aws ssm list-command-invocations`

An invocation is copy of a command sent to a specific managed node.

### `coily aws ssm list-compliance-items`

For a specified resource ID, this API operation returns a list of compliance statuses for different resource types.

### `coily aws ssm list-compliance-summaries`

Returns a summary count of compliant and non-compliant resources for a compliance type.

### `coily aws ssm list-document-metadata-history`

Information about approval reviews for a version of a change template in Change Manager.

### `coily aws ssm list-document-versions`

List all versions for a document.

### `coily aws ssm list-documents`

Returns all Systems Manager (SSM) documents in the current Amazon Web Services account and Amazon Web Services Region.

### `coily aws ssm list-inventory-entries`

A list of inventory items returned by the request.

### `coily aws ssm list-nodes`

Takes in filters and returns a list of managed nodes matching the filter criteria.

### `coily aws ssm list-nodes-summary`

Generates a summary of managed instance/node metadata based on the filters and aggregators you specify.

### `coily aws ssm list-ops-item-events`

Returns a list of all OpsItem events in the current Amazon Web Services Region and Amazon Web Services account.

### `coily aws ssm list-ops-item-related-items`

Lists all related-item resources associated with a Systems Manager OpsCenter OpsItem.

### `coily aws ssm list-ops-metadata`

Amazon Web Services Systems Manager calls this API operation when displaying all Application Manager OpsMetadata objects or blobs.

### `coily aws ssm list-resource-compliance-summaries`

Returns a resource-level summary count.

### `coily aws ssm list-resource-data-sync`

Lists your resource data sync configurations.

### `coily aws ssm list-tags-for-resource`

Returns a list of the tags assigned to the specified resource.

### `coily aws ssm modify-document-permission`

Shares a Amazon Web Services Systems Manager document (SSM document)publicly or privately.

### `coily aws ssm put-compliance-items`

Registers a compliance type and other compliance details on a designated resource.

### `coily aws ssm put-inventory`

Bulk update custom inventory items on one or more managed nodes.

### `coily aws ssm put-parameter`

Create or update a parameter in Parameter Store.

### `coily aws ssm put-resource-policy`

Creates or updates a Systems Manager resource policy.

### `coily aws ssm register-default-patch-baseline`

Defines the default patch baseline for the relevant operating system.

### `coily aws ssm register-patch-baseline-for-patch-group`

Registers a patch baseline for a patch group.

### `coily aws ssm register-target-with-maintenance-window`

Registers a target with a maintenance window.

### `coily aws ssm register-task-with-maintenance-window`

Adds a new task to a maintenance window.

### `coily aws ssm remove-tags-from-resource`

Removes tag keys from the specified resource.

### `coily aws ssm reset-service-setting`

ServiceSetting is an account-level setting for an Amazon Web Services service.

### `coily aws ssm resume-session`

Reconnects a session to a managed node after it has been disconnected.

### `coily aws ssm send-automation-signal`

Sends a signal to an Automation execution to change the current behavior or status of the execution.

### `coily aws ssm send-command`

Runs commands on one or more managed nodes.

### `coily aws ssm start-associations-once`

Runs an association immediately and only one time.

### `coily aws ssm start-automation-execution`

Initiates execution of an Automation runbook.

### `coily aws ssm start-change-request-execution`

Creates a change request for Change Manager.

### `coily aws ssm start-execution-preview`

Initiates the process of creating a preview showing the effects that running a specified Automation runbook would have on the targeted resources.

### `coily aws ssm start-session`

Initiates a connection to a target (for example, a managed node) for a Session Manager session.

### `coily aws ssm stop-automation-execution`

Stop an Automation that is currently running.

### `coily aws ssm terminate-session`

Permanently ends a session and closes the data connection between the Session Manager client and SSM Agent on the managed node.

### `coily aws ssm unlabel-parameter-version`

Remove a label or labels from a parameter.

### `coily aws ssm update-association`

Updates an association.

### `coily aws ssm update-association-status`

Updates the status of the Amazon Web Services Systems Manager document (SSM document) associated with the specified managed node.

### `coily aws ssm update-document`

Updates one or more values for an SSM document.

### `coily aws ssm update-document-default-version`

Set the default version of a document.

### `coily aws ssm update-document-metadata`

Updates information related to approval reviews for a specific version of a change template in Change Manager.

### `coily aws ssm update-maintenance-window`

Updates an existing maintenance window.

### `coily aws ssm update-maintenance-window-target`

Modifies the target of an existing maintenance window.

### `coily aws ssm update-maintenance-window-task`

Modifies a task assigned to a maintenance window.

### `coily aws ssm update-managed-instance-role`

Changes the Identity and Access Management (IAM) role that is assigned to the on-premises server, edge device, or virtual machines (VM).

### `coily aws ssm update-ops-item`

Edit or change an OpsItem.

### `coily aws ssm update-ops-metadata`

Amazon Web Services Systems Manager calls this API operation when you edit OpsMetadata in Application Manager.

### `coily aws ssm update-patch-baseline`

Modifies an existing patch baseline.

### `coily aws ssm update-resource-data-sync`

Update a resource data sync.

### `coily aws ssm update-service-setting`

ServiceSetting is an account-level setting for an Amazon Web Services service.

### `coily aws ssm wait` (group)

Wait until a particular condition is satisfied.

Subcommands: `command-executed`

### `coily aws ssm wait command-executed`

Wait until JMESPath query Status returns Success when polling with get-command-invocation.

## `coily aws sts`

### `coily aws sts` (group)

Security Token Service (STS) enables you to request temporary, limited-privilege credentials for users.

Subcommands: `assume-role`, `assume-role-with-saml`, `assume-role-with-web-identity`, `assume-root`, `decode-authorization-message`, `get-access-key-info`, `get-caller-identity`, `get-federation-token`, `get-session-token`

### `coily aws sts assume-role`

Returns a set of temporary security credentials that you can use to access Amazon Web Services resources.

### `coily aws sts assume-role-with-saml`

Returns a set of temporary security credentials for users who have been authenticated via a SAML authentication response.

### `coily aws sts assume-role-with-web-identity`

Returns a set of temporary security credentials for users who have been authenticated in a mobile or web application with a web identity provider.

### `coily aws sts assume-root`

Returns a set of short term credentials you can use to perform privileged tasks on a member account in your organization.

### `coily aws sts decode-authorization-message`

Decodes additional information about the authorization status of a request from an encoded message returned in response to an Amazon Web Services request.

### `coily aws sts get-access-key-info`

Returns the account identifier for the specified access key ID.

### `coily aws sts get-caller-identity`

Returns details about the IAM user or role whose credentials are used to call the operation.

### `coily aws sts get-federation-token`

Returns a set of temporary security credentials (consisting of an access key ID, a secret access key, and a security token) for a user.

### `coily aws sts get-session-token`

Returns a set of temporary credentials for an Amazon Web Services account or IAM user.

