# 1. Migrate the provider to terraform-plugin-framework

- Status: Proposed
- Date: 2026-07-10
- Deciders: maintainers

## Context

The provider is built on `terraform-plugin-sdk/v2` (`helper/schema`). SDKv2 is in
maintenance mode; HashiCorp's recommended path for actively developed providers is
`terraform-plugin-framework`.

The recent MONGODB-AWS / IAM work on `mongodb_db_user` made the cost of staying on
SDKv2 concrete. Several fixes existed only to work around SDKv2 semantics rather than
to express real behavior:

- `password` was forced to `Optional + Computed` purely so `ResourceDiff.Clear`
  ("Clear only operates on computed keys") was legal.
- Cross-field rules (password ↔ `auth_mechanism`, ARN-shaped `name` when IAM) had to
  live in `CustomizeDiff` because `ValidateDiagFunc` cannot see sibling attributes.
- `auth_database` is forced to `$external` for IAM users via `SetNew` inside
  `CustomizeDiff`.
- IAM users are detected on read by inspecting the `$external` database because SDKv2
  gives no first-class way to model a conditionally-computed value.

Each of these is a first-class, well-supported concept in the framework
(typed null/unknown values, attribute/plan modifiers, `ConfigValidators`), so the same
behavior is expressed more directly and with less room for the classes of bug we hit.

The team has two related migrations in flight — the `terraform-provider-airflow`
SDKv2→framework mux migration (in progress; surfaced an env-conditional
provider-schema gotcha) and `terraform-provider-confluent-schema-registry` (framework
migration pending). Neither has fully landed yet, so this is a pattern the team is
converging on rather than a proven path. The upside: the mux approach and the emerging
playbook are reusable, and completing a smaller provider here can de-risk the larger
two.

A precondition is now satisfied: the provider has a real CI safety net — split
build/vet/tidy, unit + property, and acceptance (MongoDB 7.0.29) jobs — so a
behavior-preserving refactor of this size can be validated at each step.

## Decision

Migrate the provider to `terraform-plugin-framework`, **incrementally**, using
`terraform-plugin-mux` (`tf5to6server` + `tf6muxserver`) so framework and SDKv2
resources coexist and the provider stays shippable throughout.

Order of migration:

1. **Mux scaffold** (S) — stand up the muxed provider server (`tf5to6server` +
   `tf6muxserver`) with the existing SDKv2 resources unchanged. No behavior change.
2. **`mongodb_db_user`** (L) — migrate first: most complex, benefits most (IAM
   plan/validation → plan modifiers + `ConfigValidators`), and exercises the hardest
   case early. Trade-off: this compounds first-time mux-scaffold risk with the hardest
   resource. If the scaffold proves shaky, migrate a trivial resource (e.g.
   `db_role`) first to isolate mux-wiring problems from resource-logic problems.
3. **`mongodb_db_role`, `mongodb_db_collection`, `mongodb_db_index`** (M total) —
   largely mechanical once the pattern is set.
4. **Provider configuration + teardown** (S) — migrate the provider schema, then
   remove mux once nothing SDKv2 remains.

Sizes are rough (S/M/L) and dominated by porting each resource's acceptance tests to
`terraform-plugin-testing`, not by the resource logic. The existing acceptance suite
is the regression oracle.

**Exit criterion per phase — proceed only if all hold:** the migrated resource's
acceptance suite is green on the existing CI; a state-upgrade test from the last
SDKv2 release shows no diff (see *State compatibility* below); and the provider still
builds and serves. If any fails, stop — mux keeps the provider shippable at the last
good phase.

## Consequences

Positive:

- Cross-field validation and conditional defaults become explicit and testable
  (`ConfigValidators`, attribute/plan modifiers) instead of `CustomizeDiff` workarounds.
- Correct null/unknown handling removes the `Optional + Computed` hacks and the
  `d.Clear` class of error.
- Aligns with the team's other providers; shared patterns and reviewers.
- Off a maintenance-mode SDK and onto the supported one.

Negative / costs:

- Real engineering effort: 4 resources + provider config, plus rewriting acceptance
  tests against `terraform-plugin-testing`.
- **No user-facing functional change** — the payoff is maintainability and
  future-proofing, not features. Sequence deliberately, not urgently.
- Temporary complexity while the provider is muxed (two schema styles in one binary).
- **State compatibility is the primary risk.** Moving a resource from SDKv2 to the
  framework must preserve its state representation, or existing users get spurious
  post-upgrade diffs or plan errors. Mitigation: a per-resource state-upgrade
  acceptance test (last SDKv2 release → framework version) as the phase gate above.
  This is the key unknown to validate on `db_user` before committing to the rest.
- Watch for the env-conditional provider-schema gotcha hit during the airflow
  migration (a Required/Optional flip driven by an environment variable).

## Alternatives considered

- **Stay on SDKv2.** Lowest effort, but keeps accumulating workarounds, stays on a
  maintenance-mode SDK, and diverges from the team's other providers. Rejected.
- **Big-bang rewrite (no mux).** Cleaner end state with no mux layer, but the provider
  is unshippable mid-flight and the change is unreviewable in one pass. Rejected in
  favor of the incremental mux approach.
