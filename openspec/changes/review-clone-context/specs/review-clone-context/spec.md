## ADDED Requirements

### Requirement: Clone target repo to temp directory

The review prompt SHALL instruct Claude to clone the PR's target repository to `/tmp/{owner}-{repo}` before performing the review.

#### Scenario: First review of a repo
- **WHEN** Claude starts a review and `/tmp/{owner}-{repo}` does not exist
- **THEN** Claude SHALL run `git clone` to clone the repository to `/tmp/{owner}-{repo}`

#### Scenario: Repeated review of the same repo
- **WHEN** Claude starts a review and `/tmp/{owner}-{repo}` already exists
- **THEN** Claude SHALL run `git fetch` and checkout the PR's base branch to ensure the local copy is up to date

#### Scenario: Clone failure due to access denied
- **WHEN** Claude attempts to clone a repo it does not have access to
- **THEN** the git clone command fails with a visible error, and Claude reports the access issue to the user

### Requirement: Autonomous context discovery

The review prompt SHALL instruct Claude to search the cloned repo for project convention and configuration files, rather than relying on a hardcoded list.

#### Scenario: Repo with CLAUDE.md
- **WHEN** the cloned repo contains a `CLAUDE.md` file
- **THEN** Claude discovers and reads it as part of understanding project conventions

#### Scenario: Repo with linter configs
- **WHEN** the cloned repo contains linter configuration files (e.g., `.golangci.yml`, `.eslintrc.*`, `pyproject.toml`)
- **THEN** Claude discovers and reads them to inform its review

#### Scenario: Repo with additional convention files
- **WHEN** the cloned repo contains convention files not in the previous hardcoded list (e.g., `.editorconfig`, `CONTRIBUTING.md`, `Makefile`)
- **THEN** Claude MAY discover and use them if relevant to the review

#### Scenario: Repo with no convention files
- **WHEN** the cloned repo contains no recognizable convention files
- **THEN** Claude proceeds with the review using only the PR diff and general review criteria
