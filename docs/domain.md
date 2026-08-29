# Domain Rules

## Purpose

This template includes one example domain slice to demonstrate how product
language, invariants, HTTP handling, and persistence stay aligned. The example
defines User creation only; it is not a general identity or authentication
system.

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

## States and Transitions

User currently has no lifecycle state. The only defined transition is creation:

| Current state | Action | Preconditions | Next state | Observable result |
| --- | --- | --- | --- | --- |
| absent | Create User | valid name and non-empty normalized email not already used | created | UUID assigned and User persisted |
| absent | Create User | normalized email already used | absent | `already_exists`; no new User |

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

## Allowed and Forbidden Behavior

- `POST /users` may create exactly one User for a valid request.
- Repeated creation with the same normalized email must not create another User.
- A rejected request must not write a partial User.
- HTTP DTOs, GORM records, database errors, and Gin contexts are not domain
  concepts and must not define new domain behavior.
- Authentication and authorization are intentionally absent from this example;
  adding either requires a separate contract.

## Verification

Handler tests cover the HTTP boundary, Service tests cover every invariant, and
Repository tests cover persistence and unique-email enforcement with real
SQLite. Any new state, transition, or invariant requires allowed and rejected
tests before implementation.
