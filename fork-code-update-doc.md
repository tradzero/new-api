# Fork Code Update Documentation

> Fork: `tradzero/new-api` (origin) from `QuantumNous/new-api` (upstream)
>
> Merge base: `ba032b72`

This document records all code modifications made in this fork relative to upstream, for reference during upstream rebasing and future maintenance.

---

## Change Summary

**Committed changes** (from merge base): 7 files, +934 / -31 lines
**Uncommitted changes** (audio task integration): 8 files, +55 / -7 lines

---

## Modified Files

### 1. `relay/channel/task/zhipu/adaptor.go` (NEW FILE, ~750 lines)

**Purpose:** Zhipu video/audio task adaptor — the core adaptor handling async video and audio generation tasks.

**What it does:**
- Defines `zhipuVideoRequest` struct with 40+ fields for all supported video/audio parameters
- Defines response structs (`zhipuVideoSubmitResponse`, `zhipuVideoFetchResponse`)
- Implements `channel.TaskAdaptor` interface (`Init`, `ValidateRequestAndSetAction`, `BuildRequestURL`, `BuildRequestBody`, `DoRequest`, `DoResponse`, `FetchTask`, `ParseTaskResult`, `ConvertToOpenAIVideo`)
- Contains the **PricingFunc registry** system: maps model name prefixes to pricing calculation functions
- Switches between video endpoint (`/api/paas/v4/videos/generations`) and audio endpoint (`/api/paas/v4/async/audios/generations`) based on `RelayMode`

**Pricing functions implemented:**

| Function | Models | Logic |
|----------|--------|-------|
| `pricingPerSecond` | cogvideox* | `seconds` |
| `pricingSoraV2` | sora-2* | `seconds` * `size` ratio |
| `makePricingVeo(audioRatio)` | veo-3.x* | `seconds` * `audio` * `sample_count` |
| `makePricingKling(proRatio)` | kling-v1-6, v2-1, v2-5-turbo | `seconds/5` * `mode` ratio |
| `pricingKlingMaster` | kling-v2-master* | `seconds/5` |
| `pricingSeedance` | doubao-seedance* | `service_tier` * `audio` (token-based) |
| `makePricingHailuo(ratioTable)` | minimax-hailuo* | lookup `resolution:duration` ratio |
| `pricingKlingO1` | kling-video-o1 | `mode` * `video_ref` |
| `pricingKlingV26` | kling-v2-6 | `duration` * `sound` * `voice` |
| `pricingKlingV26MC` | kling-v2-6-motion-control | `seconds` * `mode` |
| `pricingFlat` | kling-custom-voice | _(empty — flat per-use)_ |

**Model name matching:** `getPricingFunc()` tries exact match first, then longest prefix match, then defaults to `pricingPerSecond`.

**Key design note:** `OtherRatios` values are multiplied together with `ModelPrice * GroupRatio` in `relay/relay_task.go` to calculate the final quota. Base prices (`ModelPrice`) are configured in the system settings, not hardcoded.

---

### 2. `relay/common/relay_info.go`

**Changes:** Extended `TaskSubmitReq` struct with 30+ video/audio generation fields.

**Added fields (beyond upstream):**

```
AspectRatio, NegativePrompt, PersonGeneration, SampleCount, Seed,
ResizeMode, CompressionQuality, WithAudio, GenerateAudio, ServiceTier,
RequestID, ImageURL, FirstFrameImage, LastFrameImage, Resolution,
PromptOptimizer, FastPretreatment, Quality, FPS,
VideoList, ImageList, ElementList, Sound, VoiceList, VideoURL,
KeepOriginalSound, CharacterOrientation, VoiceName, VoiceURL
```

**Why:** These fields are parsed from user requests and passed through to upstream providers via the adaptor's `convertToRequestPayload()`.

---

### 3. `relay/common/relay_utils.go`

**Changes:**
- `ValidateBasicTaskRequest`: Added prompt validation bypass for `doubao-seedance` (with `content`) and `kling-custom-voice` models
- `ValidateMultipartDirect`: Same prompt bypass logic added

**Code pattern:**
```go
skipPrompt := (req.Content != nil && strings.HasPrefix(req.Model, "doubao-seedance")) ||
    strings.HasPrefix(req.Model, "kling-custom-voice")
```

---

### 4. `relay/constant/relay_mode.go`

**Changes:** Added 4 new relay mode constants:

```go
RelayModeVideoFetchByID     // GET /v1/videos/:task_id, /v1/video/generations/:task_id
RelayModeVideoSubmit        // POST /v1/videos, /v1/video/generations

RelayModeAudioTaskSubmit    // POST /v1/task/audio
RelayModeAudioTaskFetchByID // GET /v1/task/audio/:task_id
```

**Note:** Video relay modes are set by `Distribute()` middleware, NOT by `Path2RelayMode()`. The middleware writes to `c.Set("relay_mode", ...)`, and `genBaseRelayInfo` reads it via `c.GetInt("relay_mode")` fallback.

---

### 5. `middleware/distributor.go`

**Changes:** Added path detection blocks in `Distribute()` for video and audio task paths:

