# claude-review

claude-review is a lightweight companion for working on planning documents with Claude Code. It lets you review a
Markdown plan in the browser, leave inline comments, and hand those comments back to the same Claude Code session that
created the plan.

## How it fits into your workflow
- Start Claude Code in a project, ask it to produce a plan (for example `PLAN.md`), then open the rendered Markdown in
  your browser.
- Highlight portions of the document and add contextual comments without leaving the page.
- Run the `/review` slash command in your Claude Code session to pull the pending comments directly into Claude Code,
  apply the requested edits, and resolve them.
- Watch the plan automatically reload in your browser, repeating the cycle until the document matches your intent.
