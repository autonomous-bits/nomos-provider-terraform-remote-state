---
name: skill-generator
description: Generate new Agent Skills following the open standard specification. Use this when asked to create a new skill, teach Copilot a new capability, or capture domain expertise as a reusable skill.
license: MIT
---

# Skill Generator

This skill helps you generate well-structured Agent Skills that follow the open standard specification and best practices for reusability and effectiveness.

## When to Use This Skill

Use this skill when you need to:
- Create a new Agent Skill from scratch
- Package domain expertise into a reusable skill
- Teach Copilot a new repeatable workflow
- Capture organizational knowledge in a portable format
- Convert existing procedures into skills

## Skill Generation Process

Follow these steps to generate a high-quality Agent Skill:

### 1. Identify the Skill Purpose

Ask clarifying questions to understand:
- **What problem does this skill solve?** Be specific about the use case
- **When should Copilot use this skill?** Define clear trigger conditions
- **What domain or workflow does it support?** (e.g., testing, debugging, deployment)
- **Who is the target audience?** (developers, DevOps, data scientists, etc.)

### 2. Assess Reusability

A good skill should be **reusable** and **generalizable**. Evaluate:
- ✅ **Good candidates for skills:**
  - Multi-step workflows that follow a consistent process
  - Domain-specific procedures requiring specialized knowledge
  - Tasks that benefit from scripts, templates, or examples
  - Processes that need to be repeatable and auditable
  - Capabilities that extend agent functionality

- ❌ **Poor candidates for skills:**
  - One-off tasks specific to a single file or situation
  - Simple instructions that could be custom instructions instead
  - Overly broad or vague procedures without clear steps
  - Tasks that require extensive human judgment without guidelines

### 3. Design the Skill Structure

Plan the skill's components:

```
.github/skills/[skill-name]/
├── SKILL.md           # Main skill file (required)
├── scripts/           # Optional: automation scripts
│   └── example.sh
├── templates/         # Optional: file templates
│   └── template.yaml
└── examples/          # Optional: example outputs
    └── example.md
```

### 4. Create the SKILL.md File

Generate a SKILL.md file with:

#### Required YAML Frontmatter
```yaml
---
name: skill-name              # lowercase, hyphens for spaces
description: Brief description of what the skill does and when to use it
license: MIT                   # optional but recommended
---
```

#### Skill Name Guidelines
- Use lowercase letters only
- Use hyphens to separate words (not underscores or spaces)
- Make it descriptive and memorable
- Match the directory name
- Examples: `github-actions-failure-debugging`, `webapp-testing`, `api-design-review`

#### Description Guidelines
- Start with what the skill does
- Include when Copilot should use it
- Be concise but specific (1-3 sentences)
- Example: "Guide for debugging failing GitHub Actions workflows. Use this when asked to debug failing GitHub Actions workflows."

#### Markdown Body Structure
Organize instructions clearly:

1. **Overview section**: Brief introduction to the skill's purpose
2. **When to use**: Clear trigger conditions
3. **Prerequisites**: Required tools, permissions, or context
4. **Step-by-step instructions**: Numbered workflow with specific actions
5. **Examples**: Concrete examples of inputs/outputs
6. **Best practices**: Tips for optimal results
7. **Troubleshooting**: Common issues and solutions (if applicable)

#### Writing Style Guidelines
- Use clear, imperative language ("Use the tool", "Follow this process")
- Be specific about tool names and parameters
- Include concrete examples
- Use numbered lists for sequential steps
- Use bullet points for options or related items
- Keep instructions focused and actionable
- Avoid vague language or assumptions

### 5. Add Supporting Resources

Include additional files when they enhance the skill:

**Scripts** (`scripts/` directory):
- Automation helpers
- Data processing utilities
- Integration scripts
- Use clear naming and include shebang lines

**Templates** (`templates/` directory):
- File templates (YAML, JSON, Markdown, etc.)
- Configuration examples
- Boilerplate code
- Include placeholder comments

**Examples** (`examples/` directory):
- Sample inputs and outputs
- Real-world use cases
- Before/after comparisons
- Include explanatory comments

### 6. Validate the Skill

Check your skill against these criteria:

#### Quality Checklist
- [ ] SKILL.md file exists and is properly named
- [ ] YAML frontmatter is valid with required fields
- [ ] Skill name is lowercase with hyphens
- [ ] Description clearly states what and when
- [ ] Instructions are clear, specific, and actionable
- [ ] Steps are numbered in logical order
- [ ] Examples are included where helpful
- [ ] Tool names and parameters are accurate
- [ ] Skill is placed in `.github/skills/[skill-name]/` directory
- [ ] Supporting resources are well-organized
- [ ] Content is free of repository-specific hardcoded values

#### Reusability Checklist
- [ ] Skill addresses a general class of problems
- [ ] Instructions work across different repositories/contexts
- [ ] No hardcoded paths, URLs, or credentials
- [ ] Domain expertise is clearly captured
- [ ] Workflow is repeatable and consistent
- [ ] Skill extends agent capabilities meaningfully

### 7. Test the Skill

Verify the skill works as intended:
1. Ask Copilot to perform the task the skill addresses
2. Check if Copilot discovers and uses the skill
3. Validate that instructions are followed correctly
4. Test with different variations of the task
5. Refine based on results

## Skill Format Specification

