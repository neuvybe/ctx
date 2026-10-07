# Booking requirements

The library manages sessions with a positive seat capacity and bookings with
a caller-supplied unique identifier.

- A successful booking immediately confirms its requested seats; there is no
  tentative hold or expiration lifecycle.
- A booking must name an existing session, have a nonempty email, and request
  a positive number of seats no greater than `MaxSeatsPerBooking`.
- Confirmed bookings consume session capacity. Requests above remaining
  capacity fail without creating a booking.
- Cancellation releases the booking's seats immediately. Repeating a
  cancellation succeeds without releasing seats twice.
- A booking identifier cannot be reused, including after cancellation.
- Emails are trimmed and lowercased when stored. One email may have multiple
  bookings; this is not an authentication or account system.
- The service owns only process-memory state. Restarting or constructing a new
  service does not retain sessions or bookings.

The source constant defines the current per-booking seat limit. Changes to
that limit require context reconciliation; these requirements do not define
any payment, access-control, refund, or cancellation-fee policy.
