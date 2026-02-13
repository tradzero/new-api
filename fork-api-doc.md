# Fork API Documentation

> Fork: `tradzero/new-api` from upstream `QuantumNous/new-api`

This document records all custom API endpoints added by this fork, primarily for **async video/audio generation tasks** proxied through the Zhipu adaptor (and other provider-specific adaptors).

---

## Video Task Endpoints

All endpoints require `Authorization: Bearer <token>` header.

### OpenAI-Compatible Video API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/videos` | Submit video generation task |
| GET | `/v1/videos/:task_id` | Get video task status (OpenAI format) |
| GET | `/v1/videos/:task_id/content` | Proxy video file content |

### General Video Generation API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/video/generations` | Submit video generation task |
| GET | `/v1/video/generations/:task_id` | Get video generation status |
| POST | `/v1/videos/:video_id/remix` | Create remixed video |

### Kling Native API

Uses `KlingRequestConvert` middleware to transform official Kling format.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/kling/v1/videos/text2video` | Text-to-video generation |
| POST | `/kling/v1/videos/image2video` | Image-to-video generation |
| GET | `/kling/v1/videos/text2video/:task_id` | Get text2video status |
| GET | `/kling/v1/videos/image2video/:task_id` | Get image2video status |

### Jimeng Native API

Uses `JimengRequestConvert` middleware to transform official Jimeng format.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/jimeng/` | Maps to `CVSync2AsyncSubmitTask` / `CVSync2AsyncGetResult` |

---

## Audio Task Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/task/audio` | Submit async audio task (e.g. voice cloning) |
| GET | `/v1/task/audio/:task_id` | Get audio task status |

---

## Request Format

### Task Submit Request (JSON)

```jsonc
{
  // === Core fields ===
  "model": "cogvideox-3",       // Required. Model name
  "prompt": "A cat walking...", // Required (unless model is doubao-seedance with content, or kling-custom-voice)

  // === Image/Reference ===
  "image": "https://...",       // Single image URL
  "images": ["url1", "url2"],   // Multiple image URLs
  "image_url": "https://...",   // Direct passthrough (can be string or array)
  "input_reference": "url",     // Sora-specific reference image

  // === Duration & Size ===
  "duration": 5,                // Duration in seconds (integer)
  "seconds": "5",               // Duration as string (legacy)
  "size": "720x1280",           // Output dimensions

  // === Video Generation Parameters ===
  "mode": "std",                // "std" or "pro" (affects billing for Kling models)
  "quality": "quality",         // Video quality setting
  "fps": 30,                    // Frames per second
  "aspect_ratio": "16:9",       // Aspect ratio
  "resolution": "768P",         // Resolution (e.g. "512P", "768P", "1080P")
  "negative_prompt": "...",     // Content to exclude
  "person_generation": "...",   // Person generation control
  "sample_count": 1,            // Number of samples to generate
  "seed": 12345,                // Random seed for reproducibility
  "resize_mode": "...",         // Image resize mode
  "compression_quality": "...", // Compression quality
  "first_frame_image": "url",   // First frame reference image
  "last_frame_image": "url",    // Last frame reference image
  "prompt_optimizer": true,     // Enable prompt optimization
  "fast_pretreatment": true,    // Enable fast preprocessing

  // === Audio/Sound Parameters ===
  "with_audio": true,           // Include audio in video (Veo models)
  "generate_audio": true,       // Generate audio (Seedance models)
  "sound": "on",                // Sound track toggle (Kling V2.6)
  "voice_list": [...],          // Voice list (Kling V2.6)
  "voice_name": "ssw",          // Voice name (kling-custom-voice)
  "voice_url": "https://...",   // Voice audio URL (kling-custom-voice)
  "keep_original_sound": "...", // Preserve original audio (motion control)

  // === Video/Image/Element Lists ===
  "video_list": [...],          // Reference video list (kling-video-o1)
  "image_list": [...],          // Image list for compositing
  "element_list": [...],        // Element list for compositing
  "video_url": "https://...",   // Source video URL (motion control)
  "character_orientation": "...", // Character pose (motion control)

  // === Service Parameters ===
  "service_tier": "default",    // "default" (online) or "flex" (offline) — Seedance
  "request_id": "uuid",         // Idempotency key
  "content": [...],             // Content array (Seedance format: text + image objects)

  // === Metadata ===
  "metadata": {                 // Additional key-value pairs, used as fallback
    "watermark_enabled": false,
    "user_id": "..."
  }
}
```

