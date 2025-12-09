# digitalhub-cli-sdk

A Go SDK for interacting programmatically with **DigitalHub Core API**.

It provides programmatic access to the same capabilities offered by the DigitalHub CLI (`dhcli`), but is completely standalone and can be imported into any Go application or microservice.

---

## ✨ Highlights

- Official Core API v1 support
- CRUD operations on all core resources (projects, artifacts, functions, runs, tasks, etc.)
- Function execution (`RunService`)
- S3-compatible transfer (`TransferService`)
- Logs (`LogService`)
- Metrics (`MetricsService`)
- Task auto-creation for Run
- Backwards compatible with dhcli logic
- No CLI dependency
- Usable standalone inside external Go applications

---

## 📦 Installation

```bash
go get github.com/scc-digitalhub/digitalhub-cli-sdk/sdk
```
