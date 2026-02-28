# Phase 1: Config Push API

> Config push endpoint for browser extension to auto-upload .ovpn files, with auto-deploy to existing tunnels.

## Task 1: Agent — Config Push Handler

**Files:**
- Modify: `backend/cmd/vpn-agent/config_store.go` — add `SaveWithMeta`, `GetMeta`, version tracking
- Modify: `backend/cmd/vpn-agent/server_handlers.go` — add `handlePushConfigs`
- Modify: `backend/cmd/vpn-agent/server.go` — register new route

**Step 1: Add config metadata tracking to ConfigStore**

Add to `config_store.go`:

```go
// ConfigMeta tracks upload history for a config file
type ConfigMeta struct {
	UploadedAt    time.Time `json:"uploaded_at"`
	ContentHash   string    `json:"content_hash"`
	Version       int       `json:"version"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	Stale         bool      `json:"stale"`
}

// ConfigStoreMeta holds all config metadata
type ConfigStoreMeta struct {
	Configs map[string]*ConfigMeta `json:"configs"`
}
```

Add methods:
```go
func (s *ConfigStore) SaveWithMeta(fileName string, content []byte) (*ConfigMeta, bool, error)
// Returns (meta, isNew, error). isNew=false if content hash matches existing.
// Increments version if content differs. Saves meta to configs_meta.json.

func (s *ConfigStore) GetMeta(name string) *ConfigMeta
func (s *ConfigStore) ListWithMeta() ([]OvpnConfigWithMeta, error)
func (s *ConfigStore) RecordSuccess(name string)  // Update last_success_at
func (s *ConfigStore) loadMeta() *ConfigStoreMeta
func (s *ConfigStore) saveMeta(meta *ConfigStoreMeta)
```

`OvpnConfigWithMeta` extends `OvpnConfig`:
```go
type OvpnConfigWithMeta struct {
	OvpnConfig
	UploadedAt    time.Time  `json:"uploaded_at"`
	Version       int        `json:"version"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	Stale         bool       `json:"stale"`
}
```

Content hash: `sha256(content)[:16]`

**Step 2: Add push handler**

Add to `server_handlers.go`:

```go
type pushConfigRequest struct {
	Configs []pushConfigItem `json:"configs"`
	ReplaceExisting bool     `json:"replace_existing"`
}

type pushConfigItem struct {
	Filename   string `json:"filename"`
	Content    string `json:"content"`     // base64 encoded
	Region     string `json:"region"`      // optional
	AutoDeploy bool   `json:"auto_deploy"` // auto switch tunnels using this config
}

type pushConfigResult struct {
	Saved    []string `json:"saved"`
	Skipped  []string `json:"skipped"`  // same content hash
	Deployed []string `json:"deployed"` // tunnels that were switched
	Errors   []string `json:"errors"`
}

func (s *Server) handlePushConfigs(w http.ResponseWriter, r *http.Request) {
	// 1. Parse JSON body
	// 2. For each config:
	//    a. Base64 decode content
	//    b. Validate .ovpn extension
	//    c. Call store.SaveWithMeta(filename, content)
	//    d. If isNew=false (same hash), add to skipped
	//    e. If auto_deploy, find tunnels using this config name
	//    f. For matching tunnels, call tunnelMgr.Restart(name)
	// 3. Return pushConfigResult
}
```

**Step 3: Register route**

In `server.go`, add to `Router()`:
```go
mux.HandleFunc("POST /api/configs/push", s.handlePushConfigs)
```

**Step 4: Run and verify**

```bash
cd backend && go build ./cmd/vpn-agent/
```
Expected: compiles without errors

**Step 5: Commit**

```bash
git add backend/cmd/vpn-agent/
git commit -m "feat(vpn-agent): add config push endpoint with metadata tracking"
```

---

## Task 2: Backend — Config Push Proxy

**Files:**
- Modify: `backend/internal/service/vpn_agent_service.go` — add `PushConfigs` method
- Modify: `backend/internal/handler/admin/vpn_handler.go` — add `PushConfigs` handler
- Modify: `backend/internal/server/routes/admin.go` — register route

**Step 1: Add types and service method**

In `vpn_agent_service.go`, add:

```go
type PushConfigItem struct {
	Filename   string `json:"filename"`
	Content    string `json:"content"`
	Region     string `json:"region,omitempty"`
	AutoDeploy bool   `json:"auto_deploy"`
}

type PushConfigsInput struct {
	Configs         []PushConfigItem `json:"configs"`
	ReplaceExisting bool             `json:"replace_existing"`
}

type PushConfigsResult struct {
	Saved    []string `json:"saved"`
	Skipped  []string `json:"skipped"`
	Deployed []string `json:"deployed"`
	Errors   []string `json:"errors"`
}

func (s *VpnAgentService) PushConfigs(ctx context.Context, input PushConfigsInput) (*PushConfigsResult, error) {
	// POST /api/configs/push to VPN Agent
	// Forward the request body as-is
}
```

**Step 2: Add handler**

In `vpn_handler.go`, add:

```go
func (h *VpnHandler) PushConfigs(c *gin.Context) {
	if !h.requireEnabled(c) { return }
	var input service.PushConfigsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(input.Configs) == 0 {
		response.Error(c, http.StatusBadRequest, "no configs provided")
		return
	}
	result, err := h.vpnAgentService.PushConfigs(c.Request.Context(), input)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "agent error: "+err.Error())
		return
	}
	response.Success(c, result)
}
```

**Step 3: Register route**

In `admin.go`, add inside vpn group:
```go
vpn.POST("/configs/push", h.Admin.Vpn.PushConfigs)
```

**Step 4: Build and verify**

```bash
cd backend && go build -tags embed ./cmd/server/
```

**Step 5: Commit**

```bash
git add backend/internal/
git commit -m "feat(backend): add config push proxy endpoint"
```

---

## Task 3: Frontend — Push Status in VPN View

**Files:**
- Modify: `frontend/src/api/vpn.ts` — add `pushConfigs` API function
- Modify: `frontend/src/views/admin/VpnView.vue` — add push status indicators

**Step 1: Add API function**

In `vpn.ts`:

```typescript
export interface PushConfigItem {
  filename: string
  content: string  // base64
  region?: string
  auto_deploy: boolean
}

export interface PushConfigsInput {
  configs: PushConfigItem[]
  replace_existing: boolean
}

export interface PushConfigsResult {
  saved: string[]
  skipped: string[]
  deployed: string[]
  errors: string[]
}

export function pushConfigs(input: PushConfigsInput) {
  return apiClient.post<PushConfigsResult>('/admin/vpn/configs/push', input)
}
```

**Step 2: Add config metadata display to VpnView**

In `VpnView.vue`, enhance config list items to show:
- Upload time badge
- Version number
- Stale warning indicator

Add a "Push Test" button in the configs tab toolbar for manual testing.

**Step 3: Build and verify**

```bash
cd frontend && pnpm build
```

**Step 4: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add config push API and metadata display"
```
