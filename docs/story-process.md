# Story process

Stories are small, focused, and iterative. A story is not a spec handed over finished - it's arrived at through conversation, structured as [example mapping](https://cucumber.io/docs/bdd/example-mapping/).

## Example mapping the conversation

Four kinds of things come out of the conversation:

- **the story** - what we're building
- **rules** - the business/domain rules that must hold for the story to be true
- **examples** - concrete illustrations of a rule
- **questions** - things we don't know yet and shouldn't guess at

Every example should map to at least one rule, and every rule should have at least one example. A rule with no example hasn't been pinned down yet; an example that doesn't reveal a rule is probably noise. As new examples come up, they either confirm an existing rule or expose a problem with it - a contradiction, a missing case, a rule that was actually two rules wearing a trenchcoat. When that happens we fix the rule (or split it), not just add the example on top and move on.

We keep going until there are no open questions left and new examples stop surfacing new rules - a stable map is what "done with this conversation" looks like. If the map keeps growing (many rules, many examples, a pile of open questions), that's the signal to split the story, not to push through in one sitting.

Gherkin is documentation, not test source. We do not generate tests from feature files - the spec exists so a human or an agent can read what the system is supposed to do without reading code.

## From map to story file

Once the map is stable, the story file records two things, kept separate rather than blended together:

- the consolidated examples, written up as Gherkin scenarios
- the rules the map converged on, stated explicitly as rules in their own right

The rules are the more durable output of the two - examples are illustrations, but the rules are what the exercise was actually for.

## Rules become code, not just prose

A story's rules should show up literally in the domain code - a guard clause, a validation function, a named constant, whatever's idiomatic for the rule - not be something an agent has to re-infer from reading Gherkin scenarios each time. The story file's rules section is a checklist of what the domain layer must actually enforce; implementing a story means making each rule true in code, not just making the examples pass.

## From story to specification

An in-port is meant to model a user doing a job (see `docs/architecture.md`), so most stories should point to a specification that drives a single use-case in-port - see `docs/adr/0022-specifications-and-drivers.md` for what a specification is and why it's written against a `Driver` rather than a concrete environment. That specification, run through the in-process driver at minimum, is the automated proof the job works.

Some stories will exercise more than one in-port - typically one to write, one to read back and confirm the effect. Where that's the case, the story references all the specifications involved, not just one.

## Reading ADRs before writing code

Implementation starts after the story, not straight from it. Once a story's map is stable, read the ADRs in `docs/adr/` whose `scope` overlaps the code about to be touched - the same path-based scoping the pre-commit ADR check uses (see `docs/source-control.md`) - before writing anything. The point is to write it correctly the first time rather than relying on the pre-commit check to catch a decision that could have been known upfront.

## Keeping completed stories honest

Gherkin is documentation, not a generated or executable spec, so nothing mechanically stops a completed story's spec from drifting away from what its linked specification(s) actually verify as the code evolves. We don't try to force that link into a hard gate - whether a line of English still matches what a specification verifies is a judgment call, not something a linter can check.

One layer underneath that judgment call isn't a judgment call at all, though: whether the reference resolves. A test that fails against every completed story's references, checking the specification file exists, the named test function exists in it, and it currently passes (through at least the in-process driver) - is a plain existence check, not semantic drift-detection, and runs before the agent-driven check below ever gets involved. A story pointing at a specification that's been renamed, moved, or deleted is a broken reference regardless of what the English says, and that much is fully mechanical.

On top of that: whenever a specification referenced by a completed story changes, an agent reads that story's Gherkin and rules against the current specification and flags anything that's drifted - new behaviour the spec doesn't mention, a rule that no longer holds, a scenario the specification no longer covers. This is targeted at stories whose linked specifications actually changed, not a blanket sweep, and it's advisory rather than blocking - semantic checks will have false positives, and this should never be what holds up `go test -race -count=3 -shuffle=on ./...` passing.

## Folder layout

- `stories/backlog/` - stories not yet built
- `stories/completed/` - a Gherkin spec, the rules the map converged on, and a reference to the specification(s) that exercise it

A story only moves from `backlog/` to `completed/` once there's a real specification to point to. No specification, no move - that's what "done" means here.

One exception: `stories/iteration-0.md`, sitting outside both folders. Scaffolding work - making the deterministic gates described across `docs/` actually exist - has no user doing a job, so it has no example map, no Gherkin, and nothing to converge on. It's a checklist, not a story, and it's not expected to fit the format above.
