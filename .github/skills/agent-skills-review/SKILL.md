---
name: agent-skills-review
description: >-
  Reviews SKILL.md files for structural requirements, quality standards, and design patterns.
  Checks specification completeness, implementation feasibility, and consistency with established patterns.
  Use when creating new skills, reviewing skill pull requests, or auditing skill quality.
license: Apache-2.0
metadata:
  author: y-miyazaki
  version: "1.0.0"
---

## Purpose

Reviews SKILL.md files for structural requirements, quality standards, and design patterns to ensure skill quality, specification completeness, implementation feasibility, and consistency with established review skill patterns.

## When to Use This Skill

**Recommended usage**:

- New SKILL.md creation - quality assurance
- SKILL.md modification/update - change quality validation
- Pull request review - for agent skills-related PRs
- Skill governance audit - batch quality check for multiple skills
- Agent Skills standardization initiatives

## Input Specification

This skill expects:

- **SKILL.md file** (required) - Target file to review (.github/skills/\*/SKILL.md)
- **agent-skills.instructions.md** (required) - Validation criteria reference (included in this skill's references)
- **PR description and skill overview context** (recommended) - Understanding of skill purpose and target background

Format:

- SKILL.md: Markdown with YAML front matter (name, description, license)
- Reference files: Category-specific Markdown (detailed criteria)

## Output Specification

**Output format (MANDATORY)** - Use this exact structure:

- ## Checks Summary section: Total/Passed/Failed/Deferred counts
- ## Checks (Failed/Deferred Only) section: Show only ❌ and ⊘ items in checklist order
- ## Issues section: Numbered list with full details for each failed or deferred item
- Keep full evaluation data for all checks internally using fixed ItemIDs from [references/common-checklist.md](references/common-checklist.md)
- If there are no failed or deferred checks: output "No failed or deferred checks" in Checks and "No issues found" in Issues

See [references/common-output-format.md](references/common-output-format.md) for detailed format specification and examples.

## Execution Scope

**How to use this skill**:

- This skill provides manual review guidance requiring human/AI judgment
- Reviewer reads SKILL.md files and systematically applies review checklist items from [references/common-checklist.md](references/common-checklist.md)
- **Boundary**:
  - Focus only on checks that require human/AI judgment
  - Treat deterministic validation automation as out of scope for this review skill
  - Do not run yamllint or scripts/validate.sh from this review skill
- **When to use**: Review .github/skills/\*/SKILL.md files for quality, specification completeness, and design pattern compliance

**What this skill does**:

1. **Structure Validation**: Verify SKILL.md contains required sections, YAML fields, and reference file header consistency
   - S-01: Section order and completeness
   - S-03: Reference file header level standards
   - YAML frontmatter fields
2. **Manual Quality Review** (systematic evaluation via human/AI judgment)
   - Q-01: Output is Truly Structured
   - Q-02: Scope Boundaries
   - Q-03: Execution Determinism
   - Q-04: Input/Output Specificity
   - Q-05: Constraints Clarity
   - Q-06: No Implicit Inference
   - P-01: Design Pattern Compliance
   - P-02: Output Format Compliance
3. **Report Generation**
   - Checks Summary section: Total/Passed/Failed/Deferred counts
   - Checks (Failed/Deferred Only) section: Show only ❌ and ⊘ items in checklist order
   - Issues section: Failed or deferred items only with full details
   - Full evaluation data for all checks is retained internally using fixed ItemIDs

What this skill does NOT do (Out of Scope):

- YAML/Markdown syntax errors (use yamllint, markdownlint for that)
- Automated file modifications
- PR merge approval
- Skill execution or functionality testing
- Reference file syntax validation
- Execute yamllint or scripts/validate.sh from this review skill

**Design Philosophy**:

This skill embodies the philosophy it recommends by implementing it in practice:

- Deterministic checks (structure, metrics, file existence) → Automated in scripts/ for objective verification
- Judgment-based checks (semantic evaluation, design decisions) → Manual review for human/AI strengths
- Result: Token efficiency + verification quality combined

Implementation: deterministic checks are delegated to validation tooling, and this review workflow focuses on judgment-based evaluation, achieving context optimization and verification credibility.

**Key principles**:

- **Meta-Pattern**: This skill itself demonstrates the philosophy it recommends (deterministic → automation, judgment-based → manual). Serves as a model for other skill design.
- **Reference-driven**: Detailed check criteria defined in references/\*.md files. Load reference files only when reviewing specific categories.
- **Two-Phase Approach**: Structure validation (automated via scripts) followed by quality review (manual evaluation using reference files).

## Constraints

**Prerequisites**:

1. SKILL.md must have YAML front matter (name, description, license)
2. Target SKILL.md and required references are available
3. Understanding of role boundaries between validation and review workflows
4. Understanding of agent-skills.instructions.md Structural Requirements required
5. Access to reference files (structure.md, quality.md, patterns.md) available

**Limitations**:

- Scope: `.github/skills/*/SKILL.md` files only (other formats out of scope)
- Judgment-based checks (Q-01–Q-06, P-01–P-02) require systematic review (cannot be fully automated)
- Recommendations must be concrete/specific (no vague expressions like "should be improved")

## Failure Behavior

**Error Handling**:

- Missing required section → **CRITICAL** severity (structural violation, cannot merge)
- Missing YAML frontmatter field → **CRITICAL** severity
- Non-structured output format → **CRITICAL** severity
- Ambiguous reasoning or expressions → **IMPORTANT** severity
- Minor design improvements → **ENHANCEMENT** level

**Reporting Content**:

- Number of failed checks, breakdown by category
- For each issue: CheckID, category, problem description, impact, concrete recommendation
- Summary of automation failure details if applicable

## Reference Files Guide

When using this skill with an agent, reference the following files via @-mention for detailed guidance:

**Standard Components** (always read):

- [common-checklist.md](references/common-checklist.md) - Complete review checklist (S-01 through P-02)
- [common-output-format.md](references/common-output-format.md) - Report format specification

**Category Details** (read when reviewing related aspects):

- [category-patterns.md](references/category-patterns.md) - Read when checking design pattern compliance (P-01, P-02)
- [category-quality.md](references/category-quality.md) - Read when checking quality standards (Q-01 through Q-06)
- [category-structure.md](references/category-structure.md) - Read when checking structural requirements (S-01, S-02)

## Workflow

### Step 1: Context Understanding

- Understand purpose, scope, background from PR/skill description
- Review agent-skills.instructions.md requirements

### Step 2: Confirm Review Boundary

Focus on manual checks only:

- Structure clarity and instruction quality
- Design pattern compliance and specificity
- Deterministic workflow documentation quality

Do not execute validation tools in this review workflow.

### Step 3: Systematic Manual Review

- Verify Q-01–Q-06 (quality checks) systematically using references/quality.md
- Verify P-01–P-02 (pattern checks) systematically using references/patterns.md
- Mark each as ✅ or ❌; for failures, provide concrete reason + recommendation

### Step 4: Report Generation

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
     - CheckID + Item Name
     - File: file path and line number
     - Problem: Description of the issue
     - Impact: Scope and severity
     - Recommendation: Specific fix suggestion with code/config examples

### Output Format Example

```markdown
# Agent Skills Review Result

## Checks Summary

- Total checks: 10
- Passed: 8
- Failed: 2
- Deferred: 0

## Checks (Failed/Deferred Only)

- S-01 Section Order and Completeness: ❌ Fail
- Q-02 Scope Boundaries: ❌ Fail

## Issues

**No issues found** (if all checks pass and there are no deferred checks)

**OR**

1. S-01: Section Order and Completeness
   - File: `.github/skills/example-skill/SKILL.md`
   - Problem: Missing "Execution Scope" section
   - Impact: Skill cannot clearly communicate what it does and does not do
   - Recommendation: Add `## Execution Scope` section between Output Specification and Constraints

2. Q-02: Scope Boundaries
   - File: `.github/skills/example-skill/SKILL.md` L45
   - Problem: Out of Scope section missing, boundary between this skill and related validation skill unclear
   - Impact: Agent may attempt to run validation tools during review
   - Recommendation: Add explicit "What this skill does NOT do" list with cross-references to related skills
```

## Available Review Categories

Review categories are organized by domain. Claude will read the relevant category file(s) based on the SKILL.md being reviewed.

**Checklist**: Complete review checklist → [references/common-checklist.md](references/common-checklist.md)
**Output Format Reference**: Canonical report template → [references/common-output-format.md](references/common-output-format.md)

**Structure**: Section order, completeness, YAML frontmatter, reference file headers → [references/category-structure.md](references/category-structure.md)
**Quality**: Output structure, scope boundaries, execution determinism, I/O specificity, constraints clarity, no implicit inference → [references/category-quality.md](references/category-quality.md)
**Patterns**: Design pattern compliance, output format compliance → [references/category-patterns.md](references/category-patterns.md)

## Best Practices

When performing SKILL.md reviews:

- **Constructive and specific**: Include concrete examples and reference to existing well-structured skills
- **Context-aware**: Understand skill purpose and target audience, consider tradeoffs
- **Clear priorities**: Distinguish between CRITICAL (structural) and ENHANCEMENT (quality improvements)
- **Prioritize automation**: Avoid excessive focus on YAML syntax or formatting (use yamllint/markdownlint for that)
- **Prevent scope creep**: Pay special attention to Q-02 Scope Boundaries items
- **Respect patterns**: Emphasize consistency with established Review/Validation skill patterns

For detailed checks in each category, refer to the corresponding file in the [references/](references/) directory.
