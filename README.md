# go_software_build
A Go package to make it easier to use autotools and such to automatically build and install software packages

## Versioning policy

This project follows Go module major-version rules.

- `v1.x.y` uses module path `github.com/gvallee/go_software_build`.
- Any breaking API change is released as a new major line with `/vN` suffix (for example `v2` => `github.com/gvallee/go_software_build/v2`).
- Within one major line, changes are backward compatible.
- Consumers that require API stability should pin to a major line (`v1.*`, `v2.*`, etc.).
