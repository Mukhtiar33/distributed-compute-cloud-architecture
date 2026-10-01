# Document 05 — User Roles (Provider, Consumer, Hybrid)

## Objective & Scope
Define the responsibilities, capabilities, and trust posture of each role, and how the platform treats a participant who is both.

## 1. Consumer
- **Interacts via:** zip upload (project + manifest) → zip download (results), per Document 01 §1.3.
- **Never handles:** hardware selection, environment setup, scheduling, worker management.
- **Trust posture:** untrusted by default. Every submission passes static + dynamic scanning (Document 01 §2) before touching a worker.
- **Responsibilities:** provide a valid manifest (job type, environment identifier, entrypoint, inputs for parallel jobs, expected output shape). No responsibility for splitting/sharding (Document 01 §5, revised).

## 2. Provider
- **Interacts via:** Worker Agent, a single downloadable background service (Document 01 §1.2).
- **Never handles:** manual environment installation, manual job management — the agent handles environment caching, sandbox lifecycle, and status reporting autonomously.
- **Trust posture:** untrusted by default, symmetric to the consumer. Provider machine output is validated in three stages before reaching the consumer (Document 01 §2 revision).
- **Responsibilities:** keep the Worker Agent running if they want to earn credits/participate; the platform is responsible for not disrupting their normal machine use (throttling, resource limits).

## 3. Hybrid (Provider + Consumer)
- A single individual may hold both identities simultaneously (e.g., a Pilot participant who both submits jobs and contributes compute).
- **Critical isolation requirement:** a hybrid user's own submitted jobs must never be preferentially routed to their own worker, and their worker must not have privileged access to their own consumer data beyond what any other worker would have. This prevents a hybrid user from using their dual role to bypass isolation guarantees (e.g., inspecting their own job's execution in ways the sandbox wouldn't normally allow, or trusting their own submissions without scanning).
- Identity separation: enrollment/session credentials (Document 01 §3) for the Worker Agent identity are entirely separate from whatever authentication the Consumer interface uses — a hybrid user has two distinct identities in the system, not one dual-purpose one.

## 4. Role Summary Table

| Capability | Consumer | Provider | Platform-Mediated |
|---|---|---|---|
| Submit jobs | Yes | No | — |
| Contribute compute | No | Yes | — |
| Choose environment/hardware | No | No | Control Plane decides |
| See other participants' data | No | No | Never — enforced by isolation (Document 01 §4) |
| Direct communication with counterpart | No | No | All mediated through Control Plane |

## Senior Review
**Security Notes:** The hybrid-user isolation requirement (Section 3) is the one genuinely novel risk in this document — it's an easy case to overlook because "it's just one person" feels lower-risk than it is. Worth a specific test case in the Pilot given friends will likely hold both roles.
**Open Questions:** Does the Pilot's manual onboarding process issue separate consumer and provider identities by default, or does a single sign-up implicitly grant both? Recommend separate identities issued explicitly, even if the same person requests both, to keep the isolation boundary enforceable in code rather than by convention.
