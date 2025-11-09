## Rules
### Interaction & Execution Rules
- Always answer first: Respond clearly to the user’s question. Do not modify or write files when answering the question.
- Separate proposals: When proposing edits or command executions that change the state of files, data, or configurations, present them as proposals including:
  - your understanding of the background
  - the steps, commands, target files, and expected changes
- Explicit approval required:
  - Require explicit user approval only for actions that create, update, delete, or otherwise modify files or data.
  - You may execute **read-only** or **inspection** commands at any time.
- Skip redundant confirmation:
  - If the user’s instruction already includes a clear imperative, do not re-confirm. Proceed directly with execution, unless the command would cause an irreversible or high-impact change.
- Ask only when ambiguous: 
  - If scope or intent is unclear, ask clarifying questions before taking any action.

### Language Rules
- **Always think in English.**
  - **Translate all user-facing outputs into Japanese.**
- Do not invent new words or unusual abbreviations.
- Avoid over-compressed or unnatural phrasing.
- Prioritize clarity and natural sentence flow over brevity.

### Documentation Rules
- **Unless otherwise specified, all documents and code comments must be written in English.**

### Development Policy
- All development activities must follow the rules and guidelines described in CONTRIBUTING.md.

### Shell Rules
 - Use `python3` instead of `python`.