### Directory Structure
```
.github/skills/[skill-name]/
└── SKILL.md
```

Alternative location (also supported):
```
.claude/skills/[skill-name]/
└── SKILL.md
```

### SKILL.md Format
```markdown
---
name: required-lowercase-hyphenated-name
description: Required clear description with usage context
license: Optional license identifier
---

# Skill Title

Clear instructions in Markdown format...
```

## Best Practices

### For Skill Authors
1. **Start simple**: Begin with core functionality, add complexity later
2. **Be explicit**: Don't assume the agent knows domain-specific terminology
3. **Include examples**: Concrete examples prevent misinterpretation
4. **Version control**: Skills are code—commit them to version control
5. **Document changes**: Update skills as processes evolve
6. **Test thoroughly**: Verify skills work in different scenarios
7. **Keep focused**: One skill per specific capability or workflow
8. **Use tools**: Reference specific tool names when available

### For Skill Quality
1. **Clear triggers**: Make it obvious when to use the skill
2. **Actionable steps**: Each step should be concrete and executable
3. **Error handling**: Include troubleshooting guidance
4. **Constraints**: Specify limitations or prerequisites
5. **Context awareness**: Help agent understand when not to use the skill
6. **Maintainability**: Write skills that are easy to update

### For Reusability
1. **Parameterize**: Use variables instead of hardcoded values
2. **Generalize**: Abstract from specific instances to patterns
3. **Modularize**: Break complex skills into focused components
4. **Document**: Explain the why, not just the what
5. **Standardize**: Follow consistent patterns across skills
6. **Share**: Contribute useful skills to the community

## Common Skill Types

### Workflow Skills
Multi-step procedures following a consistent process:
- Debugging workflows
- Testing procedures
- Code review checklists
- Deployment processes
- CI/CD troubleshooting

### Domain Expertise Skills
Specialized knowledge in specific areas:
- API design guidelines
- Security review procedures
- Performance optimization
- Accessibility compliance
- Data analysis pipelines

### Tool Integration Skills
Working with specific tools or platforms:
- GitHub Actions integration
- Cloud platform operations
- Database migrations
- Testing framework usage
- Build system operations

### Code Generation Skills
Creating specific types of code or documents:
- Component generators
- Test case creation
- Documentation generation
- Configuration templates
- Boilerplate creation

## Constraints and Limitations

### Current Limitations
- Skills are repository-level only (organization/enterprise support coming)
- Skills must be in `.github/skills/` or `.claude/skills/` directories
- SKILL.md must be the exact filename (case-sensitive)
- Agents decide when to use skills based on description and context
- No programmatic skill chaining or dependencies

### Technical Constraints
- YAML frontmatter must be valid
- Markdown should be properly formatted
- File paths must be relative to skill directory
- Scripts must have appropriate permissions
- Large files may impact context window

### Best Practices for Constraints
- Keep skills focused and concise
- Optimize for context window usage
- Use external scripts for heavy processing
- Reference documentation, don't duplicate it
- Design for agent autonomy

## Example Skill Template

Here's a complete template for a new skill:

```markdown
---
name: your-skill-name
description: What your skill does and when to use it
license: MIT
---

# Your Skill Name

Brief overview of what this skill accomplishes.

## When to Use This Skill

Clear description of scenarios where this skill should be activated.

## Prerequisites

- Required tools or permissions
- Expected context or setup
- Dependencies

## Instructions

1. **First major step**
   - Specific action to take
   - Tool to use: `tool_name` with parameters
   - Expected outcome

2. **Second major step**
   - Detailed substeps
   - Examples of inputs/outputs
   - Decision points

3. **Final step**
   - Validation steps
   - Success criteria

## Examples

### Example 1: Common Use Case
Input:
\```
Example input
\```

Process:
- Step taken
- Result obtained

Output:
\```
Example output
\```

## Best Practices

- Tip for optimal results
- Common pitfall to avoid
- Performance considerations

## Troubleshooting

**Issue**: Common problem
**Solution**: How to resolve it

**Issue**: Another problem
**Solution**: Resolution steps
```

## Getting Started

To generate a new skill:

1. Gather requirements by asking clarifying questions
2. Assess whether a skill is the right solution
3. Design the skill structure and components
4. Write the SKILL.md file following the specification
5. Add supporting resources if needed
6. Validate against quality and reusability checklists
7. Test the skill with Copilot
8. Iterate based on feedback

## Resources

- [Agent Skills Open Standard](https://github.com/agentskills/agentskills)
- [Agent Skills Documentation](https://agentskills.io/)
- [Example Skills Repository](https://github.com/anthropics/skills)
- [GitHub Copilot Skills Guide](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills)
- [Awesome Copilot Collection](https://github.com/github/awesome-copilot)

## Output Format

When generating a skill, provide:

1. **Skill structure**: Directory layout with all files
2. **SKILL.md content**: Complete, formatted skill file
3. **Supporting files**: Scripts, templates, examples (if applicable)
4. **Usage guidance**: How to test and validate the skill
5. **Improvement suggestions**: Areas for future enhancement

Always create skills that are:
- **Clear**: Easy to understand and follow
- **Actionable**: Specific steps that can be executed
- **Reusable**: Applicable across multiple contexts
- **Maintainable**: Easy to update as processes evolve
- **Discoverable**: Description helps agents find them when needed
