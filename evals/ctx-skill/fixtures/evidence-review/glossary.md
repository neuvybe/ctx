# Glossary — booking

<!-- ctx:doc {"status":"verified","verifiedAt":"{{SOURCE_COMMIT}} @ {{VERIFIED_DATE}}","sources":["booking/service.go","docs/requirements.md"]} -->

| Term | Project meaning |
|---|---|
| Session | A bookable event with fixed seat capacity, not a login session |
| Confirmed | A booking whose seats consume session capacity, not payment confirmation |
| Cancelled | A retained booking that no longer consumes seats |
| Availability | Capacity minus confirmed seats, computed on demand |
