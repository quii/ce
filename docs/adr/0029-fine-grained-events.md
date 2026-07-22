---
id: 0029
title: Fine-grained events, not one per use case
status: Accepted
scope:
  - internal/domain/**
enforcement: judgment
---

# 0029: Fine-grained events, not one per use case

## Decision

An event models a domain fact - something that happened - not the shape of whichever use case happened to trigger it. A single use case can, and often should, raise more than one event atomically in one write; there is no requirement that a command produce exactly one event. Two tests decide whether a candidate event is really several facts wearing a trenchcoat - either is sufficient on its own:

1. **A second caller already exists.** Could a different, already-known use case plausibly produce this same fact? If yes, that's real evidence the fact deserves to be its own event, reused by every caller that produces it. The test for "already-known" is deliberately narrow: a concrete, currently-planned use case, not a hypothetical one imagined to feel thorough - splitting with no second caller in sight is speculative complexity in exactly the sense `docs/adr/0013-implement-only-the-current-test.md` already rules out for production code generally, it just happens to show up as event design instead of a domain method or a field.
2. **A sub-part already has its own independent multiplicity in the domain model.** Try to state the candidate event as one past-tense sentence. If it only holds together as several clauses joined by "and," and one of those clauses' subjects can already occur many times independently within the aggregate (per the domain model as already understood, not a guess) - a conversation *has many* threads, a thread *has many* messages (`brief.md`) - that's the split signal, even with only one caller today. This test doesn't need a second use case to fire: the multiplicity is a property of the domain itself, knowable on day one, not of how many callers happen to exist yet. A genuinely atomic event doesn't decompose this way - "a message was posted to a thread" has one subject (the message) with attributes (author, text, timestamp), not several independently-multiplying subjects glued together.

Test 2 is what test 1 alone would have missed: on a from-scratch design with only "start a conversation" as a known use case, test 1 gives no signal at all (there is no second caller yet), so it would have produced exactly the single monolithic event this ADR is retiring. Test 2 would have caught it immediately - "a conversation was created, and a thread was started on it, and a message was posted to the thread" is visibly three clauses, and two of their subjects (threads, messages) are already documented as things a conversation/thread can have many of.

Neither test is a license to guess every possible future split upfront - a candidate event that passes both tests (one cohesive fact, no independently-multiplying sub-parts, e.g. "a message was posted") stays one event, full stop.

## Rationale

`ConversationStarted` (the "start a conversation" story's genesis event) bundled conversation creation, thread creation, and the opening message into one event, because that's what the one use case that existed at the time needed - and test 2 above would already have flagged it, had it been applied at the time. It looked reasonable in isolation, but the event mirrored an HTTP operation rather than three separate domain facts - so when "reply to a thread" needed "a message was posted to a thread" as its own event (`ReplyPosted`), the opening message's already-recorded shape couldn't be reused at all; a second, near-identical event type had to be built from scratch (test 1 catching, in hindsight, exactly what test 2 could have caught upfront). The same gap was about to repeat itself the moment tags needed "something happened to a thread that isn't conversation-creation," and would repeat again whenever a "start a new thread on an existing conversation" story shows up - `ConversationStarted` has no way to produce a thread on its own, only a whole conversation.

Both gaps were visible from the domain model alone, before either "reply to a thread" or "thread participants" existed as stories - test 2 didn't need hindsight to catch them.

## Consequences

A single logical write can raise multiple events, committed atomically in one transaction (multiple event rows, multiple outbox rows, sequential sequence numbers) - `out.EventStore.Append` accepts a batch of events, not just one, and `out.Projection.Apply` applies all of them together before advancing the checkpoint. See `docs/write-path.md` for the mechanics.

`ConversationStarted` is retired in favour of three events raised atomically by `StartConversation`: `ConversationCreated` (creator, resource URL), `ThreadStarted` (thread ID, title, author, recipients), and `MessagePosted` (conversation/thread/message ID, author, text, timestamp) - the same event `ReplyPosted` already was, renamed and now reused for both the opening message and every reply. None of this is visible to a caller: `StartConversation`'s request/response shape, its validation rules, and `GetConversation`'s representation are all unchanged - this is entirely an internal event-sourcing reshape, proven by lower-level tests (use-case-level tests inspecting the outbox directly, and adapter contract tests), not by any change to the existing black-box specifications. `stories/completed/start-a-conversation.md`'s rule 5 ("there is no separate event for the thread or message") no longer describes the event shape accurately and needs updating to match, the same way rule 10 was updated for "thread participants".

Nothing here is constrained by existing production data - this reshape is free to change the Postgres schema (event types, columns) outright rather than migrate a previous shape forward, since no prior event shape has been released.

## Enforcement

Judgment - when a story introduces a new event or extends the shape of an existing one, check whether it's actually modelling a reusable domain fact or just mirroring the one use case that happens to produce it today, using both tests from the Decision section - a concrete second caller, and independent multiplicity of a sub-part already established in the domain model.
