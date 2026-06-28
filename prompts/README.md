# Prompts

Prompt templates embedded into the binary at compile time.

| File | Used By | Purpose |
|------|---------|---------|
| `system.txt` | All modes | System instructions and strict rules prepended to every LLM call |
| `selection_instruction.txt` | Two-pass mode (pass 1) | Instruction telling the model to choose a response type |
| `selection_context_prefix.txt` | Two-pass mode (pass 2) | Prefix appended to the user prompt to communicate the selected response |