- `/v1/videos/*` and `/v1/video/generations/*` → sets `RelayModeVideoSubmit` or `RelayModeVideoFetchByID`
- `/v1/task/audio/*` → sets `RelayModeAudioTaskSubmit` or `RelayModeAudioTaskFetchByID`
- POST requests extract model from request body for channel selection
- GET requests skip channel selection (`shouldSelectChannel = false`)

---

### 6. `router/video-router.go`

**Changes:** Added all video/audio task routes.

**Route groups:**
1. `/v1/*` — OpenAI-compatible video routes + general video generation + audio task routes
2. `/kling/v1/*` — Kling native API routes (with `KlingRequestConvert` middleware)
3. `/jimeng/*` — Jimeng native API routes (with `JimengRequestConvert` middleware)

All routes use `middleware.TokenAuth()` and `middleware.Distribute()`.

---

### 7. `controller/relay.go`

**Changes:**
- `taskRelayHandler`: Added `RelayModeAudioTaskFetchByID` to the fetch case alongside `RelayModeSunoFetch`, `RelayModeSunoFetchByID`, `RelayModeVideoFetchByID`

```go
case relayconstant.RelayModeSunoFetch, ..., relayconstant.RelayModeAudioTaskFetchByID:
    err = relay.RelayTaskFetch(c, relayInfo.RelayMode)
```

---

### 8. `relay/relay_task.go`

**Changes:**
- Added `RelayModeAudioTaskFetchByID` to `fetchRespBuilders` map, reusing `videoFetchByIDRespBodyBuilder`

```go
relayconstant.RelayModeAudioTaskFetchByID: videoFetchByIDRespBodyBuilder,
```

---

### 9. `relay/channel/zhipu_4v/adaptor.go` + `image.go`

**Changes:** Enhanced Zhipu image generation support:
- Extended image request handling with more parameters
- Image response format adjustments
- Usage/billing fixes for image generation

---

### 10. `dto/openai_image.go`

**Changes:** Added new fields to image request DTO:
- `N`, `AspectRatio`, `Resolution`, `ImageList`, `ElementList`

---

### 11. `relay/relay_adaptor.go`

**Changes:** Registered new task adaptors in `GetTaskAdaptor()`:
- `ChannelTypeZhipu_v4` → `taskZhipu.TaskAdaptor{}`
- Other task adaptors (ali, kling, jimeng, vertex, vidu, doubao, sora, gemini, hailuo) were registered here as well

---

## Pricing System Architecture

```
User Request
    │
    ▼
ValidateRequestAndSetAction()
    │  Calls getPricingFunc(modelName)
    │  Sets info.PriceData.OtherRatios = pricingFn(&req)
    ▼
RelayTaskSubmit() in relay_task.go
    │  modelPrice = GetModelPrice(modelName)  // from system config
    │  groupRatio = GetGroupRatio(group)
    │  ratio = modelPrice * groupRatio
    │  for _, ra := range OtherRatios { ratio *= ra }
    │  quota = int(ratio * QuotaPerUnit)
    ▼
Pre-deduct quota → DoRequest → DoResponse → Post-consume quota
```

**Key:** `OtherRatios` is a `map[string]float64`. All values != 1.0 are multiplied into the final ratio. Each pricing function returns different keys (e.g. `"seconds"`, `"mode"`, `"audio"`, `"size"`) depending on the model's billing dimensions.

---

## Upstream Rebase Notes

### High conflict risk (modify carefully):
- `relay/common/relay_info.go` — `TaskSubmitReq` has many added fields
- `relay/common/relay_utils.go` — validation logic modified
- `relay/constant/relay_mode.go` — new constants added (iota-based, position matters)
- `middleware/distributor.go` — path handling blocks added inline

### Low conflict risk (new/isolated files):
- `relay/channel/task/zhipu/adaptor.go` — entirely new file
- `router/video-router.go` — mostly isolated, but check for route conflicts

### Tips:
1. When rebasing, resolve `relay_mode.go` first — iota values affect all relay mode consumers
2. `TaskSubmitReq` fields are additive; append new fields at the end to minimize conflicts
3. The pricing registry in `zhipu/adaptor.go` is self-contained; unlikely to conflict with upstream
4. Check `relay/relay_adaptor.go` for new upstream task adaptors that may overlap

---

## Commit History (recent, fork-specific)

```
1428de4f  v2.6 motion control
342e7429  kling v26
d2aa37de  kling-video-o1
bb498930  parameter passthrough
801351cf  fit for hailuo
0fabd4b9  fix for doubao seedance
b41372bb  seedance integration
31f38c5f  zhipu seedance apply
d53d1e80  add N, AspectRatio, Resolution, ImageList, ElementList to image request
3466d9b7  kling billing logic
a2afc1e5  first/last frame passthrough
d19ea19f  pricing calculation
cc16ddda  zhipu veo fit
92b29180  sora price
5508f4e5  image integration (debug removed)
142e2b01  zhipu video relay (initial)
```

**Uncommitted:** Audio task integration (`kling-custom-voice` via `/v1/task/audio`)
