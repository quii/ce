# Conversation Engine (CE) - A containerised application for storing and managing threaded conversations about entities

*Threaded conversations as commodity infrastructure.*

CE is an application that provides a RESTful HTTP API for managing conversations. It should be viewed as simple commodity software that a team can deploy, connect to a Postgres database and have a way of adding conversations to their system

## What is a conversation

A conversation is a collection of data of various participants discussing a "resource". As we are leaning on REST and hypermedia, this will be represented as a URL. CE is not concerned with the specifics though, and will not use this data - it should be seen as merely a label for the participants

A conversation should be seen as a tree of threads. Each conversation will have at least one starting thread. A thread can have a title, and a thread can have a set of participants, which can change over time.

A participant, is just an identifier, like a UUID, provided by the calling system. CE does not validate these identifiers, but it can be used in a query - like "get all conversations by participant"

A thread has a list of messages. Messages have text in them, and can have newlines, but no other formatting. Messages can also have attachments. A new child thread can be spawned at any point with a list of messages, as users discover tangents to go down. 

Attachments, again leaning on hypermedia, are URLs provided by the calling system. CE does not read, validate or offer any AUTHZ/N on them. It is up to the calling application to decide how it handles attachments.

On a technical level, conversations can only be accessed by the application that created it

# Non-goals

- Validating or interpreting the "resource" a conversation is about - it's just a URL as far as CE is concerned
- Validating participant identifiers - they're opaque, supplied by the calling system
- Reading, storing, or authorizing access to attachments - they're URLs, handled entirely by the calling application
- Authenticating requests - that's the job of whatever sits in front of CE (see authentication)
- Fine-grained authorization - see authorization
- True deletion/erasure - a "deleted" message is a tombstone, not gone; the audit log is immutable
- Rich text or formatting in messages - plain text with newlines only
- Being the primary UI - the htmx frontend exists to help with demoing over a browser, not as a production interface

# Cross-functional requirements

## Auditability

Auditability is a must. Every single part of a conversation around an entity should be retrievable. For this reason, the system should use event sourcing. 

## Authentication

CE does not authenticate requests itself. It expects to sit behind an API gateway (or reverse proxy) that authenticates the calling application and forwards a verified identity via a header. The header name is configurable rather than hardcoded. If the header is missing, CE fails closed (401) - there is no unauthenticated fallback path.

## Authorization

Beyond the rule that an application can only access conversations it created (see above), authorization is not CE's concern - it's up to the calling application to decide what its own users/participants can see and do. CE just gives callers the means to scope their own queries.

For example, a caller can pass multiple `participant` querystring params - `?participant=abc&participant=def` - to say "only give me data involving these participants". Whether and how that scoping is applied is entirely up to the calling application; CE just supports the query shape.

## Developer experience

The whole stack should come up locally with a single `docker compose up` - Postgres, the API, the relay, and a thin `web` app that renders HTML/HTMX over the API's generated client. Each is its own binary under `cmd/`, built from the same Dockerfile via a `SERVICE` build arg, so there's still only one image definition and no other local dependencies required.

# High level technical choices

## "True" REST and hypermedia

We lean on using hypermedia, hyperlinks and so on to reduce coupling and increase discoverability. 

To help with demo-ing, we will use content negotiation. CE will have a simple HTML/htmx/CSS "stack" to let us drive the application through a browser, but if a machine wants to do it, they can use the same endpoints and so on, but leverage hypermedia to use JSON instead. 

## Technical approach

Writes are event sourced rather than row-mutated - every state change is captured as an event, which is what full audit retrieval requires, and gives us a clean answer for messages being editable/deletable (an edit or delete is a new event, not an UPDATE/DELETE). Those events reach read-optimised projections via a transactional outbox, so the event store and projections stay consistent without a dual-write problem. See `docs/` for more on the architecture and technical detail behind this.

## Tech stack

- Docker 
- Go
- htmx
- CSS 

For Go and CSS, no frameworks whatsoever. Standard library, standard, well supported CSS. 
