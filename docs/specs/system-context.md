```mermaid
   C4Context
       title System Context — agentic-pbt

       Person(developer, "Developer", "Implements features and runs local verification.")
       Person(agent, "Implementing agent", "Writes implementation and property tests.")

       System(pbt, "agentic-pbt", "Portable PBT policy, verification engine, and pbt CLI.")

       System_Ext(pi, "Pi + agentic-pbt extension", "Injects policy, exposes status/tools, and requests correction.")
       System_Ext(claude, "Claude Code + agentic-pbt plugin", "Injects policy and maps lifecycle hooks to the shared CLI.")
       System_Ext(repo, "Target Git repository", "Application code, PBT configuration, contracts, and test suites.")
       System_Ext(runner, "Language PBT runner", "Hypothesis/pytest initially; fast-check/Vitest later.")
       System_Ext(ci, "CI", "Performs fresh verification in a controlled checkout.")
       System_Ext(docs, "Library documentation sources", "Installed, cached, or official version-matched PBT documentation.")

       Rel(developer, pi, "Uses")
       Rel(developer, claude, "Uses")
       Rel(agent, pi, "Implements through")
       Rel(agent, claude, "Implements through")

       Rel(pi, pbt, "Invokes CLI; translates events", "JSON/subprocess")
       Rel(claude, pbt, "Invokes CLI; translates hooks", "JSON/subprocess")

       Rel(pbt, repo, "Reads configuration and inputs; writes contracts/evidence")
       Rel(pbt, runner, "Executes configured property tests")
       Rel(pbt, docs, "Resolves focused, version-matched guidance")
       Rel(ci, pbt, "Runs fresh verification")
       Rel(ci, repo, "Checks out and validates")
 ```
