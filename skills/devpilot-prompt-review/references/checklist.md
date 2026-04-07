# Claude Prompting Best Practices Checklist

This checklist is the knowledge base for prompt review. Each item includes the principle, why it matters, and what to look for when reviewing prompts.

Applies to: Claude Opus 4.6 / Sonnet 4.6 / Haiku 4.5

---

## Table of Contents

1. [Clarity and Directness](#1-clarity-and-directness)
2. [Motivational Context](#2-motivational-context)
3. [Few-Shot Examples](#3-few-shot-examples)
4. [Structure with XML Tags](#4-structure-with-xml-tags)
5. [Document Placement](#5-document-placement)
6. [Positive Framing](#6-positive-framing)
7. [Tool Use Calibration](#7-tool-use-calibration)
8. [Thinking and Reasoning](#8-thinking-and-reasoning)
9. [Agent Safety](#9-agent-safety)
10. [Agent Efficiency](#10-agent-efficiency)
11. [Model-Specific Awareness](#11-model-specific-awareness)

---

## 1. Clarity and Directness

**Principle:** Write prompts like instructions for a smart colleague who has zero context about your project. Be explicit about what you want — don't rely on the model inferring your intent.

**What to look for:**
- Vague or ambiguous instructions ("handle this appropriately", "do the right thing")
- Missing success criteria — what does a good output look like?
- Steps that assume prior knowledge the model doesn't have
- Implicit expectations not spelled out

**Good pattern:** Use numbered lists or bullet points for multi-step instructions to ensure ordering and completeness.

---

## 2. Motivational Context

**Principle:** Explaining *why* is more powerful than just saying *what*. When the model understands the purpose, it can generalize and make better judgment calls on edge cases.

**What to look for:**
- Instructions that say "do X" without explaining why X matters
- Constraints without rationale (the model may violate them if it doesn't understand the reason)
- Missing audience/consumer information — who will read/use the output?

**Good pattern:** "This output will be read aloud by a TTS system, so avoid abbreviations and ellipses" — the model can then also avoid other TTS-unfriendly patterns on its own.

---

## 3. Few-Shot Examples

**Principle:** 3-5 structured examples dramatically improve output accuracy and consistency. Examples teach format, tone, and edge-case handling more effectively than prose instructions.

**What to look for:**
- Complex format requirements with no examples
- Inconsistent output across invocations (signals missing examples)
- Examples mixed into instructions without clear separation

**Good pattern:** Wrap examples in `<example>` tags to separate them from instructions. Show both input and expected output. Include at least one edge case.

---

## 4. Structure with XML Tags

**Principle:** Use XML tags (`<instructions>`, `<context>`, `<input>`, etc.) to clearly separate different sections of a prompt. This significantly reduces misinterpretation, especially in complex prompts.

**What to look for:**
- Long prompts with no structural separation
- Context, instructions, and input data mixed together
- Model confusing input data for instructions

**Good pattern:** `<instructions>`, `<context>`, `<examples>`, `<input>` as distinct sections.

---

## 5. Document Placement

**Principle:** Place long reference documents at the top of the prompt, before queries and instructions. Testing shows this can improve response quality by up to 30%.

**What to look for:**
- Long reference text placed after instructions or at the bottom
- Large data blocks interleaved with instructions
- Missing "cite before answering" guidance for long-document tasks

**Good pattern:** Long document first, then instructions, then query. For long-document tasks, ask the model to quote relevant passages before answering.

---

## 6. Positive Framing

**Principle:** Tell the model what TO DO rather than what NOT to do. Positive instructions are more effective because they give the model a clear target behavior.

**What to look for:**
- Heavy use of "don't", "never", "avoid" without a positive alternative
- Lists of prohibited behaviors with no guidance on desired behavior
- "Don't use markdown" instead of "Respond in plain prose paragraphs"

**Good pattern:** Replace "Don't use bullet points" with "Write in flowing prose paragraphs." The model knows what to aim for.

---

## 7. Tool Use Calibration

**Principle:** New Claude models (4.5/4.6) are significantly more proactive about tool use. Aggressive forcing language from older prompts ("CRITICAL: You MUST use...", "ALWAYS call...") now causes overtriggering and should be softened.

**What to look for:**
- ALL-CAPS forcing language for tool calls ("MUST", "ALWAYS", "CRITICAL")
- Prompts written for older models that forced tool use to compensate for reluctance
- Missing action-oriented language — "suggest changes" instead of "make the changes"
- No proactivity calibration (`<default_to_action>` vs `<do_not_act_before_instructions>`)

**Good pattern:** Use calm, direct language. "Use the search tool to find relevant files" instead of "YOU MUST ALWAYS USE THE SEARCH TOOL". Add proactivity controls only when needed.

---

## 8. Thinking and Reasoning

**Principle:** Claude 4.6 uses adaptive thinking — it automatically calibrates thinking depth based on problem complexity. Over-specified thinking instructions (manual chain-of-thought, step-by-step breakdowns) can actually hurt performance.

**What to look for:**
- Manual chain-of-thought prompting ("First think about X, then consider Y, then...")
- Explicit `budget_tokens` instead of `effort` parameter (deprecated pattern)
- "Always think step by step" — generic instruction that the model handles better autonomously
- Missing self-verification for math/code tasks

**Good pattern:** Use `thinking: {type: "adaptive"}` with appropriate `effort` level. Add "Verify your answer against [criteria] before responding" for code/math. Let "think deeply" replace manual step breakdowns.

---

## 9. Agent Safety

**Principle:** Claude Opus 4.6 may proactively execute irreversible operations (file deletion, force push, database drops). Agent prompts must explicitly require confirmation before destructive actions.

**What to look for:**
- Agent prompts with no safety guardrails for destructive operations
- Missing "ask before executing irreversible actions" instruction
- No distinction between read-only exploration and write operations
- Overly autonomous agent prompts without human checkpoints

**Good pattern:** "Before executing any destructive or irreversible operation (deleting files, force pushing, dropping tables), describe what you plan to do and wait for confirmation."

---

## 10. Agent Efficiency

**Principle:** New models tend toward over-engineering — creating unnecessary abstractions, extra files, and premature generalization. Agent prompts should constrain scope to what's actually needed.

**What to look for:**
- No scope constraints ("only make the requested changes, keep the solution simple")
- Missing anti-hallucination guidance ("read files before making claims about them")
- Encouraging exploration before action when unnecessary ("investigate everything first")
- No guidance on subagent usage (new models may over-delegate)

**Good pattern:** "Only make the changes that were requested. Read relevant files before modifying them. Use subagents only for genuinely parallel or independent tasks, not for simple operations you can handle directly."

---

## 11. Model-Specific Awareness

**Principle:** Different model generations have different defaults. Prompts should be calibrated for the target model, not carry over legacy workarounds.

**What to look for:**
- Prefill patterns (deprecated in Claude 4.6 — use structured outputs or system instructions instead)
- Legacy anti-laziness prompts ("Don't be lazy", "Complete the entire task") — unnecessary for new models and may cause overcompensation
- Missing `max_tokens` guidance (recommend 64k for complex tasks)
- Sonnet-specific: default effort is `high`, which may cause unnecessary latency — consider `medium` for most applications
- Math expressions: Opus 4.6 defaults to LaTeX — specify "use plain text math" if needed

**Good pattern:** Review whether each instruction is still needed for the target model. Remove workarounds for older model limitations.
