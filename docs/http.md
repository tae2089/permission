# HTTP API Contract

This document is the canonical source for HTTP handler, error response, trace
context, recovery, and access logging behavior.

## Handler Contract

Feature handlers use Gin's native handler shape and are registered directly:

```go
func (h *Handler) Create(c *gin.Context) {
	var request createRequest
	if err := input.DecodeJSON(c.Writer, c.Request, &request); err != nil {
		c.Error(trace.Wrap(err, "decode create user request"))
		c.Abort()
		return
	}
	created, err := h.service.Create(c.Request.Context(), CreateInput{
		Name:  request.Name,
		Email: request.Email,
	})
	if err != nil {
		c.Error(trace.Wrap(err, "create user request"))
		c.Abort()
		return
	}
	c.Header("Location", "/users/"+created.ID.String())
	c.Status(http.StatusCreated)
}
```

Handlers own successful responses. `internal/http/errors` owns error
classification and safe response types; the global
`middleware.ErrorHandler` owns failed responses. A handler that fails attaches
the error with `c.Error`, aborts, and returns without logging or writing a
response. Gin's `*gin.Context` must not pass into services or repositories.

A handler must not write any part of a success response before attaching an
error. Once response headers or a body have started, the global middleware
records the error but cannot safely replace that response.

## Input Contract

Handlers select the decoder matching the endpoint's declared media type:

| Decoder | Required media type | Limit | Result |
| --- | --- | ---: | --- |
| `DecodeJSON` | `application/json` | 64 KiB | caller-provided DTO |
| `DecodeXML` | `application/xml` | 64 KiB | caller-provided DTO |
| `DecodeForm` | `application/x-www-form-urlencoded` | 64 KiB | body-only `url.Values` |
| `DecodeMultipart` | `multipart/form-data` | 32 MiB | values and files in `*multipart.Form` |

Media-type parameters are allowed. `DecodeMultipart` requires a boundary and
keeps at most 8 MiB of file content in memory before using temporary files.
`DecodeForm` preserves repeated values and never merges URL query parameters
into the decoded body.

JSON and XML bodies must contain exactly one non-empty document. Empty
URL-encoded and multipart forms are transport-valid; required fields are a
separate validation concern. Malformed, oversized, and wrong-content-type input
is `bad_request`.

All decoders enforce transport policy before a feature maps its private input
into a typed Service input:

- transport shape is checked in the Handler; domain invariants are checked in
  the Service;
- request DTOs remain private to their Handler and never enter a Service or
  Repository.

`DecodeJSON` rejects unknown object fields. Go's standard XML decoder ignores
elements that do not match the target DTO; an endpoint that requires an XML
schema or unknown-element rejection must define and test that additional
contract explicitly.

The caller owns every successfully returned multipart form and must call
`Form.RemoveAll` on every path after it finishes reading uploaded files. Do not
ignore a cleanup error.

Do not use Gin bind helpers that write a response as a side effect. Failed
decoding must follow the same global error path as every other Handler failure.

## Server Timeouts

The HTTP server applies bounded connection lifetimes independently of request
body size limits:

| Phase | Timeout |
| --- | ---: |
| Request headers | 5 seconds |
| Complete request read | 60 seconds |
| Response write | 60 seconds |
| Idle keep-alive connection | 120 seconds |

These defaults prevent slow clients from retaining connections indefinitely.
An endpoint that legitimately needs longer uploads, downloads, or streaming
must define a separate timeout contract before changing the global limits.

## User Creation

`POST /users` is the example write endpoint.

Request:

```json
{
  "name": "Alice",
  "email": "alice@example.com"
}
```

On success it returns `201 Created`, sets `Location: /users/<uuid>`, and has no
response body. Clients cannot provide the UUID. Name and email normalization and
unique-email behavior are defined in [Domain Rules](domain.md).

An invalid name or empty email returns `400 bad_request`. An email that
normalizes to an existing User returns `409 already_exists`. The Handler does
not translate those errors itself.

## Project Bootstrap

Project bootstrap endpoints require the instance administrator credential:

```text
Authorization: Bearer <instance administrator key>
```

The key is configured by `instance.admin_key`. A missing, malformed, or invalid
credential returns `401 unauthenticated` with the standard fixed public
message. The response never identifies which part of the credential failed.

`POST /v1/projects` accepts strict JSON:

```json
{
  "name": "billing-api"
}
```

