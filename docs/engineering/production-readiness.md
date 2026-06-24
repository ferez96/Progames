# Production readiness checklist

## Security / stability

- [ ] Rate limit `/practice/run` and `/signup` — prevent Docker build queue saturation and signup spam
- [ ] Add `/healthz` (liveness) and `/readyz` (readiness) endpoints for container orchestrator probes

## Ops

- [ ] Reverse proxy config sample (nginx or Caddy) — TLS termination, `PROGAMES_FORCE_SECURE_COOKIE=true`
- [ ] Backup runbook — `progames.db` (SQLite file copy / `.backup`) and `artifacts/files/`

## Product

- [ ] Bot protocol spec — document the stdin/stdout wire format for bot authors
- [ ] Leaderboard / ranking — aggregate match results into an ELO or win-rate table
- [ ] Admin panel — disable agents, inspect queue, manage users
