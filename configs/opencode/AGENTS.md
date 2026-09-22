# Global Software Engineering Rules

These rules are mandatory in every project. A more specific project `AGENTS.md`
may define project details and may override only the areas explicitly delegated
below, such as the testing strategy. Never silently ignore or weaken a global
rule because a task appears small, urgent, or obvious.

## Shell And Tool Selection

- Prefer Bash when one well-designed command or pipeline can inspect, filter,
  aggregate, or summarize information more quickly and readably than multiple
  independent tool calls. Use the configured Bash shell and available Unix
  utilities deliberately; do not use shell merely to avoid a suitable native
  OpenCode tool.
- Choose tools by capability. Use native `read`, `grep`, and `glob` for precise
  file access and simple searches; use LSP for semantic code intelligence; use
  native edit tools for every file change; and use dedicated web, question,
  skill, task, and MCP tools for their respective domains. Use Bash for
  pipelines, cross-file summaries, repository inspection, and project-provided
  test, lint, format, build, and verification commands.
- Before relying on Bash or a Unix utility option, determine the installed
  implementation and exact version and consult its official documentation.
  Prefer portable POSIX behavior when practical. Never assume GNU-specific
  flags: environments may provide GNU, BSD, BusyBox, or uutils implementations.
- Build pipelines so failures remain visible. Quote expansions and paths,
  separate options from operands with `--` when supported, avoid parsing
  human-oriented output when a stable machine-readable mode exists, and use
  `set -o pipefail` when the success of every pipeline stage matters.
- Keep shell output focused. Filter at the source, cap potentially large output,
  preserve diagnostics needed to understand failures, and avoid decorative
  separators or repeated command output. Use `jq` for JSON only when its exact
  installed version and required behavior have been verified.
- Never use Bash to create, overwrite, patch, rename, move, copy, or delete
  files. This prohibits redirections to files and mutating utilities or modes
  such as `tee`, `sed -i`, `perl -pi`, `cp`, `mv`, `rm`, `truncate`, and `touch`.
  Use native edit tools so every content change remains explicit and reviewable.
- Project tools that legitimately modify files, including formatters, code
  generators, package managers, and snapshot updaters, may run only when their
  state changes are part of the approved plan and the Bash permission is
  explicitly approved. Inspect their resulting diff immediately afterward.
- Never use Bash to bypass OpenCode permissions, access denied paths, inspect
  real `.env` files or secrets, evade an approval gate, or conceal a destructive
  operation inside a script, subshell, command substitution, pipeline, alias,
  or indirect command.
- Treat Bash permission as a security boundary. Commands not covered by a
  narrow read-only allow rule must require user approval, and destructive or
  direct file-writing command patterns must remain denied.

## Mandatory Workflow Gate

- Exploration, documentation research, diagnosis, and planning are allowed
  before approval only when they do not modify project or external state.
- Before creating, editing, deleting, generating, installing, formatting, or
  committing anything, load and follow the `brainstorming` skill from
  `obra/superpowers`.
- Present a concrete design and implementation plan covering scope, affected
  files, abstractions, dependencies, tests, verification, risks, and explicit
  non-goals.
- Do not perform any implementation action until the user explicitly approves
  that exact plan. Silence, a request for options, or approval of a different
  plan is not approval.
- Approval applies only to the accepted scope. Stop and request renewed
  approval when requirements, files, architecture, dependencies, or testing
  strategy materially change.
- After design approval, use the `writing-plans` skill before implementation
  when required by the Superpowers workflow.
- A request to implement an already approved plan is sufficient approval only
  when the plan and its scope are present in the current conversation.
- The only exception is the project-local MongoDB onboarding initiated through
  `/mongodb-setup` and executed by `mongodb-context`. For that workflow, the
  completed interview, exact proposed configuration diff, and explicit user
  approval together constitute the required design and implementation plan.
  This exception permits editing only one approved project OpenCode
  configuration file and configuring the approved pinned MongoDB MCP package.
  It never permits source-code changes, executing dependency installation
  during onboarding, database writes, secret handling, commits, or unrelated
  edits.

## Version-Specific Documentation

