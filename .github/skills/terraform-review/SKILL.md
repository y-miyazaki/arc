---
name: terraform-review
description: >-
  Reviews Terraform configurations for design decisions, security patterns, and best practices.
  Checks module structure, variable design, tagging, state management, and compliance requiring human judgment.
  Use when reviewing Terraform pull requests, evaluating infrastructure architecture, or assessing security of IaC code.
license: Apache-2.0
metadata:
  author: y-miyazaki
  version: "1.0.0"
---

## Purpose

Conducts code review of Terraform configurations checking design decisions and best practices requiring human judgment.

Manual code review guidance for Terraform configurations, covering design decisions and patterns requiring human judgment.

## When to Use This Skill

Recommended usage:

- Performing code reviews on Terraform pull requests
- Checking Terraform configurations before merging
- Ensuring security and compliance standards
- Validating best practices adherence
- Architecture and design review

## Input Specification

This skill expects:

- Terraform files (required) - `.tf` files in the PR
- PR description and linked issues (required) - Context for understanding changes
- Related documentation (optional) - README or Terraform documentation updates

Format:

- Terraform files: Target Terraform files under review
- PR context: Markdown text describing purpose and changes
- Optional validation context: Summary of validation outcomes when provided
- Environment: Specify target environment (dev/staging/production)

## Output Specification

**Output format (MANDATORY)** - Use this exact structure:

- ## Checks Summary section: Total/Passed/Failed/Deferred counts
- ## Checks (Failed/Deferred Only) section: Show only ❌ and ⊘ items in checklist order
- ## Issues section: Numbered list with full details for each failed or deferred item
- Keep full evaluation data for all checks internally using fixed ItemIDs from references/common-checklist.md
- If there are no failed or deferred checks: output "No failed or deferred checks" in Checks and "No issues found" in Issues

See references/common-output-format.md for detailed format specification and examples.

## Execution Scope

**How to use this skill**:

- This skill provides manual review guidance requiring human/AI judgment
- Reviewer reads Terraform configurations and systematically applies review checklist items from [references/common-checklist.md](references/common-checklist.md)
- **Boundary**:
  - Focus only on checks that require human/AI judgment
  - Treat syntax/lint/security automation as out of scope for this review skill
  - Do not run terraform-validation from this review skill
- **When to use**: For design decisions, security patterns, and best practices requiring judgment

**What this skill does**:

- Review design decisions and architecture patterns requiring human judgment
- Check security patterns (encryption, IAM, resource policies, VPC)
- Validate module structure and responsibility separation
- Assess variable design (type safety, defaults, validation)
- Review output design and sensitive data handling
- Check tagging consistency and requirements
- Verify naming conventions and documentation completeness
- Assess monitoring, alerting, and logging patterns
- Review state management and backend configuration
- Check dependency ordering and implicit dependencies
- Evaluate design patterns and anti-patterns

What this skill does NOT do (Out of Scope):

- Check syntax errors (use terraform validate for that)
- Run linting (use tflint for that)
- Perform security scanning (use trivy config for that)
- Execute terraform fmt/validate/tflint/trivy commands from this review skill
- Execute terraform plan or apply
- Modify Terraform files automatically
- Approve or merge pull requests
- Review non-Terraform files in the PR
- AWS-specific checks for non-AWS environments

## Constraints

Prerequisites:

- PR context and Terraform files are available
- PR description and context must be available
- Reviewer must have access to reference documentation
- AWS-based Terraform (other providers may need adjustment)

Limitations:

- Review focuses on design patterns and best practices, not syntax
- Cannot validate actual AWS resource creation or behavior
- Assumes familiarity with Terraform best practices
- Reference documentation required for detailed category checks
- AWS-specific recommendations may need adjustment for other cloud providers

## Failure Behavior

Error handling:

- Missing PR context: Request PR description and linked issues, cannot proceed without context
- Invalid Terraform syntax: Record as validation concern and continue reviewing judgment-based items when possible
- Inaccessible reference files: Output warning, proceed with available knowledge only
- Ambiguous design decision: Flag as potential issue with recommendation to clarify intent or add comments

Error reporting format:

- Clear indication of blocking issues vs. recommendations
- Specific file paths and line numbers for all issues
- Code examples for recommended fixes
- References to patterns in reference documentation

## Reference Files Guide

When using this skill with an agent, reference the following files via @-mention for detailed guidance:

**Standard Components** (always read):

- [common-checklist.md](references/common-checklist.md) - Complete review checklist with ItemIDs
- [common-output-format.md](references/common-output-format.md) - Report format specification

**Category Details** (read when reviewing related code):

- [category-compliance.md](references/category-compliance.md) - Read when reviewing OPA policies or compliance standards
- [category-cost.md](references/category-cost.md) - Read when reviewing resource sizing, lifecycle policies, or cost optimization
- [category-data-sources.md](references/category-data-sources.md) - Read when reviewing data source usage or imports
- [category-dependency.md](references/category-dependency.md) - Read when reviewing depends_on or implicit dependencies
- [category-events.md](references/category-events.md) - Read when reviewing monitoring, alerting, or logging patterns
- [category-global.md](references/category-global.md) - Read when reviewing module usage, secrets, or for_each patterns
- [category-migration.md](references/category-migration.md) - Read when reviewing import strategies or state migration
- [category-modules.md](references/category-modules.md) - Read when reviewing module structure or provider versions
- [category-naming.md](references/category-naming.md) - Read when reviewing naming conventions or documentation
- [category-outputs.md](references/category-outputs.md) - Read when reviewing output design or sensitive data handling
- [category-patterns.md](references/category-patterns.md) - Read when reviewing design patterns or anti-patterns
- [category-performance.md](references/category-performance.md) - Read when reviewing API limits, parallel execution, or large-scale configs
- [category-security.md](references/category-security.md) - Read when reviewing encryption, IAM, resource policies, or VPC security
- [category-state.md](references/category-state.md) - Read when reviewing state management or backend configuration
- [category-tagging.md](references/category-tagging.md) - Read when reviewing tag consistency or requirements
- [category-tfvars.md](references/category-tfvars.md) - Read when reviewing tfvars, secret handling, or environment separation
- [category-variables.md](references/category-variables.md) - Read when reviewing variable types, defaults, or validation
- [category-versioning.md](references/category-versioning.md) - Read when reviewing versioning strategies

