# AGENTS.md — openshift/aws-karpenter-provider-aws

This file provides AI-specific guidance for working in the OpenShift downstream fork of the [Karpenter AWS Provider](https://github.com/aws/karpenter-provider-aws). For contribution guidelines, see [CONTRIBUTING_OPENSHIFT.md](CONTRIBUTING_OPENSHIFT.md).

## Project Overview

This repo is the **AWS provider for Karpenter** — a Kubernetes node autoscaler that provisions compute resources based on pod scheduling constraints. This is the OpenShift downstream fork of [aws/karpenter-provider-aws](https://github.com/aws/karpenter-provider-aws).

Karpenter watches for unschedulable pods and provisions instances to run them, selecting instance types based on pod requirements (CPU, memory, GPU, architecture, zones, etc.). It also handles node lifecycle events like spot interruptions, capacity rebalancing, and consolidation to reduce costs.

This provider implements AWS-specific logic: EC2 instance selection, AMI resolution, launch template management, subnet/security group discovery, pricing data, and interruption queue handling (SQS). The core scheduling and node lifecycle logic lives in [openshift/kubernetes-sigs-karpenter](https://github.com/openshift/kubernetes-sigs-karpenter).

The Karpenter controller from this repo is deployed and managed by the [karpenter-operator](https://github.com/openshift/karpenter-operator).

### HyperShift Integration

On HyperShift, the `OpenshiftEC2NodeClass` CRD from [openshift/karpenter-operator](https://github.com/openshift/karpenter-operator) is a wrapper over the `EC2NodeClass` CRD that gets shipped from this repository. The karpenter-operator deploys both the `OpenshiftEC2NodeClass` and `EC2NodeClass` to the hosted cluster.

The `OpenshiftEC2NodeClass` extends (not replaces) the upstream `EC2NodeClass` by preserving all standard AWS configuration fields (subnet selectors, security group selectors, instance profiles, block device mappings, etc.) while adding OpenShift-specific capabilities:

- **Version Resolution**: Resolves OpenShift semantic versions (e.g., "4.20.1") through the Cincinnati API and tracks them in status as fully qualified release images. When `spec.version` is unset, nodes inherit the HostedControlPlane's release image.
- **Kubelet Configuration**: Injects node-level kubelet settings through MachineConfig for OpenShift-specific node customization.
- **Version Compatibility**: Validates version skew constraints via the `SupportedVersionSkew` condition (nodes cannot exceed control plane version, with n-3 minor version tolerance for 4.y releases).
- **HyperShift Token Integration**: Links token secrets to HyperShift NodePool instances via the `hypershift.openshift.io/nodePool` annotation, critical for multi-pool HostedCluster deployments.

The `Ready` condition is computed atomically by the EC2 node class controller, combining upstream `EC2NodeClass` readiness signals with the OpenShift-specific `VersionResolved` condition status.

### Single Binary

This repo produces one main binary:

| Binary | Source | Purpose |
|--------|--------|---------|
| `controller` | `cmd/controller/main.go` | The Karpenter AWS provider controller. Integrates with core Karpenter to provision EC2 instances, manage EC2NodeClass resources, handle spot interruptions, and maintain AWS resource state. |

### CRDs

| CRD | API Group | Purpose |
|-----|-----------|---------|
| EC2NodeClass | `karpenter.k8s.aws/v1` | Defines AWS-specific node configuration: AMI selectors, instance profile, user data, security groups, subnets, block device mappings, and metadata options. Referenced by NodePool (from core Karpenter). |

## Repository Structure

```text
cmd/
  controller/           # Main controller binary entry point
pkg/
  apis/
    v1/                 # EC2NodeClass CRD type definitions, validation, defaults
  cloudprovider/        # CloudProvider interface implementation for AWS
  controllers/
    interruption/       # Handles spot interruption events via SQS
    nodeclass/          # Reconciles EC2NodeClass, manages readiness and status
    nodeclaim/          # AWS-specific NodeClaim lifecycle (drift detection, termination)
    capacityreservation/ # On-demand capacity reservation tracking
    arczonalshift/      # AWS ARC zonal shift awareness
  providers/
    amifamily/          # AMI selection logic for AL2, Bottlerocket, Ubuntu, Windows, Custom
    instance/           # EC2 instance provisioning, termination, launch template creation
    instancetype/       # EC2 instance type discovery and offerings cache
    pricing/            # EC2 on-demand and spot pricing data
    subnet/             # VPC subnet discovery and selection
    securitygroup/      # Security group discovery
    launchtemplate/     # EC2 launch template management
    instanceprofile/    # IAM instance profile resolution
    sqs/                # SQS interruption queue polling
  operator/             # Operator setup, dependency injection
  fake/                 # Fake AWS clients for testing
test/
  suites/               # E2E test suites organized by feature area
  pkg/                  # Shared test utilities and environment setup
charts/
  karpenter/            # Helm chart for deploying Karpenter (upstream)
designs/                # Design docs for features and architectural decisions
hack/                   # Build and development scripts
openshift/
  Containerfile.rhel    # RHEL-based container build file for OpenShift
```

## Upstream / Downstream Relationship

This repo tracks upstream `aws/karpenter-provider-aws` on the `main` branch. The OpenShift downstream fork carries patches for RHEL support, OpenShift-specific configuration, and integration with the karpenter-operator.

### What is downstream-only

These files exist only in the downstream fork:

- `OWNERS` — OpenShift approver list
- `OWNERS_ALIASES` — Aliases for approver groups
- `.ci-operator.yaml` — OpenShift CI build root config
- `openshift/Containerfile.rhel` — RHEL-based container build
- `CONTRIBUTING_OPENSHIFT.md` — This downstream contributing guide
- `AGENTS.md` — This file
- `.coderabbit.yaml` — CodeRabbit configuration

### What is upstream

Everything else. In particular: the core provider logic in `pkg/`, the controllers, the EC2NodeClass API, the AMI family implementations, the pricing and instance type providers, the Helm charts, and the `.github/` workflows (which are not used in our CI but are kept for upstream compatibility).

Avoid modifying upstream files unless necessary. Downstream-only patches increase the maintenance burden during rebase cycles. Instead, contribute features and fixes upstream where possible.

### Key Carry Patches

The downstream fork maintains several categories of carry patches that are re-applied during each rebase:

1. **UserData Drift Detection** (commit [882c076](https://github.com/openshift/aws-karpenter-provider-aws/commit/882c076), PR [#21](https://github.com/openshift/aws-karpenter-provider-aws/pull/21)): The karpenter-operator passes a rotating token to the EC2NodeClass userData field as part of the Ignition payload. This token changes frequently, causing the hash to change and triggering drift unintentionally. This patch modifies the hash calculation to extract and use only the `TargetConfigVersionHash` HTTP header value from the Ignition config (instead of hashing the entire userData field with its rotating token). The hash is unique and acts as a trigger for drift rollout, similar to its usage in HyperShift's NodePool API. If the userData is not a valid Ignition config, this special handling is bypassed and the raw userData is hashed normally.

2. **Version Injection** (commit [f636c63](https://github.com/openshift/aws-karpenter-provider-aws/commit/f636c63), PR [#25](https://github.com/openshift/aws-karpenter-provider-aws/pull/25)): OpenShift doesn't maintain git tags like upstream does, so the build cannot use `git describe --tags` to determine the version. This patch adds a manually-maintained `OPENSHIFT_AWS_KARPENTER_VERSION` variable in the Makefile that overrides the version injected into `sigs.k8s.io/karpenter/pkg/operator.Version` at build time. This version must be manually updated during each rebase cycle to match the upstream release being rebased onto.

3. **Vendor Directory Handling** (commit [7ac760c](https://github.com/openshift/aws-karpenter-provider-aws/commit/7ac760c)): The downstream build process requires a vendored `vendor/` directory for reproducible builds in the OpenShift CI environment. Upstream does not vendor dependencies. This patch fixes the `.gitignore` to only ignore `go.work` and `go.work.sum` at the repository root (changing patterns from `go.work` to `/go.work`), preventing vendored dependencies that include files with the same name from being incorrectly ignored. When updating dependencies, run `go mod tidy && go mod vendor` and commit vendor changes in a separate commit.

### Rebase Cycle

Periodically, the `main` branch is rebased onto a newer upstream release. During a rebase:
- `UPSTREAM: <drop>:` commits are discarded
- `UPSTREAM: <carry>:` commits are re-applied
- `UPSTREAM: 1234:` commits are dropped if upstream PR 1234 is now included, otherwise re-applied

The current tracked upstream version is defined in the `Makefile` as `OPENSHIFT_AWS_KARPENTER_VERSION`.

## Architecture: What Is Not Obvious

### Integration with Core Karpenter

The AWS provider implements the `cloudprovider.CloudProvider` interface from `sigs.k8s.io/karpenter`. The core Karpenter controllers (in kubernetes-sigs-karpenter) handle:
- NodePool and NodeClaim lifecycle (scheduling, provisioning, deprovisioning)
- Disruption budgets and consolidation
- Drift detection and node expiration

The AWS provider handles:
- EC2 instance creation and termination
- AMI selection based on EC2NodeClass configuration
- Launch template generation
- Spot interruption handling via SQS
- Pricing and instance type offerings

### EC2NodeClass Reconciliation

EC2NodeClass resources are reconciled by the `nodeclass` controller. The controller:
1. Discovers subnets and security groups matching the selectors
2. Resolves AMIs using the AMI selectors
3. Validates instance profile and user data
4. Updates the status with resolved resources and readiness condition

NodePools reference EC2NodeClasses via `spec.template.spec.nodeClassRef`. When a pod needs a node, core Karpenter creates a NodeClaim, and the AWS provider uses the referenced EC2NodeClass to provision an EC2 instance.

### Interruption Handling

The `interruption` controller polls an SQS queue for spot interruption notices, scheduled maintenance events, and instance rebalance recommendations. When an event is detected:
1. The controller adds a taint to the affected node
2. Core Karpenter drains the node
3. The provider terminates the EC2 instance

The SQS queue is configured via EventBridge rules in the cluster's AWS account.

### Instance Type Selection

The `instancetype` provider discovers all available EC2 instance types in the region and builds an offerings cache. When provisioning a node, the provider:
1. Reads the NodeClaim's scheduling requirements (CPU, memory, GPU, architecture, etc.)
2. Queries the offerings cache for compatible instance types
3. Applies pricing data (on-demand and spot)
4. Returns a ranked list to the core scheduler

The cache is refreshed periodically and on EC2NodeClass changes.

### AMI Families

The `amifamily` package contains implementations for each supported OS:
- **AL2** (Amazon Linux 2) — Default, uses SSM Parameter Store for AMI discovery
- **Bottlerocket** — Container-optimized OS
- **Ubuntu** — Canonical Ubuntu images
- **Windows** — Windows Server variants
- **Custom** — OpenShift-specific AMI family for RHEL-based nodes

Each family provides:
- AMI selection logic
- User data generation (cloud-init or custom)
- Instance metadata options
- Block device mappings

## Common Pitfalls

1. **Add a carry prefix to every commit.** Every non-upstream commit must use `UPSTREAM: <carry>:`, `UPSTREAM: <drop>:`, or `UPSTREAM: 1234:`. Only commits merged from upstream don't require the prefix.

2. **Do not hand-edit generated files.** Files matching `zz_generated*` and CRD manifests in `charts/karpenter/crds/` are generated. Run `make codegen` to regenerate deepcopy methods.

3. **Do not modify CRD manifests directly.** The CRDs are generated from kubebuilder markers in the Go types. Edit `pkg/apis/v1/*.go` and regenerate, rather than editing YAML.

4. **Do not add AWS-specific features to core Karpenter.** If a feature applies to all cloud providers (AWS, Azure, GCP), it belongs in `kubernetes-sigs-karpenter`. AWS-specific features (EC2 metadata, launch templates, AMI selection) belong here.

5. **Do not use GitHub Actions.** The `.github/workflows/` directory is from upstream and is not used in our CI. Our CI runs through OpenShift CI (Prow). The CI config is in [openshift/release](https://github.com/openshift/release/tree/master/ci-operator/config/openshift/aws-karpenter-provider-aws).

6. **Watch for AWS API rate limits.** The provider makes many AWS API calls (EC2, SSM, Pricing, SQS). Use the caching providers (`pkg/cache/`) to avoid hitting rate limits. Do not add direct AWS SDK calls in controllers without caching.

7. **Instance type offerings change.** EC2 instance types are not static. An instance type available in one zone may not be available in another, and offerings change over time. Always use the `instancetype` provider's offerings cache rather than hardcoding instance types.

8. **Spot pricing is volatile.** Spot prices change frequently and vary by zone. The `pricing` provider maintains a cache updated every few minutes. Do not cache spot prices in memory without refresh logic.

9. **EC2NodeClass status is eventually consistent.** Subnet and security group discovery is asynchronous. A newly created EC2NodeClass may not be ready immediately. Controllers should check the `Ready` condition in the status.

10. **Launch templates are versioned.** The provider creates a new launch template version on each configuration change. Do not assume launch template IDs are stable — use the version tracking in the `launchtemplate` provider.

11. **User data must be base64-encoded.** EC2 expects user data in base64. The `amifamily` implementations handle this automatically, but if you're modifying user data generation, ensure proper encoding.

## Human-in-the-Loop Triggers

Stop and consult a human before:

- **Modifying EC2NodeClass API types** (`pkg/apis/v1/`) — API changes have compatibility implications and may require coordinated changes in the karpenter-operator repo, hypershift repo, and official Red Hat documentation
- **Changing IAM permissions** — The controller runs with IAM permissions defined by the karpenter-operator. Adding new AWS API calls may require IAM policy updates.
- **Adding or removing AMI families** — AMI family changes affect the product surface area and may require documentation updates
- **Modifying interruption handling** — Changes to the interruption controller can affect workload availability during spot interruptions
- **Changing pricing or instance type discovery** — These providers are critical for cost optimization and provisioning decisions
- **Modifying the upstream/downstream boundary** — Any change to which files are carried vs. upstream
- **Rebase-related decisions** — Whether a carry patch is still needed, whether to drop or keep

## Paired Changes

These files must be updated together:

| If you change... | Also update... |
|-----------------|----------------|
| API types in `pkg/apis/v1/` | Run `make codegen` to regenerate deepcopy methods |
| EC2NodeClass kubebuilder markers | CRDs are regenerated upstream — verify markers are correct for validation |
| AMI family implementations | Update corresponding tests in `pkg/providers/amifamily/*_test.go` |
| AWS provider interface | Update fake implementations in `pkg/fake/` for testing |
| Instance type offerings | Update tests that depend on specific instance types |
| Pricing data structures | Update tests that mock pricing responses |
| `go.mod` dependencies | Run `go mod tidy && go mod vendor` and commit vendor changes separately |

## Further Reading

- [CONTRIBUTING_OPENSHIFT.md](CONTRIBUTING_OPENSHIFT.md) — Downstream contribution workflow, PR commands, test expectations
- [Karpenter Documentation](https://karpenter.sh/docs/) — Upstream Karpenter concepts and user guide
- [AWS Provider Documentation](https://karpenter.sh/docs/cloud-providers/aws/) — AWS-specific configuration and features
- [EC2NodeClass API Reference](https://karpenter.sh/docs/concepts/nodeclasses/) — EC2NodeClass field reference
- [Design Docs](designs/) — Architectural decision records and feature designs
