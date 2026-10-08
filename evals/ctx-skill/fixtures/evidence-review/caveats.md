# Caveats — booking

<!-- ctx:doc {"status":"verified","verifiedAt":"{{SOURCE_COMMIT}} @ {{VERIFIED_DATE}}","sources":["booking/service.go","docs/requirements.md","README.md"]} -->

## Process-memory state

- Status: accepted limitation.
- Impact: a new service or process restart does not retain sessions/bookings.
- Evidence: `New` and the maps in `booking/service.go`.
- Owner decision: `docs/requirements.md` explicitly requires process-memory
  state and states that restarting or constructing a new service retains none.

## Linear capacity accounting

- Status: accepted limitation.
- Impact: capacity checks and availability queries scan all stored bookings.
- Evidence: `usedSeats` in `booking/service.go`.
- Owner decision: the owner accepted the linear scan as a scalability tradeoff
  appropriate for this fixture, as recorded in `docs/requirements.md`.

## Retained cancelled identifiers

- Status: required behavior.
- Impact: cancellation does not allow an identifier to be reused.
- Evidence and owner decision: `docs/requirements.md`, plus `Book` and `Cancel`
  in `booking/service.go`. This is not a documented eviction or memory-growth
  policy; no owner decision about retention limits is recorded.
