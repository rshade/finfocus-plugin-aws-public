---
description: Performs automated code review on specified target files or directories with best practice suggestions.
---
## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Goal

Perform an automated code review on the specified target files or directories. Detect the programming language and apply language-specific best practices and common issue detection. Provide actionable feedback to improve code quality.

## Operating Constraints

**STRICTLY READ-ONLY**: Do **not** modify any files. Output a structured analysis report with findings and suggestions.

**Language Support**: Only analyze files in supported languages (Go, JavaScript/TypeScript, Python). Skip unsupported files with a note.

## Execution Steps

### 1. Parse Arguments

Parse the user input to extract:
- Target file or directory (default: current directory)
- Output format (text, json, markdown - default: text)
- Minimum severity level (low, medium, high - default: medium)
- Language override (auto-detected by default)
- Fix suggestion flag (default: false)

### 2. Validate Target

Check that the target exists:
- If it's a file, ensure it exists
- If it's a directory, ensure it exists and is accessible
- If target is not specified, use current directory

### 3. Detect Language

If language is not explicitly specified:
- For directories: Scan for files with known extensions (.go, .js, .ts, .py)
- For files: Determine language from file extension
- If no supported files found, exit with appropriate message

### 4. Execute Language-Specific Review

Run the appropriate review function based on detected language:

#### For Go files:
- Check for proper error handling patterns
- Verify context usage and cancellation
- Validate mutex usage and locking patterns
- Check for proper resource management (defer statements)
- Look for common anti-patterns and best practice violations

#### For JavaScript/TypeScript files:
- Check for consistent async/await vs Promise usage
- Verify proper error handling patterns
- Look for var vs let/const usage
- Identify potential type safety issues (TypeScript)
- Check for common anti-patterns

#### For Python files:
- Check for proper exception handling
- Verify context manager usage for resource management
- Look for Python-specific best practices
- Identify common anti-patterns

### 5. Filter by Severity

Filter findings based on the specified minimum severity level:
- High: Critical issues that should be addressed immediately
- Medium: Important issues that should be considered
- Low: Minor issues or suggestions for improvement

### 6. Format Output

Format the results according to the specified output format:
- Text: Human-readable console output with colors
- JSON: Structured JSON output for programmatic consumption
- Markdown: Markdown formatted output for documentation

### 7. Present Results

Display the findings with:
- File path and line numbers where applicable
- Severity level
- Description of the issue
- Suggestion for improvement
- If fix suggestion is enabled, provide specific fix recommendations

## Output Format

### Text Format (default)
```
Code Review Tool
Target: [target]
Language: [language]
Severity: [severity]
Format: [format]

Performing [language] code review on: [target]
----------------------------------------
[HIGH/MEDIUM/LOW]: [issue description]
[File:line] [specific location details]
Suggestion: [improvement suggestion]

[HIGH/MEDIUM/LOW]: [issue description]
...

[language] code review completed with [n] potential issues
```

### JSON Format
```json
{
  "target": "[target]",
  "language": "[language]",
  "severity": "[severity]",
  "format": "[format]",
  "issues": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "description": "[issue description]",
      "file": "[file path]",
      "line": [line number],
      "suggestion": "[improvement suggestion]"
    }
  ],
  "summary": {
    "total_issues": [n],
    "high": [count],
    "medium": [count],
    "low": [count]
  }
}
```

### Markdown Format
```markdown
# Code Review Report

## Summary
- **Target**: [target]
- **Language**: [language]
- **Severity**: [severity]
- **Format**: [format]
- **Total Issues**: [n]

## Issues

### [HIGH/MEDIUM/LOW] [Issue Description]
- **File**: [file path]
- **Line**: [line number]
- **Details**: [specific location details]
- **Suggestion**: [improvement suggestion]

...
```

## Operating Principles

### Accuracy
- **Precise Detection**: Use reliable patterns for issue detection
- **Minimize False Positives**: Validate findings before reporting
- **Language-Specific**: Apply rules appropriate to each language

### Actionability
- **Clear Descriptions**: Provide specific, actionable issue descriptions
- **Practical Suggestions**: Offer concrete improvement suggestions
- **Contextual Examples**: Include relevant code examples when helpful

### Performance
- **Efficient Scanning**: Use optimized file scanning techniques
- **Minimal Overhead**: Keep analysis lightweight
- **Scalable**: Handle projects of various sizes effectively

## Context

$ARGUMENTS