# Overview — booking

<!-- ctx:doc {"status":"verified","verifiedAt":"{{SOURCE_COMMIT}} @ {{VERIFIED_DATE}}","sources":["README.md","docs/requirements.md","booking/service.go"]} -->

This is a disposable in-memory Go library for reserving seats in sessions, not
a production booking system. Its callers are Go programs. It has no HTTP
server, database, payment provider, or background expiration (`README.md`).

The module is `example.invalid/booking-demo` and declares Go 1.21 in `go.mod`.
The API is `New`, `CreateSession`, `Book`, `Cancel`, `Available`, and
`GetBooking`. Bookings confirm immediately; cancelled identifiers stay reserved.

Read [behavior](behavior.md) for domain rules and their test evidence,
[architecture](architecture.md) for capacity accounting and locking, and
[caveats](caveats.md) for limitations and owner decisions.