`name` must match `^[a-z0-9-]{1,63}$`. On success it returns `201 Created`,
sets `Location: /v1/projects/<uuid>`, and returns:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "billing-api",
  "created_at": "2026-08-29T10:00:00Z"
}
```

An invalid request body or name returns `400 bad_request`. Creation persists the
Project and its `project.created` audit event atomically; an internal storage
failure returns the standard `500 internal` response and creates neither row.

`GET /v1/projects` returns `200 OK`:

```json
{
  "projects": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "billing-api",
      "created_at": "2026-08-29T10:00:00Z"
    }
  ]
}
```

Projects are ordered by `created_at` ascending, then `id` ascending. The empty
result is `{"projects":[]}`.

## Project API Keys

`POST /v1/projects/<project-id>/api-keys` issues an opaque key. An instance
administrator may issue only the initial or recovery `project_admin` key; a
Project administrator key may issue either `project_admin` or `decision` for
its own Project. The request is strict JSON containing `{"kind":"project_admin"}`
or `{"kind":"decision"}`. A successful `201 Created` response contains the
public key metadata and `secret`; that raw value is never returned again.

`GET /v1/projects/<project-id>/api-keys` returns metadata only. A Project
administrator cannot read another Project's keys. `POST .../<key-id>/rotate`
issues a second active key of the same kind, and `POST .../<key-id>/revoke`
returns `204 No Content` and rejects that key immediately. There may be at most
two active keys of one kind per Project; exceeding the limit returns `429
limit_exceeded`. Every successful management change records an audit event.

## Error Flow

Create an `internal/apperr` error where the application meaning becomes known.
Each returning function that adds useful context—including capability packages
such as token or sealing code—uses Trace v3 to wrap the cause before returning:

```go
// Repository
if errors.Is(err, gorm.ErrRecordNotFound) {
	return apperr.New(apperr.KindNotFound, "user not found")
}
return trace.Wrap(err, "query user")

// Service
if err := repository.Find(ctx, id); err != nil {
	return trace.Wrap(err, "get user")
}
```

Do not log and return the same error. The request boundary records the final
handled error on the active OpenTelemetry span and logs it once with the
application-owned structured error attribute.

After downstream handlers finish, `middleware.ErrorHandler` reads the last Gin
context error. `errors.Resolve` inspects the complete trace chain and produces
the HTTP status and safe response. Outer operation messages remain internal;
the innermost semantic error determines the public classification.

Define authentication and authorization meanings in `internal/apperr` where
the decision becomes known, then use `trace.Wrap` at outer boundaries that add
useful operation context. `internal/http/errors` alone maps those meanings to
statuses and owns this template's response envelope and `trace_id` correlation
field. Trace v3 provides no HTTP response or middleware API.

## Error Response

Errors use one JSON envelope:

```json
{
  "error": {
    "code": "not_found",
    "message": "user not found",
    "trace_id": "0af7651916cd43dd8448eb211c80319c"
  }
}
```

`code` is a stable machine-readable string. The numeric HTTP status remains in
the HTTP response status and is not duplicated in the body.

| Error meaning | HTTP status | Code | Public message |
| --- | ---: | --- | --- |
| bad parameter | 400 | `bad_request` | semantic error message |
| authentication required | 401 | `unauthenticated` | fixed |
| access denied | 403 | `access_denied` | fixed |
| not found | 404 | `not_found` | semantic error message |
| already exists | 409 | `already_exists` | semantic error message |
| conflict | 409 | `conflict` | semantic error message |
| limit exceeded | 429 | `limit_exceeded` | semantic error message |
| canceled | 499 | `canceled` | fixed |
| not implemented | 501 | `not_implemented` | `internal server error` |
| unavailable | 503 | `unavailable` | `internal server error` |
| timeout | 504 | `timeout` | `internal server error` |
| unknown | 500 | `internal` | `internal server error` |

Invalid, expired, or revoked authentication credentials are `unauthenticated`
and map to 401. An authenticated principal without the required permission is
`access_denied` and maps to 403. Keep these as separate `apperr.Kind` values.
For bearer tokens, this follows
[RFC 6750 section 3.1](https://www.rfc-editor.org/rfc/rfc6750.html#section-3.1).

Outer `trace.Wrap` messages, causes, source locations, database errors, headers,
and request bodies are internal and must never appear in an HTTP response.

## Trace Context

Every request runs inside an OpenTelemetry server span. `otelgin` continues a
valid incoming W3C `traceparent` and `tracestate`; otherwise the injected SDK
provider creates a new trace and span. The active trace ID appears in error
responses and completion logs, and the active span ID appears in completion
logs.

The server does not generate or return `X-Request-ID`. Trace and span IDs are
correlation values, not credentials or authorization inputs. Export is disabled
by default, but the SDK provider remains active so identifiers are always
available.

`internal/apperr`, Trace v3, and OpenTelemetry trace have separate roles:
`apperr` defines application meaning, Trace records origin and propagation, and
OpenTelemetry carries distributed trace and span context.

## Request Logging

Write one completion record after the middleware chain finishes:

- stable message: `request completed`;
- attributes: `trace_id`, `span_id`, `method`, matched `route`, `status`,
  `duration`, and `response_bytes`;
- 5xx uses `Error`, successful `/healthz` uses `Debug`, and other responses use
  `Info`;
- handled errors include an application-owned `error` group containing the
  ordinary message and `fmt.Sprintf("%+v", err)` diagnostics;
- use the matched Gin route, never the raw URL or query string.

Never log credentials, authorization or cookie headers, DSNs, arbitrary
headers, or unbounded request and response bodies.

## Recovery

Recovery converts a panic into a traced error plus an application-private stack
diagnostic, records that stack only in the internal structured log, attaches
the error to Gin, and aborts the remaining handlers. The global error
middleware returns the generic 500 envelope. A typed error used as a panic
value must not change the response status.

Middleware order is:

```text
otelgin -> request logger -> error handler -> recovery -> handler
```

This order lets Recovery return a converted panic to the surrounding error
handler, then lets request logging observe the final status and error. Broken
connections and responses that have already started are recorded but not
rewritten.
