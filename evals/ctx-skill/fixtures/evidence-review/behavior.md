# Behavior — booking

<!-- ctx:doc {"status":"verified","verifiedAt":"{{SOURCE_COMMIT}} @ {{VERIFIED_DATE}}","sources":["booking/service.go","booking/service_test.go","docs/requirements.md"]} -->

## Implemented rules

Sessions have fixed positive capacity. A booking names an existing session,
uses a unique caller-supplied identifier, and requests one to
`MaxSeatsPerBooking` seats. Email is trimmed and lowercased. Successful bookings
are immediately confirmed; only confirmed bookings consume capacity.

Cancellation retains the booking, marks it cancelled, and releases its seats.
Repeat cancellation succeeds without releasing twice. Cancelled identifiers
cannot be reused. These rules are explicit in `docs/requirements.md` and
implemented in `booking/service.go`.

State belongs to one `Service` instance. There is no persistence; a new service
starts empty, and a process restart loses state. The requirements explicitly
choose process-memory state.

## Test evidence

`TestConfirmedBookingsRespectCapacityAndCancellation` directly verifies that
each rejected booking leaves every booking field and every session's
availability unchanged. It also asserts normalized email, rejects retrieval of
the failed overflow booking, cancels the same booking twice, checks restored
capacity, and rejects reuse of the cancelled identifier.

`TestInvalidRequestsAndProcessMemoryBoundary` directly tests process-restart
data loss for sessions and bookings, alongside invalid-input cases.

`TestConcurrentBookingsCannotOversubscribe` races two one-seat bookings against
a capacity-one session, asserting exactly one success and one `ErrNoSeats`.
The mutex implementation supports concurrent use beyond that specific scenario;
those other combinations are not individually asserted here.
