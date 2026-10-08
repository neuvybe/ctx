# Settings storage

Each key uses `<key>.json` in the store directory. The JSON object contains one
integer field named `value`; files are written with mode `0600`.
The implementation calls `os.WriteFile` directly. It does not write a temporary
file, rename it, call `Sync`, or make a crash-safety guarantee.
