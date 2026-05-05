# MRAG Research Python Agents

This directory is reserved for stage 1 research-asset helper scripts.

Current scope:

- paper parsing helpers
- insight extraction helpers
- idea generation helpers

Non-goals for stage 1:

- workflow orchestration
- experiment planning
- automatic execution
- autonomous agents

Scripts in this directory should follow a simple contract:

- accept CLI arguments
- read/write MRAG workspace v2 paths when needed
- print a single JSON object to stdout on success
- exit non-zero on failure
