# Settings requirements

- Keys contain only ASCII letters, digits, underscores, or hyphens and are
  nonempty. Values are nonnegative integers.
- A successful write replaces that key's value. Distinct keys remain separate.
- Invalid keys or values fail before changing stored settings.
- Missing keys return `ErrNotFound`.
- Successfully written files can be read by a later process using the same
  directory. Power-loss durability, concurrent writes to one key, encryption,
  and multi-key transactions are outside this fixture's requirements.

No performance target or decision accepting a particular I/O tradeoff is recorded.
