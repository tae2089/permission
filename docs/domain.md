# Domain Rules

## Purpose

This service currently defines an initial Project bootstrap slice alongside the
template User example. Project is the top-level isolation boundary for the
future authorization service; this slice does not implement Tenant, Principal,
Role, Policy, or permission evaluation. It does not own end-user login,
password, or OIDC authentication.

## Canonical Terms

### User

A person-shaped example resource identified by a UUID and described by a name
and normalized email.

- Avoid: account, member, principal, identity
- Related: normalized email
- Notes: creation assigns the UUID; the example defines no update, deletion,
  authentication, authorization, or lifecycle state.

### Normalized Email

The User email after trimming surrounding whitespace and converting letters to
lowercase.

- Avoid: username, login
- Related: User
- Notes: uniqueness applies to the normalized value. The example requires a
  non-empty value but intentionally does not define RFC email syntax validation.

### Project

An instance-scoped authorization boundary identified by a server-generated
UUID, immutable name, and creation timestamp.

- Avoid: tenant, account, organization
- Related: Audit Event
- Notes: Project names are not unique. Project API keys are always scoped to
  exactly one Project.

### Project API Key

An opaque machine credential scoped to one Project. It is either a
`project_admin` key, which manages keys for its own Project, or a `decision`
key, reserved for future permission-decision requests.

- The raw secret is returned only when issued or rotated; only its verification
  hash is persisted.
- A Project and key kind may have at most two active keys, so clients can
  deploy a replacement before explicitly revoking the former key.
- Revocation is immediate and leaves immutable audit evidence. There is no
  automatic expiry in this slice.

### Audit Event

An immutable record of a management change. Creating a Project records one
`project.created` event with actor `instance_admin`, Project scope, and the new
Project as its target.

- Avoid: request log, decision log
- Related: Project
- Notes: audit events never store the administrator credential or authorization
  header.

## States and Transitions

The defined creation transitions are:

| Current state | Action | Preconditions | Next state | Observable result |
| --- | --- | --- | --- | --- |
| absent | Create User | valid name and non-empty normalized email not already used | created | UUID assigned and User persisted |
| absent | Create User | normalized email already used | absent | `already_exists`; no new User |
| absent | Create Project | valid Project name and authenticated instance administrator | created | UUID, timestamp, and `project.created` audit event persisted atomically |
| absent | Create Project | invalid name, missing credentials, or audit persistence failure | absent | `bad_request`, `unauthenticated`, or internal error; no Project row |

Update, deletion, suspension, and restoration are unresolved and must not be
implemented without new domain rules.

## Invariants

### User name

- Trim surrounding whitespace before validation and persistence.
- The trimmed name must contain 1–100 Unicode code points.
- Valid: `" Alice "` becomes `"Alice"`.
- Invalid: whitespace only or more than 100 characters returns `bad_request`.

### User email

- Trim surrounding whitespace and lowercase before persistence.
- The normalized value must not be empty.
- The normalized value must be unique across Users.
- Valid: `" ALICE@Example.COM "` becomes `"alice@example.com"`.
- Invalid: whitespace only returns `bad_request`.
- Reusing `"alice@example.com"` with any letter case returns `already_exists`.

### User identity

- Creation assigns a non-zero UUID before persistence.
- Clients cannot provide or replace the UUID during creation.

### Project name

- The client supplies an immutable name at creation.
- It must match `^[a-z0-9-]{1,63}$` exactly.
- Valid characters are lowercase ASCII letters, digits, and hyphens. Uppercase
  letters, whitespace, Korean characters, and every other character are
  rejected.
- The initial slice deliberately has no Project-name uniqueness invariant.

### Project creation and audit

- Creation assigns a non-zero UUID and a UTC timestamp before persistence.
- Project creation and its `project.created` audit event are one atomic
  same-database operation.
- If the audit record cannot be saved, the Project must not be created.

## Allowed and Forbidden Behavior

- `POST /users` may create exactly one User for a valid request.
- Repeated creation with the same normalized email must not create another User.
- A rejected request must not write a partial User.
- `POST /v1/projects` may create exactly one Project only for an authenticated
  instance administrator and valid name.
- `GET /v1/projects` may list Projects in creation-time order, with UUID as the
  deterministic tie-breaker.
- The bootstrap credential authenticates the instance administrator only; it is
  not a Project API key and is never persisted in a Project or audit record.
- HTTP DTOs, GORM records, database errors, and Gin contexts are not domain
  concepts and must not define new domain behavior.

## Verification

Handler tests cover the HTTP boundary and administrator authentication; Service
tests cover Project naming and audit-event construction; Repository tests cover
real SQLite persistence, ordering, and transaction rollback when audit storage
fails. Any new state, transition, or invariant requires allowed and rejected
tests before implementation.
