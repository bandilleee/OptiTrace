# Bounded verification policy contract

These are synthetic South African demonstration policies. A version consists of a supported profile and fixed thresholds (60 seconds source freshness, inclusive 24 hours for a recent event). Arbitrary Rego is not accepted. `GenerateRego` compiles this bounded table into embedded OPA rules.

| Profile | Number verification | SIM swap | Device swap |
|---|---|---|---|
| complete | compulsory | compulsory | compulsory |
| conditional | compulsory | optional | optional |

All compulsory observations must be available and valid before any positive/adverse outcome. A failed compulsory check returns `indeterminate`, even when another check is adverse. A valid adverse observation requires additional verification. With no adverse observation, every observation must be valid and clear to return `verified`. Optional failure can coexist with an adverse outcome; it can never establish `verified`.

Unfetched, unavailable, stale, timeout, rate-limited and permission-denied observations remain distinct in traces. Internally the proof groups resolved failures for decision equivalence, without erasing their recorded source state. The oracle enumerates clear/adverse/unavailable completions of every unfetched optional observation, then requires all compulsory work to have completed or explicitly failed.

Number mismatch is an explicit boolean. SIM/device changes carry an event timestamp; an adverse event without that timestamp is unavailable. Sources must identify the synthetic subject and `ZA:account-verification` scope. Source timestamps may not be in the future; age exactly 60 seconds remains fresh. Events exactly 24 hours old remain recent. Future events are unavailable. No event plus explicit false means no observed change.

The planner enumerates 13 ordered partitions of three capabilities, covering sequential orders, one parallel group, and mixed groups. Candidate cost estimates come from a fixed eight-row correlated tuning table, never from a request's unobserved source data. Estimated worst tuning-row latency bounds eligibility. Exhaustive completion proof controls stopping independently of estimates. This is an exact bounded enumerator, not a general-purpose optimiser of arbitrary policy programs.