- Before relying on the behavior, API, configuration, or command syntax of any
  language, runtime, framework, library, tool, service, or utility, determine
  the exact version used by the project from manifests, lockfiles, toolchain
  files, generated metadata, or the runtime itself.
- Fetch and consult the official documentation for that exact version before
  designing or implementing dependent work. Do not substitute latest-version
  documentation for a legacy version such as React 16.
- Prefer primary sources in this order: versioned official documentation,
  official release or migration notes, and the source repository at the exact
  release tag. Treat blogs, snippets, memory, and unversioned search results as
  non-authoritative.
- Record the version and authoritative URLs used in the plan or final report.
- If the exact version cannot be determined, ask the user before assuming one.
- If authoritative documentation cannot be fetched or accessed, stop the
  dependent design or implementation and ask the user for accessible sources
  or permission to use a clearly identified alternative. Never invent API
  details from memory.

## Architecture And Maintainability

- Inspect the relevant code, project rules, architecture, naming, error
  handling, and tests before proposing a design.
- Select the simplest maintainable solution that satisfies the approved scope
  and fits the project's established, actively used patterns.
- Do not introduce a new architectural layer, abstraction level, paradigm, or
  competing pattern without explicit user approval.
- When the maintainable solution requires an abstraction different from the
  existing ones, stop and present a refactoring plan that explains the
  inconsistency, migration scope, compatibility impact, tests, and incremental
  rollout. Implement neither the feature divergence nor the refactor until the
  user approves the revised plan.
- Do not perform unrelated refactors. Do not preserve a known unsafe or broken
  pattern merely for consistency; report the conflict and request a decision.

## Conditional System Design Documentation

- Apply this section only when OpenCode is creating a project from scratch, the
  user explicitly activates it for the current session, or a scoped project
  `AGENTS.md` declares it active. Do not impose it retroactively on an existing
  project without one of these activation conditions.
- When active, make system-design specification the first implementation
  deliverable after the design and implementation plan are approved and before
  implementing the rest of the application.
- Define the project's data types in its native implementation language. These
  types are the canonical source of truth and must represent the actual code,
  constraints, optionality, and relationships.
- Create `docs/architecture/data-model.md` with a Mermaid `classDiagram` or
  `erDiagram` that represents the canonical types and their relationships.
  Include references to the source files that define those types. Keep this
  document synchronized whenever relevant code changes.
- Add one Mermaid diagram to `README.md` that shows the system only at a high
  level of abstraction: major components, external actors or systems, and
  principal connections. Exclude classes, fields, methods, implementation
  details, and low-level request flows.
- Use Mermaid embedded in Markdown as the primary diagramming format. Use
  PlantUML only when Mermaid cannot correctly express the required diagram,
  record a brief justification, and never maintain duplicate Mermaid and
  PlantUML versions of the same diagram.
- Keep `README.md` limited to the project name, prerequisites, installation,
  local configuration without secrets, local execution, tests, auxiliary
  services, and the required high-level system diagram. Put architectural
  explanations, design rationale, domain details, and all other extended
  documentation under `docs/`, not in the README.
- Treat diagrams as maintained artifacts, not aspirational designs. Every
  change to canonical types, type relationships, system boundaries, major
  components, or component dependencies must update the affected diagram in
  the same approved change.
- Keep diagrams readable and at their declared abstraction level. Validate
  syntax with project-provided tooling when available. Do not install Mermaid,
  PlantUML, a renderer, or a documentation dependency solely for validation
  without a separately approved dependency decision.

## Code Language And Documentation

- Write all source code and developer-facing text in English without
  exception. This includes identifiers, comments, docstrings, documentation,
  log messages, error messages, test names, fixtures, and configuration text
  maintained by OpenCode.
- Use the language's native docstring or documentation-comment convention. If
  the language has no docstring syntax, use its standard API documentation
  comment format.
- Every public function and public method must have a docstring containing a
  concise human-readable explanation in English. Document parameters, return
  values, raised errors, side effects, and relevant invariants whenever they
  apply.
- Every private function and private method must also have a docstring. When
  its behavior is completely evident from precise naming, parameters, and
  return type, the docstring may contain only the structural signature
  information required by the language or documentation tooling; do not add a
  redundant human-language explanation.
