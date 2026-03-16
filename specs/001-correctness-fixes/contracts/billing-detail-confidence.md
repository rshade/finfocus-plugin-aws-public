# Contract: BillingDetail Confidence Signal

**Version**: 1.0
**Date**: 2026-02-16

## Overview

Extends the `GetProjectedCostResponse.BillingDetail` string field with
structured bracket suffixes to signal property confidence levels.

## Format Specification

### Full Format

```text
{description} [defaults:{key}={value},{key}={value}] [confidence:{level}]
```

### Grammar

```text
billing_detail   = description [SP defaults_tag] [SP confidence_tag]
description      = *CHAR                      ; existing human-readable text
defaults_tag     = "[defaults:" defaults "]"
defaults         = default_pair *("," default_pair)
default_pair     = key "=" value
key              = 1*ALPHA_LOWER              ; property name (lowercase)
value            = 1*(ALPHA / DIGIT / ".")    ; default value applied
confidence_tag   = "[confidence:" level "]"
level            = "high" / "medium" / "low"
```

### Parsing Regex

```text
defaults:   \[defaults:([^\]]+)\]
confidence: \[confidence:(high|medium|low)\]
```

### Examples

**EBS with defaulted size (low confidence)**:

```text
EBS gp3 storage, 8GB (defaulted) [defaults:size=8] [confidence:low]
```

**RDS with all explicit properties (high confidence)**:

```text
RDS db.t3.micro, PostgreSQL, gp2, 100GB [confidence:high]
```

**RDS with partial defaults (medium confidence)**:

```text
RDS db.t3.micro, MySQL (defaulted), gp2, 20GB (defaulted) [defaults:engine=mysql,allocatedStorage=20] [confidence:medium]
```

**Lambda with all defaults (low confidence)**:

```text
Lambda x86_64, 128MB, 0 req/mo, 100ms [defaults:memory=128,requests=0,duration=100,architecture=x86_64] [confidence:low]
```

## Backward Compatibility

- Existing consumers that parse `BillingDetail` as human-readable text
  will see the bracket suffixes but they do not affect readability.
- Consumers that do not look for bracket patterns are unaffected.
- The `[defaults:...]` tag is OMITTED when no defaults are applied
  (high confidence), keeping the output clean for the common case.
- The `[confidence:...]` tag is ALWAYS present when the feature is
  active.

## Confidence Level Computation

```text
applicable = total properties that CAN be defaulted for this service
defaulted  = count of properties that WERE defaulted this request
ratio      = defaulted / applicable

high:   ratio == 0.0
medium: 0.0 < ratio < 0.5
low:    ratio >= 0.5

Special case: applicable == 0 (e.g., EC2) → always "high"
```
