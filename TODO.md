# TODO

This file tracks technical debt and upcoming tasks for ZoneKit.



## Priority 3 (Refactoring)

- [ ] Refactor `pkg/plugin` to align with the new v2.0.0 architecture if needed.
- [ ] Improve error handling in the `client` package.

## Known Issues

- Complex DNS records with multiple priority weights (like NAPTR) may not parse correctly from all Zone files.
