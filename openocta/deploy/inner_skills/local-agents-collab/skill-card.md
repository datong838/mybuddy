## Description: <br>
Guides OpenOcta to delegate work to local CLI agents (OpenClaw, Hermes, Cursor, Codex, OpenCode, Trae) via the local_agent tool after LLM analysis. Prevents direct CLI invocation from shell. <br>

## Publisher: <br>
OpenOcta (built-in) <br>

## Use Case: <br>
When users @mention local coding agents or ask to run Cursor/Codex/OpenCode on the machine, the main agent formulates complete tasks and calls local_agent instead of bypassing the LLM. <br>

## Known Risks and Mitigations: <br>
Risk: Delegated CLIs may modify files or run commands in the workspace. <br>
Mitigation: Task text should state scope and constraints; rely on OpenOcta approval/sandbox policies where configured. <br>
