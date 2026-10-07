# Booking demo

A small in-memory Go library for reserving seats in a session. It has no HTTP
server, database, payment provider, background expiration, or persistent state.

`booking.Service` creates sessions, accepts confirmed bookings, cancels them,
and reports available seats. The terminology and domain requirements are in
[requirements](docs/requirements.md); inspect source and tests for current
behavior.

Run the tests from this repository's root:

```bash
go test ./...
```

This project is a disposable fixture for testing repository-context workflows,
not a production booking system.
