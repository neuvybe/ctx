# Architecture — booking

<!-- ctx:doc {"status":"verified","verifiedAt":"{{SOURCE_COMMIT}} @ {{VERIFIED_DATE}}","sources":["booking/service.go","booking/service_test.go","go.mod"]} -->

One Go package exposes an in-process `Service` holding session and booking
maps. `New` allocates both maps. Public methods lock one `sync.Mutex`; returned
bookings are value copies. No persistence or background work exists.

`Book` validates normalized input, rejects duplicate identifiers, looks up the
session, checks remaining capacity, then stores a confirmed record. Rejection
paths return before that store. `Cancel` changes status rather than deleting;
an already-cancelled booking returns successfully without another mutation.

`Available` subtracts `usedSeats` from session capacity. `usedSeats` scans all
bookings and counts only confirmed seats in that session, under the mutex.
Its work grows with the total number of stored bookings, including cancelled
entries traversed by the scan. Source reasoning supports that complexity; no
performance benchmark or scaling target is recorded.

The module uses only the standard library (`go.mod`). See
[behavior](behavior.md) for product requirements and specific test evidence.