- The private-member exception never applies to public APIs. Public functions
  and methods always require a human-readable explanation, even when their
  names appear self-explanatory.
- Treat missing or non-English required documentation as an incomplete code
  change. Add or update documentation for every function or method created or
  modified within the approved scope.
- Do not expand the task into documenting unrelated legacy code. Report
  pre-existing violations outside the approved scope instead of modifying
  them without approval.

## Runtime Configuration And Environment Variables

- Load deploy-specific configuration exactly once during application startup.
  Validate and convert every value before starting application services, then
  expose the resulting configuration through constants or immutable structures
  in the project's native language.
- Fail fast when a required value is missing or any value has an invalid type,
  format, range, or allowed value. Configuration errors must be actionable and
  written in English, but must never include secret values.
- After startup configuration is built, application modules must consume the
  immutable configuration rather than reading `process.env`, `os.environ`, or
  equivalent runtime environment APIs directly. Configuration changes require
  an application restart; runtime mutation or environment re-reading is not
  allowed.
- The following deploy-specific values must always originate from environment
  variables: URLs and external endpoints that vary by deployment; passwords,
  tokens, keys, and all other secrets; network ports; and feature flags.
  Stable internal routes, protocol constants, and values that cannot vary by
  deployment do not become environment variables merely because they resemble
  a URL or configurable string.
- For local development, load environment variables from an untracked `.env`
  file. Keep `.env` and environment-specific secret variants ignored by version
  control. Never create, inspect, echo, log, commit, or expose real secret
  values while implementing or verifying a change.
- Maintain a version-controlled `.env.example` that lists every supported
  required and optional environment variable, contains only safe placeholders
  or non-sensitive example values, identifies optionality without secrets, and
  is updated in the same change as the startup configuration contract.
- Production and other deployed environments may inject the same variables
  through the operating environment, containers, CI/CD, orchestration, or a
  secret manager; they do not require a physical `.env` file.
- Other non-sensitive configuration may live in code or a project-native
  configuration file when appropriate. For Meteor projects, use `settings.json`
  only for non-sensitive settings that do not belong to the mandatory
  environment-variable categories; secrets never move to `settings.json`.
- In fullstack applications, expose configuration to client code only through
  the documented mechanism and visibility rules of the exact framework version
  in use. Regardless of framework conventions, passwords, tokens, keys, and
  other secrets must never enter client bundles, client-readable settings,
  responses, telemetry, logs, or error messages.
- Prefer the runtime or framework's existing configuration facilities. Do not
  add `dotenv`, a validation library, or any other dependency automatically;
  apply the global dependency-selection process and obtain approval first.
- Test startup configuration behavior according to the project's testing
  strategy, including successful conversion and failures for missing or invalid
  required values. Tests must use synthetic values and must never read a real
  developer `.env` file or contact real external services.

## Application Logging

- Use the standard logging facilities of the exact language or framework
  version whenever they can satisfy this contract. Do not add a logging
  dependency for formatting, levels, colors, or JSON output that can be
  implemented with existing facilities and a small internal adapter. Any
  exception requires the global dependency-selection process and explicit
  approval.
- Configure logging once during application startup from the immutable runtime
  configuration. Use the framework's documented environment-mode convention to
  select `development` or `production`; reject unsupported values before
  starting services.
- Require the `LOG_LEVEL` environment variable. Accept exactly `DEBUG`, `INFO`,
  `WARN`, or `ERROR`, and fail fast for missing or invalid values. Apply standard
  cumulative thresholds: `DEBUG` emits every level; `INFO` emits `INFO`, `WARN`,
  and `ERROR`; `WARN` emits `WARN` and `ERROR`; `ERROR` emits only `ERROR`.
- Write application logs exclusively to standard output, with exactly one log
  record per line. Do not make the application manage log files, rotation, or
  retention unless a separately approved project requirement demands it.
- In development, emit human-readable text containing local process time in
  `DD/MM/YYYY HH:mm:ss` 24-hour format, the log level, the originating module
  when the language or framework provides it without expensive inspection, and
  the message.
