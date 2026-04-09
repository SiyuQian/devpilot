## ADDED Requirements

### Requirement: Confidence scoring round
The system SHALL invoke a scoring model (default Haiku) to independently evaluate each finding from Round 1, assigning a confidence score from 0-100.

#### Scenario: Score individual finding
- **WHEN** Round 2 receives a finding with file path, line number, severity, title, explanation, and the relevant diff context
- **THEN** the scoring model evaluates whether the finding is a real issue or false positive and returns a score 0-100

#### Scenario: Batch scoring
- **WHEN** there are multiple findings to score
- **THEN** the system batches all findings into a single Round 2 invocation (not one invocation per finding) to minimize latency

### Requirement: Confidence scale
The system SHALL use the following confidence scale for scoring.

#### Scenario: Score 0-25 (false positive)
- **WHEN** a finding does not stand up to light scrutiny, is a pre-existing issue, or is something a linter/compiler would catch
- **THEN** the scorer assigns 0-25

#### Scenario: Score 25-49 (low confidence)
- **WHEN** a finding might be real but could also be a false positive, or is a stylistic issue not called out in project conventions
- **THEN** the scorer assigns 25-49

#### Scenario: Score 50-74 (moderate confidence)
- **WHEN** a finding is verified as a real issue but may be a nitpick or unlikely to occur in practice
- **THEN** the scorer assigns 50-74

#### Scenario: Score 75-100 (high confidence)
- **WHEN** a finding is verified as a real issue that will likely be hit in practice and directly impacts functionality, or is explicitly called out in project conventions
- **THEN** the scorer assigns 75-100

### Requirement: False positive definitions
The scoring prompt SHALL include explicit false positive definitions to calibrate the scorer.

#### Scenario: Pre-existing issues
- **WHEN** a finding describes an issue that existed before this PR
- **THEN** the scorer treats it as a false positive (score 0-25)

#### Scenario: Linter/compiler-catchable issues
- **WHEN** a finding describes something a linter, type checker, or compiler would catch (imports, type errors, formatting)
- **THEN** the scorer treats it as a false positive (score 0-25)

#### Scenario: Intentional behavior changes
- **WHEN** a finding flags a functionality change that is directly related to the PR's stated purpose
- **THEN** the scorer treats it as a false positive (score 0-25)

#### Scenario: Lines not modified in PR
- **WHEN** a finding references lines that were not modified in the PR diff
- **THEN** the scorer treats it as a false positive (score 0-25)

### Requirement: Confidence threshold filtering
The system SHALL filter out findings scoring below the configured threshold (default 50).

#### Scenario: Default threshold
- **WHEN** no `--threshold` flag is provided
- **THEN** findings with score < 50 are excluded from the final output

#### Scenario: Custom threshold
- **WHEN** user provides `--threshold 75`
- **THEN** only findings scoring ≥ 75 are included in the final output

#### Scenario: Threshold zero shows all
- **WHEN** user provides `--threshold 0`
- **THEN** all findings from Round 1 are included regardless of score
