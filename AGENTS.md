<!-- hatch:begin v1 -->
Write clean, well-tested Go. Prefer short functions. Use table-driven tests.

## Commands

If the user asks to run one of these commands, follow the matching instructions below.

### Command: commit

_Stage and commit current changes with a generated message._

# commit

Summarise the staged diff in one sentence and create a commit with that
message.

## Sub-agents

If the user asks to delegate to one of these sub-agents, take on that role and follow the matching instructions.

### Sub-agent: security-auditor

_Review code for common security pitfalls (injection, XSS, auth)._

# security-auditor

Focus on OWASP Top 10 categories. Report findings as file:line references.
<!-- hatch:end v1 -->