## Workflow

### Step 1: Understand Context

Before starting the review:

- Read the PR description and linked issues
- Understand the purpose of the changes
- Check if this is new infrastructure or modification
- Verify which environment (dev/staging/production) is affected

### Step 2: Confirm Review Boundary

Focus on manual checks only:

- Design and architecture decisions
- Security and policy patterns requiring judgment
- Maintainability and operability concerns

Do not execute validation tools in this review workflow.

### Step 3: Systematic Review

Review categories systematically based on the changes. Use the reference documentation for detailed checks in each category.

### Step 4: Report Issues

Report issues following the Output Format below, using Checks Summary + Failed/Deferred-only Checks + full Issues details.

## Output Format

Review results must be output in structured format:

### Output Elements

1. **Checks** (Review items checklist)
   - Display `Checks Summary` with Total/Passed/Failed/Deferred counts
   - Display `Checks (Failed/Deferred Only)` for ❌ and ⊘ items only
   - Keep ItemIDs fixed and sorted in checklist order
   - If there are no failed or deferred checks, output "No failed or deferred checks"

2. **Issues** (Detected problems)
   - Display details for each failed or deferred item
   - Numbered list format for each problem
   - Each issue includes:
     - Item ID + Item Name
     - File: file path and line number
     - Problem: Description of the issue
     - Impact: Scope and severity
     - Recommendation: Specific fix suggestion

### Output Format Example

```markdown
# Terraform Code Review Result

## Checks Summary

- Total checks: 42
- Passed: 41
- Failed: 1
- Deferred: 0

## Checks (Failed/Deferred Only)

- G-02 Secret Hardcoding Prohibition: ❌ Fail

## Issues

**No issues found** (if all checks pass and there are no deferred checks)

**OR**

1. G-02: Secret Hardcoding Prohibition
   - File: `terraform/modules/api/main.tf` L45
   - Problem: Hardcoded password detected
   - Impact: Security risk, secrets in Git history
   - Recommendation: Use variable or AWS Secrets Manager

2. SEC-03: Resource Policy with Condition
   - File: `terraform/base/s3.tf` L12-15
   - Problem: S3 bucket policy missing condition clause
   - Impact: Potential unintended access permissions
   - Recommendation: Add `aws:SecureTransport` condition
```

## Available Review Categories

Review categories are organized by domain. Claude will read the relevant category file(s) based on the code being reviewed.

**Global & Base**: Module usage, secrets, versioning, for_each patterns → [references/category-global.md](references/category-global.md)
**Modules**: Module structure, provider versions, responsibility → [references/category-modules.md](references/category-modules.md)
**Variables**: Type safety, defaults, descriptions, validation → [references/category-variables.md](references/category-variables.md)
**Outputs**: Description requirements, sensitive data → [references/category-outputs.md](references/category-outputs.md)
**Tfvars**: Secret handling, environment separation → [references/category-tfvars.md](references/category-tfvars.md)
**Security**: Encryption, IAM, resource policies, VPC → [references/category-security.md](references/category-security.md)
**Tagging**: Tag consistency and requirements → [references/category-tagging.md](references/category-tagging.md)
**Events & Observability**: Monitoring, alerting, logging → [references/category-events.md](references/category-events.md)
**Versioning**: Immutable versioning strategies → [references/category-versioning.md](references/category-versioning.md)
**Naming & Documentation**: Naming conventions, comments → [references/category-naming.md](references/category-naming.md)
**Patterns**: Design patterns and anti-patterns → [references/category-patterns.md](references/category-patterns.md)
**State & Backend**: State management, backend configuration → [references/category-state.md](references/category-state.md)
**Compliance & Policy**: OPA policies, compliance standards → [references/category-compliance.md](references/category-compliance.md)
**Cost Optimization**: Resource sizing, lifecycle policies → [references/category-cost.md](references/category-cost.md)
**Performance & Limits**: API limits, parallel exec, large-scale → [references/category-performance.md](references/category-performance.md)
**Migration & Refactoring**: Import strategies, state migration → [references/category-migration.md](references/category-migration.md)
**Dependency & Ordering**: depends_on, implicit dependencies → [references/category-dependency.md](references/category-dependency.md)
**Data Sources & Imports**: Data source usage, imports → [references/category-data-sources.md](references/category-data-sources.md)

## Best Practices

When performing code reviews:

- **Constructive and specific**: Include code examples and reference links
- **Context-aware**: Understand PR purpose and requirements, consider tradeoffs
- **Clear priorities**: Distinguish between "must fix" and "nice to have"
- **Leverage MCP tools**: Use context7 for module docs, serena for project structure
- **Prioritize automation**: Avoid excessive focus on syntax errors and terraform fmt/validate/tflint/trivy
- **Prevent security oversights**: Pay special attention to SEC-\* items
- **Note AWS context**: AWS-specific checks may need adjustment for other cloud environments
