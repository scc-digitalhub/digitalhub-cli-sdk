# digitalhub-cli-sdk

A Go SDK for interacting programmatically with **DigitalHub Core API**.

It provides programmatic access to the same capabilities offered by the DigitalHub CLI (`dhcli`), but is completely standalone and can be imported into any Go application or microservice.

---

## ✨ Highlights

- Official Core API v1 support
- CRUD operations on all core resources (projects, artifacts, functions, runs, tasks, etc.)
- Function execution (`RunService`)
- Stop / Resume for runnable resources
- Logs and Metrics retrieval (same semantics as `dhcli`)
- S3-compatible transfer (Upload/Download via MinIO / AWS S3)
- Task auto-creation for Run (e.g. `python+job` → `python+job:run`)
- Backwards compatible with `dhcli` logic
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
Configuration can be provided via environment variables (recommended for external programs).

```go
package main

import (
	"os"

	"github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"
)

func main() {
	cfg := config.Config{
		Core: config.CoreConfig{
			BaseURL:      os.Getenv("DHCORE_ENDPOINT"),
			APIVersion:   os.Getenv("DHCORE_API_VERSION"),
			AccessToken:  os.Getenv("DHCORE_ACCESS_TOKEN"),
		},
	}
	_ = cfg
}
```

### Required environment variables

| Variable            | Example                       |
| ------------------- | ----------------------------- |
| DHCORE_ENDPOINT     | https://core.dev.atlas.fbk.eu |
| DHCORE_API_VERSION  | v1                            |
| DHCORE_ACCESS_TOKEN | eyJhbGciOi...                 |

### Optional S3/MinIO variables (TransferService)

These are used when you call `transfer.Upload(...)` / `transfer.Download(...)`.

| Variable                 | Example                              |
| ------------------------ | ------------------------------------ |
| AWS_ACCESS_KEY_ID        | A68K...                              |
| AWS_SECRET_ACCESS_KEY    | ...                                  |
| AWS_SESSION_TOKEN        | ...                                  |
| AWS_REGION               | us-east-1                            |
| AWS_ENDPOINT_URL         | https://minio-api.dev.atlas.fbk.eu   |
| S3_BUCKET                | datalake                             |

> Note: The SDK config struct uses `config.S3Config{...}`; you can map env vars however you prefer in your app.

---

## 🚀 Usage Examples

> In the examples below, **`endpoint`** is the Core resource endpoint (e.g. `"artifacts"`, `"functions"`, `"runs"`).
> If you start from CLI-like resource names, you can reuse the same mapping used by the CLI:
>
> ```go
> endpoint := utils.TranslateEndpoint(resource) // e.g. "run" -> "runs"
> ```

---

