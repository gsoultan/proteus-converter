# Quarterly Engineering Report

**Prepared by:** Platform team  
**Period:** Q3 2026

## Summary

The platform team shipped *three* major features this quarter and reduced p99 latency by **38%**.
Error budgets were respected in every month, and on-call load fell for the second quarter in a row.

1. Launched the new conversion service
2. Migrated storage to the new cluster
   - zero downtime
   - 4.2 TB moved
3. Retired the legacy queue

## Metrics

| Metric            | Q2      | Q3      | Change |
|:------------------|--------:|--------:|:------:|
| Requests / day    | 1.2M    | 1.9M    | +58%   |
| p50 latency (ms)  | 41      | 33      | -20%   |
| p99 latency (ms)  | 380     | 236     | -38%   |
| Error rate        | 0.21%   | 0.09%   | -57%   |

## Code

```go
func Convert(ctx context.Context, src []byte) ([]byte, error) {
	doc, err := parse(src)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return render(ctx, doc)
}
```

Inline `code spans` sit next to normal text.

## Follow-ups

- [x] Write the post-mortem
- [ ] Add alerting on queue depth
- [ ] Review capacity plan

> **Note:** numbers are preliminary and may change after the finance review.
>
> > Nested quote level two.

Term
: A definition list entry from the Markdown extension.

Here is a footnote reference[^1] and a second one[^2].

---

### Appendix

Line one with a hard break  
line two after it. Visit <https://example.com/docs> for details.

[^1]: The first footnote.
[^2]: The second footnote with **bold** text.
