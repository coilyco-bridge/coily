# 16. Subagent ID propagation

Category: Open questions

Audit log reads `CLAUDE_SESSION_ID` from env. Claude Code does not
actually set this env var today (as far as I know). Consequence:
`session_id` in audit records is always empty. Fix options:

- Hook into Claude Code's hooks system to inject CLAUDE_SESSION_ID.
- Give up on session correlation and use invocation timestamps instead.
- Use a different env var that Claude Code does set (TERM_PROGRAM?
  process ancestry check?).

# Decision

I don't care to attach IDS

Also, make the times unix timestamps.