### Prompt Validation Bypass

Prompt is **not required** for:
- `doubao-seedance*` models when `content` field is provided
- `kling-custom-voice*` models (audio task, only needs `voice_name` + `voice_url`)

---

## Response Format

### Submit Response (OpenAI Video format)

```json
{
  "id": "task-abc123",
  "task_id": "task-abc123",
  "object": "video",
  "model": "cogvideox-3",
  "status": "queued",
  "progress": 0,
  "created_at": 1700000000,
  "metadata": {}
}
```

### Fetch Response (OpenAI Video format)

```json
{
  "id": "task-abc123",
  "object": "video",
  "model": "cogvideox-3",
  "status": "completed",
  "progress": 100,
  "created_at": 1700000000,
  "completed_at": 1700000060,
  "metadata": {
    "url": "https://...video.mp4",
    "cover_image_url": "https://...cover.jpg"
  }
}
```

**Status values:** `queued`, `in_progress`, `completed`, `failed`

### Fetch Response (Generic Task format)

For non-`/v1/videos/` paths:

```json
{
  "code": "success",
  "data": {
    "task_id": "task-abc123",
    "action": "generate",
    "status": "SUCCESS",
    "progress": "100%",
    "fail_reason": "",
    "submit_time": 1700000000,
    "start_time": 1700000001,
    "finish_time": 1700000060,
    "data": { /* raw upstream response */ }
  }
}
```

---

## Supported Models & Pricing

All models are routed through the Zhipu adaptor (`ChannelTypeZhipu_v4`). Pricing is calculated as:

```
quota = ModelPrice * GroupRatio * product(OtherRatios) * QuotaPerUnit
```

| Model Prefix | Pricing Type | OtherRatios Factors |
|---|---|---|
| `cogvideox*` | Per-second | `seconds` |
| `sora-2` | Per-second + size | `seconds`, `size` (1.67x for 1792x1024) |
| `sora-2-pro` | Per-second + size | `seconds`, `size` (1.67x for 1792x1024) |
| `veo-3.x-generate*` | Per-second + audio | `seconds`, `audio` (2.0x with audio), `sample_count` |
| `veo-3.x-fast-generate*` | Per-second + audio | `seconds`, `audio` (1.5x with audio), `sample_count` |
| `kling-v1-6`, `kling-multi-v1-6`, `kling-v2-1` | Per-5s + mode | `seconds/5`, `mode` (1.75x for pro) |
| `kling-v2-master`, `kling-v2-1-master` | Per-5s | `seconds/5` |
| `kling-v2-5-turbo` | Per-5s + mode | `seconds/5`, `mode` (5/3x for pro) |
| `kling-video-o1` | Mode + video ref | `mode` (4/3x for pro), `video_ref` (1.5x with video_list) |
| `kling-v2-6` | Duration + sound + voice | `duration` (2x for 10s), `sound` (2x on), `voice` (1.2x with voice_list) |
| `kling-v2-6-motion-control` | Per-second + mode | `seconds`, `mode` (1.6x for pro) |
| `kling-custom-voice` | Flat per-use | _(none)_ |
| `doubao-seedance*` | Token-based | `service_tier` (2x for online), `audio` (2x with audio) |
| `minimax-hailuo-2.3-Fast` | Resolution:duration | `price` (ratio from table) |
| `minimax-hailuo-2.3` | Resolution:duration | `price` (ratio from table) |
| `minimax-hailuo-02` | Resolution:duration | `price` (ratio from table) |

**ModelPrice** is configured per-model in the system's model price settings (not hardcoded).

---

## Upstream Endpoints (Zhipu Adaptor)

| Task Type | Upstream Endpoint |
|-----------|-------------------|
| Video submit | `POST {baseURL}/api/paas/v4/videos/generations` |
| Audio submit | `POST {baseURL}/api/paas/v4/async/audios/generations` |
| Task fetch | `GET {baseURL}/api/paas/v4/async-result/{task_id}` |