### CRUD: list resources (all pages)

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
		Params: map[string]string{
			"size": "200",
			"sort": "updated,asc",
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

### CRUD: get a resource

#### Get by ID

```go
body, _, err := svc.Get(ctx, crud.GetRequest{
	ResourceRequest: crud.ResourceRequest{
		Project:  "project-name",
		Endpoint: "artifacts",
	},
	ID: "artifact-id",
})
_ = body
_ = err
```

#### Get latest version by name (CLI-compatible semantics)

```go
body, _, err := svc.Get(ctx, crud.GetRequest{
	ResourceRequest: crud.ResourceRequest{
		Project:  "project-name",
		Endpoint: "artifacts",
	},
	// When ID is empty, Get uses:
	// ?name=<name>&versions=latest
	Name: "my-artifact",
})
_ = body
_ = err
```

---

### CRUD: create a resource (from file or name)

```go
package main

import (
	"context"
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

	err = svc.Create(ctx, crud.CreateRequest{
		ResourceRequest: crud.ResourceRequest{
			Project:  "project-name",
			Endpoint: "artifacts",
		},
		// One of:
		// - FilePath: "/path/to/resource.yaml"
		// - Name:     "my-resource"
		FilePath: "/path/to/artifact.yaml",
		ResetID:  false,
	})
	if err != nil {
		panic(err)
	}
}
```

---

### CRUD: update a resource (raw JSON body)

```go
body := []byte(`{"project":"project-name","kind":"...","spec":{"foo":"bar"}}`)

err := svc.Update(ctx, crud.UpdateRequest{
	ResourceRequest: crud.ResourceRequest{
		Project:  "project-name",
		Endpoint: "artifacts",
	},
	ID:   "artifact-id",
	Body: body,
})
if err != nil {
	panic(err)
}
```

---

### CRUD: delete a resource (by ID or by name)

```go
// Delete by ID
err := svc.Delete(ctx, crud.DeleteRequest{
	ResourceRequest: crud.ResourceRequest{
		Project:  "project-name",
		Endpoint: "artifacts",
	},
	ID:      "artifact-id",
	Cascade: false,
})
if err != nil {
	panic(err)
}

// Delete all versions by name
err = svc.Delete(ctx, crud.DeleteRequest{
	ResourceRequest: crud.ResourceRequest{
		Project:  "project-name",
		Endpoint: "artifacts",
	},
	Name:    "my-artifact",
	Cascade: false,
})
if err != nil {
	panic(err)
}
```

---

## ▶️ Run / Stop / Resume (RunService)

The `run` service replicates the CLI behavior:
- resolves functions by name or ID
- finds or creates tasks
- creates runs with the correct `kind` (e.g. `python+job` → `python+job:run`)

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

	// Create a run (kind is normalized to "<taskKind>:run")
	req := run.RunRequest{
		Project:      "project-name",
		TaskKind:     "python+job",   // normalized to "python+job:run"
		FunctionName: "test-run",     // or FunctionID
		// FunctionID: "....",
		InputSpec: map[string]any{
			"input": 123,
		},
	}
	if err := runSvc.Run(ctx, req); err != nil {
		panic(err)
	}

	// Stop / Resume a runnable resource (e.g. runs)
	_, _, _ = runSvc.Stop(ctx, run.StopRequest{
		RunResourceRequest: run.RunResourceRequest{
			Project:  "project-name",
			Endpoint: "runs",
			ID:       "run-id",
		},
	})

	_, _, _ = runSvc.Resume(ctx, run.ResumeRequest{
		RunResourceRequest: run.RunResourceRequest{
			Project:  "project-name",
			Endpoint: "runs",
			ID:       "run-id",
		},
	})
}
```

The SDK automatically generates the correct run kind:

```json
{
  "kind": "python+job:run"
}
```

---

## 📜 Logs (CLI-compatible semantics)

Logs are retrieved via the same `/logs` API used by the CLI, and the default container name is inferred from `spec.task` if you don’t provide one.

```go
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	svc, err := run.NewRunService(ctx, cfg)
	if err != nil {
		panic(err)
	}

	// 1) Get logs list
	logBody, _, err := svc.GetLogs(ctx, run.LogRequest{
		RunResourceRequest: run.RunResourceRequest{
			Project:  "project-name",
			Endpoint: "runs",
			ID:       "run-id",
		},
	})
	if err != nil {
		panic(err)
	}

	// 2) Parse logs and print the main container (same as CLI)
	var logs []interface{}
	if err := json.Unmarshal(logBody, &logs); err != nil {
		panic(err)
	}

	// Compute default container like the CLI:
	// - GET resource to read spec.task
	// - container name is: c-<taskFormatted>-<id>
	resBody, _, err := svc.GetResource(ctx, run.LogRequest{
		RunResourceRequest: run.RunResourceRequest{
			Project:  "project-name",
			Endpoint: "runs",
			ID:       "run-id",
		},
	})
	if err != nil {
		panic(err)
	}
	var m map[string]interface{}
	_ = json.Unmarshal(resBody, &m)

	spec := m["spec"].(map[string]interface{})
	task := spec["task"].(string)
	taskFormatted := strings.ReplaceAll(task[:strings.Index(task, ":")], "+", "")
	mainContainer := fmt.Sprintf("c-%s-%s", taskFormatted, "run-id")

	// Find matching entry and print base64 decoded content
	for _, e := range logs {
		em := e.(map[string]interface{})
		st := em["status"].(map[string]interface{})
		if st["container"] == mainContainer {
			contentB64 := em["content"].(string)
			raw, _ := base64.StdEncoding.DecodeString(contentB64)
			fmt.Println(string(raw))
			break
		}
	}
}
```

> Tip: if you already know the container name, you can directly filter the logs list by `status.container`.

---

## 📈 Metrics (CLI-compatible semantics)

Metrics are read from the selected container log entry (`status.metrics`) exactly like the CLI.

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
	svc, err := run.NewRunService(ctx, cfg)
	if err != nil {
		panic(err)
	}

	// Prints pretty JSON or "No metrics for this run."
	if err := svc.PrintMetrics(ctx, run.MetricsRequest{
		RunResourceRequest: run.RunResourceRequest{
			Project:  "project-name",
			Endpoint: "runs",
			ID:       "run-id",
		},
		// Container: "" // optional; if empty, infer main container like CLI
	}); err != nil {
		panic(err)
	}
}
```

---

## ⬆️⬇️ Upload / Download (S3 / MinIO)

The transfer service uses:
- Core API to discover/download/upload resources
- S3-compatible object storage for the actual file transfer

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
		S3: config.S3Config{
			AccessKey:   os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretKey:   os.Getenv("AWS_SECRET_ACCESS_KEY"),
			AccessToken: os.Getenv("AWS_SESSION_TOKEN"),
			Region:      os.Getenv("AWS_REGION"),
			EndpointURL: os.Getenv("AWS_ENDPOINT_URL"),
		},
	}

	ctx := context.Background()

	tr, err := transfer.NewTransferService(ctx, cfg)
	if err != nil {
		panic(err)
	}

	// Upload (bucket is optional; your CLI uses "datalake" as default)
	_, err = tr.Upload(ctx, "artifacts", transfer.UploadRequest{
		Project:  "project-name",
		Resource: "artifacts",
		Name:     "my-artifact",
		Input:    "/tmp/data.bin",
		Verbose:  true,
		Bucket:   os.Getenv("S3_BUCKET"),
	})
	if err != nil {
		panic(err)
	}

	// Download
	_, err = tr.Download(ctx, "artifacts", transfer.DownloadRequest{
		Project:     "project-name",
		Resource:    "artifacts",
		Name:        "my-artifact",
		Destination: "/tmp/out",
		Verbose:     true,
	})
	if err != nil {
		panic(err)
	}
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
    transfer/
  utils/
```

The SDK is fully self-contained under `/sdk` and can be imported independently of the CLI.

---

## 🪪 License

Apache-2.0
