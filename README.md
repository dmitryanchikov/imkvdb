# In-Memory Key-Value Store

## Main commands

1. **App launching**
   ```bash
   go run ./cmd/cli/main.go
   ```
2. **Tests launching**

```bash
go test ./...
```

3. **golangci-lint launching**

```bash
golangci-lint run ./...
```

4. **Cleanup (cache, build artefacts, etc.)**

```bash
go clean --modcache --cache --testcache
```