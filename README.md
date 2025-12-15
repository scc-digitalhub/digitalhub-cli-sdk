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
go get github.com/scc-digitalhub/digitalhub-cli-sdk
```

---

## 🔧 Configuration

The SDK uses the same configuration model as the DigitalHub CLI.
Configuration can be provided via environment variables.

```go
import (
    "os"

    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"
)

cfg := config.Config{
    Core: config.CoreConfig{
        BaseURL:     os.Getenv("DHCORE_ENDPOINT"),
        APIVersion: os.Getenv("DHCORE_API_VERSION"),
        AccessToken: os.Getenv("DHCORE_ACCESS_TOKEN"),
    },
}
```

### Required environment variables

| Variable            | Example                       |
| ------------------- | ----------------------------- |
| DHCORE_ENDPOINT     | https://core.dev.atlas.fbk.eu |
| DHCORE_API_VERSION  | v1                            |
| DHCORE_ACCESS_TOKEN | eyJhbGciOi...                 |

---

## 🚀 Usage Examples

### List resources

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"

    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"
    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/services/crud"
)

func main() {
    cfg := config.Config{
        Core: config.CoreConfig{
            BaseURL:     os.Getenv("DHCORE_ENDPOINT"),
            APIVersion:  os.Getenv("DHCORE_API_VERSION"),
            AccessToken: os.Getenv("DHCORE_ACCESS_TOKEN"),
        },
    }

    ctx := context.Background()

    svc, err := crud.NewCrudService(ctx, cfg)
    if err != nil {
        panic(err)
    }

    items, _, err := svc.ListAllPages(ctx, crud.ListRequest{
        ResourceRequest: crud.ResourceRequest{
            Project:  "project-name",
            Endpoint: "artifacts",
        },
    })
    if err != nil {
        panic(err)
    }

    out, _ := json.MarshalIndent(items, "", "  ")
    fmt.Println(string(out))
}
```

---

### Get a resource

#### Get by ID

```go
body, _, err := svc.Get(ctx, crud.GetRequest{
    ResourceRequest: crud.ResourceRequest{
        Project:  "project-name",
        Endpoint: "artifacts",
    },
    ID: "artifact-id",
})
```

#### Get latest version by name

```go
body, _, err := svc.Get(ctx, crud.GetRequest{
    ResourceRequest: crud.ResourceRequest{
        Project:  "project-name",
        Endpoint: "artifacts",
    },
    Name: "my-artifact",
})
```

---

### Run a function

```go
package main

import (
    "context"
    "os"

    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"
    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/services/run"
)

func main() {
    cfg := config.Config{
        Core: config.CoreConfig{
            BaseURL:     os.Getenv("DHCORE_ENDPOINT"),
            APIVersion:  os.Getenv("DHCORE_API_VERSION"),
            AccessToken: os.Getenv("DHCORE_ACCESS_TOKEN"),
        },
    }

    ctx := context.Background()

    runSvc, err := run.NewRunService(ctx, cfg)
    if err != nil {
        panic(err)
    }

    req := run.RunRequest{
        Project:      "project-name",
        TaskKind:     "python+job",
        FunctionName: "test-run",
        InputSpec: map[string]any{
            "input": 123,
        },
    }

    if err := runSvc.Run(ctx, req); err != nil {
        panic(err)
    }
}
```

The SDK automatically generates the correct run kind:

```json
{
  "kind": "python+job:run"
}
```

---

### Logs

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"
    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/services/logs"
)

func main() {
    cfg := config.Config{
        Core: config.CoreConfig{
            BaseURL:     os.Getenv("DHCORE_ENDPOINT"),
            APIVersion:  os.Getenv("DHCORE_API_VERSION"),
            AccessToken: os.Getenv("DHCORE_ACCESS_TOKEN"),
        },
    }

    ctx := context.Background()

    logSvc, err := logs.NewLogService(ctx, cfg)
    if err != nil {
        panic(err)
    }

    raw, _, err := logSvc.GetLogs(ctx, logs.LogRequest{
        Project:  "project-name",
        Endpoint: "runs",
        ID:       "run-id",
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(string(raw))
}
```

---

### Metrics

```go
package main

import (
    "context"
    "os"

    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"
    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/services/metrics"
)

func main() {
    cfg := config.Config{
        Core: config.CoreConfig{
            BaseURL:     os.Getenv("DHCORE_ENDPOINT"),
            APIVersion:  os.Getenv("DHCORE_API_VERSION"),
            AccessToken: os.Getenv("DHCORE_ACCESS_TOKEN"),
        },
    }

    ctx := context.Background()

    mSvc, err := metrics.NewMetricsService(ctx, cfg)
    if err != nil {
        panic(err)
    }

    _ = mSvc.PrintMetrics(ctx, metrics.MetricsRequest{
        Project:  "project-name",
        Endpoint: "runs",
        ID:       "run-id",
    })
}
```

---

### Upload / Download (S3 / MinIO)

```go
package main

import (
    "context"
    "os"

    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"
    "github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/services/transfer"
)

func main() {
    cfg := config.Config{
        Core: config.CoreConfig{
            BaseURL:     os.Getenv("DHCORE_ENDPOINT"),
            APIVersion:  os.Getenv("DHCORE_API_VERSION"),
            AccessToken: os.Getenv("DHCORE_ACCESS_TOKEN"),
        },
    }

    ctx := context.Background()

    tr, err := transfer.NewTransferService(ctx, cfg)
    if err != nil {
        panic(err)
    }

    _ = tr.UploadFile(ctx, transfer.UploadRequest{
        Project:  "project-name",
        Resource: "artifacts",
        Path:     "/tmp/data.bin",
    })
}
```

---

## 🧪 Running integration tests

```bash
export DHCORE_ENDPOINT=...
export DHCORE_API_VERSION=v1
export DHCORE_ACCESS_TOKEN=...

go test ./...
```

---

## 📁 Repository layout

```
sdk/
  config/
  services/
    crud/
    run/
    logs/
    metrics/
    transfer/
  utils/
```

The SDK is fully self-contained under `/sdk` and can be imported independently of the CLI.

---

## 🪪 License

Apache-2.0