- In development, color only the level label when standard output is an
  interactive terminal with color support: blue for `DEBUG`, yellow for `WARN`,
  and red for `ERROR`; `INFO` requires no color. Never emit ANSI color sequences
  to non-interactive output, redirected output, or production logs. Omit colors
  rather than adding a dependency or non-portable complexity.
- In production, emit valid JSON Lines with no prefixes, suffixes, colors, or
  additional text. Every record must contain exactly the string fields
  `timestamp`, `level`, and `message`. Use an ISO 8601 UTC timestamp; `level`
  must be one of the four supported values. Do not add a module field.
- Put sanitized exception descriptions and stack traces inside the production
  `message` field when they are useful. Preserve each record as one JSON line by
  using the runtime's JSON serializer rather than manual string concatenation.
- Sanitize before calling the logger. Never log passwords, tokens, keys,
  connection strings, session identifiers, full personal data, or unredacted
  request and response bodies. Use minimal identifiers and explicitly allowlisted
  fields; logging failures must not expose the original sensitive values.
- Keep application logs distinct from metrics. JSON logs may be consumed by a
  log collector or observability pipeline; Prometheus integration requires
  separately designed numeric metrics and is not achieved by JSON logging.
- Test threshold filtering, required `LOG_LEVEL` validation, development
  formatting and local time, production JSON parsing and exact fields, UTC
  timestamps, conditional TTY colors, absence of ANSI outside an interactive
  development terminal, sanitization, exception formatting, one-record-per-line
  behavior, and exclusive standard-output emission.

## Dependency Selection

- Treat every new dependency as an architectural and security decision. Never
  install one only for convenience.
- First state the concrete project requirement that appears to require the
  dependency, then test whether that requirement is false, already covered by
  the language, runtime, platform, or an existing dependency.
- Inspect the exact candidate release, official documentation, license,
  maintenance activity, release cadence, supported runtimes, known security
  advisories, transitive dependencies, bundle or deployment cost, and likely
  upgrade burden.
- If the project needs fewer than five methods and an independent internal
  implementation is under 100 source lines, reject the dependency and propose
  the internal implementation. Count production source lines, excluding tests,
  generated code, comments, and blank lines.
- Implement internal behavior from project requirements and public contracts;
  do not copy third-party source unless its license and attribution obligations
  are explicitly reviewed and accepted.
- Verify the internal implementation against every runtime version supported
  by the project, including planned migrations from legacy releases to current
  LTS releases.
- A dependency may be proposed when the required behavior cannot reasonably be
  implemented within that threshold or when security-critical protocol or
  cryptographic behavior should not be reimplemented. Include this conclusion
  and evidence in the plan and obtain approval before installation.
- Prefer one focused, mature dependency over several overlapping dependencies.
  Pin or lock the selected version according to project conventions.

## Testing

- Testing is a release requirement unless a scoped project `AGENTS.md`
  explicitly defines otherwise.
- Read and follow the project's stated testing strategy, whether it requires a
  critical path, a coverage target such as 90 percent, unit tests only, or a
  different scope.
- If no project testing strategy exists, propose one in the plan based on risk
  and obtain user approval; do not silently choose or omit a strategy.
- Add or update tests for every behavior change according to that strategy.
  Reproduce defects with a failing test when practical.
- Run the narrowest relevant tests during development and the project-required
  verification suite before completion. Never claim success without current
  command output.

## Linting, Formatting, And Commits

- Discover project-provided scripts and targets before selecting verification
  commands.
- When lint or format scripts exist, including `npm run lint`,
  `npm run format`, `make lint`, or `make format`, run the applicable commands
  and require them to pass before creating a commit.
- Run the approved test suite before a commit as well. Do not bypass hooks,
  suppress failures, or alter lint, formatter, or test configuration merely to
  obtain a passing result unless that change is part of the approved plan.
- If any required check fails, do not commit. Report the command and failure,
  fix only issues within the approved scope, then rerun all affected checks.
- Do not create a commit unless the user requested it or the approved workflow
  explicitly includes it.

## Completion Evidence

- Before reporting completion, compare the final diff with the approved plan
  and list any deviation for user approval.
- Report documentation sources, tests, lint and formatting commands, and their
  outcomes. Clearly identify checks that could not be run and why.
