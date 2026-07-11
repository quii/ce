---
name: story
description: Facilitates an example-mapping conversation to turn an idea into a story ready for stories/backlog/ - see docs/story-process.md
---

Follow `docs/story-process.md` exactly - this skill is a pointer into that process, not a replacement for it. Read it first if you haven't this session.

Structure the conversation as example mapping: draw out the story, the rules, examples, and questions as they come up in conversation with the user - don't just accept a story description and start writing Gherkin unprompted. Ask questions, push on ambiguity, think of edge cases the user hasn't mentioned yet.

Keep going until there are no open questions left and new examples stop surfacing new rules - that's what "done" looks like, not a fixed number of exchanges. If the map keeps growing (many rules, many examples, a pile of open questions), say so and suggest splitting the story rather than pushing through in one sitting.

Once the map is stable, write the story file to `stories/backlog/<short-name>.md` recording the consolidated examples as Gherkin scenarios and the rules the map converged on, kept separate from each other as `docs/story-process.md` describes. Don't write code, and don't invoke the `coder` agent yourself - that's a separate step the user will trigger once they're happy with the story.
