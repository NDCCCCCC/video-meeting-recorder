# scripts/check_db

Two diagnostic tools that historically shared `package main` in the same
directory, which made `go build ./...` fail with `main redeclared in this block`.

After commit d7db3a8 (this directory's last refactor) the conflict is resolved
via Go build tags — each file declares an exclusive tag so the compiler only
sees one `main()` per invocation:

| File              | Build tag    | Purpose                                              |
|-------------------|--------------|------------------------------------------------------|
| `main.go`         | `role`       | Drop legacy `users.role_id` column (sq >=3.35.0)     |
| `diagnose_user.go`| `diagnose`   | List AD users + role assignments + orphan detection   |

## Usage

```bash
# Drop role_id (after migration 012):
cd scripts/check_db
go build -tags role -o check_db_role .
./check_db_role

# Diagnose AD user role assignments:
go build -tags diagnose -o check_db_diag diagnose_user.go
./check_db_diag

# Sanity: no build tag → no package matched (exit 0):
go build ./...                  # "matched no packages", no error
```

The previous workflow was `go run main.go` / `go run diagnose_user.go` per-file,
which worked but was undocumented and tripped up any new contributor trying
`go build ./...` or `go test ./...` here. Build tags make the disjoint `main()`
declarations visible to the toolchain.

> **Note**: IDEs that run `go list` without `-tags` (e.g. gopls in default
> mode) will report `No packages found for open file` for both files. This is
> expected — pick the matching tag in your IDE's Go extension settings, or rely
> on the explicit `go build -tags …` invocations above.