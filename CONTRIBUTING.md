# Contributing

## Ground rules

- **Never commit supplier or customer data.** Tests must use synthetic fixtures
  created in the test file or in `testdata/`. A real DATANORM file may be used
  locally to validate a change, but it must not end up in a commit, a test
  fixture, an issue, or a screenshot.
- Keep the public model in `productdata` free of format-specific imports.
  Format packages depend on `productdata`, never the other way around.
- Preserve raw data. When a field is not publicly specified, add it to the raw
  record instead of guessing its meaning.
- Add a stable diagnostic code for every recoverable problem. Fatal problems are
  returned as errors.

## Tests

```bash
go test ./...
go test -race ./...
go vet ./...
```

If you add a field mapping, add a table-driven test and update `SPEC.md` with the
confidence level and the evidence. "It appeared in one file" is `observed`, not
`specified`.

## Format changes

1. Add or adjust the record struct and its field table in `datanorm/field.go`.
2. Parse it in `datanorm/parser.go` and map it in `datanorm/assemble.go`.
3. Add diagnostics and validation.
4. Add a round-trip test if the encoder supports it.
5. Update `SPEC.md` and `docs/architecture.md`.

## Commit style

Use short imperative subjects, for example:

```
feat(datanorm): parse G media references
docs(spec): confirm A field 23 as text marker
fix(charset): prefer CP850 on ties
```

## License

By contributing you agree that your contribution is licensed under the MIT
license of this repository.
