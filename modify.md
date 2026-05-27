# Fork 差异说明

本文档记录当前 fork 相对主线版本 `upstream/main` 保留的本地差异，重点说明与 `zhipu_4v`、智谱视频/音频异步任务、Kling/Veo/Seedance/Hailuo/Sora 兼容相关的改动。

当前仓库关系：

- `origin`: `https://github.com/tradzero/new-api.git`
- `upstream`: `https://github.com/QuantumNous/new-api.git`
- 当前分支：`main`
- 对比基准：`upstream/main..main`

## Zhipu 4V 图像能力增强

相关文件：

- `relay/channel/zhipu_4v/adaptor.go`
- `relay/channel/zhipu_4v/image.go`

主要改动：

- 增强 `zhipu_4v` 图像生成请求转换，支持将 OpenAI image request 的 `extra` 字段透传到智谱请求结构。
- 新增或扩展图像参数：
  - `n`
  - `ratio`
  - `seed`
  - `aspect_ratio`
  - `resolution`
  - `image_list`
  - `element_list`
  - `sequential_image_generation`
  - `sequential_image_generation_options`
- 图像响应支持 OpenAI 风格返回：
  - `response_format=url` 时直接返回 URL
  - 其他情况返回 `b64_json`
  - 支持从上游 URL 下载图片并转 base64
  - 支持上游把 base64 放在 `url` 字段里的情况
- 响应中补充 `usage` 字段，并将 `input/output tokens` 归一到 `prompt/completion tokens`。
- 固定价格模型下，如果实际返回图片数量少于请求的 `n`，会按实际图片数调整 `ModelPrice`，避免多扣。

## Zhipu 4V 音频、元素、识别人脸接口

相关文件：

- `relay/channel/zhipu_4v/adaptor.go`
- `dto/element.go`
- `dto/identify_face.go`
- `relay/element_handler.go`
- `relay/identify_face_handler.go`
- `router/relay-router.go`

主要改动：

- 为 `zhipu_4v` 增加音频 TTS 请求转换，目标接口为 `/api/paas/v4/audio/tts`。
- 新增 Element 请求 DTO 和 `/v1/elements` 路由，用于智谱/Kling custom elements。
- 新增 Identify Face 请求 DTO 和 `/v1/video/identify-face` 路由。
- `zhipu_4v` adaptor 中新增上游接口映射：
  - `/api/paas/v4/images/custom-elements`
  - `/api/paas/v4/videos/identify-face`
  - `/api/paas/v4/audio/tts`

已修复问题：

- 当前树里 Element/IdentifyFace 的路由、DTO、handler 已存在，但主 relay 分发链路尚未完整接入。
- 在 1.0 代码上用临时 probe 测试验证，`RelayFormatElement` 和 `RelayFormatIdentifyFace` 进入 `GenRelayInfo` 后都会返回 `invalid relay format`。
- 已补齐 `GenRelayInfo` 对 `RelayFormatElement` / `RelayFormatIdentifyFace` 的分支。
- 已补齐 `relayHandler` 对 `RelayModeElementCreate` / `RelayModeIdentifyFace` 的分支，分别转发到 `ElementHelper` 和 `IdentifyFaceHelper`。
- 已增加回归测试，覆盖 relay info 生成和 controller 分发路径，防止后续再次落回 `invalid relay format` 或误走 `TextHelper`。

## 新增 Zhipu 异步视频/音频任务适配器

相关文件：

- `relay/channel/task/zhipu/adaptor.go`
- `relay/relay_adaptor.go`
- `relay/relay_task.go`
- `relay/common/relay_info.go`
- `relay/common/relay_utils.go`
- `constant/task.go`
- `router/audio-task-router.go`
- `router/main.go`

主要改动：

- 新增 `zhipu_video` `TaskAdaptor`，绑定到 `ChannelTypeZhipu_v4`。
- 上游提交接口：
  - 视频任务：`/api/paas/v4/videos/generations`
  - 音频异步任务：`/api/paas/v4/async/audios/generations`
  - 任务查询：`/api/paas/v4/async-result/{task_id}`
- 支持模型/能力族：
  - `cogvideox` / `cogvideox-2` / `cogvideox-3`
  - `sora-2` / `sora-2-pro`
  - `veo-3.0` / `veo-3.1` generate / fast generate
  - `doubao-seedance`
  - `minimax-hailuo`
  - `kling-video-o1`
  - `kling-v2-6`
  - `kling-v2-6-motion-control`
  - `kling-custom-voice`
  - `kling-lip-sync` 相关参数
- 扩展 `TaskSubmitReq`，支持大量视频生成参数：
  - `content`
  - `image_url` / `images` / `image`
  - `first_frame_image` / `last_frame_image`
  - `aspect_ratio` / `resolution` / `duration` / `fps` / `mode`
  - `negative_prompt` / `seed` / `sample_count`
  - `generate_audio` / `with_audio` / `service_tier`
  - `video_list` / `image_list` / `element_list`
  - `voice_list` / `voice_url` / `voice_name`
  - `video_id` / `session_id` / `face_id` / `audio_id` / `audio_url`
  - lip-sync 声音插入、音量、时间范围等字段
- 增加按模型族计算价格倍率的逻辑：
  - Sora 按秒和分辨率倍率
  - Veo 按秒和音频倍率
  - Kling 按时长、模式、音频、voice/video reference 等倍率
  - Seedance 按 `service_tier`、`generate_audio` 和任务完成后的 `total_tokens` 计费
  - Hailuo 按分辨率和时长倍率
- 新增 `/v1/audio/custom` 和 `/v1/audio/tasks/:task_id` 路由，用于音频异步任务提交和查询。

## 相关提交线索

可以从以下提交追溯本地 fork 的主要改动：

- `142e2b012`：`zhipu video relay`
- `a374b5e91`：`zhipu image adaptor handle and usage fix`
- `5236654f0`：`add more field support for zhipu image`
- `36f55c648`：`zhipu image response format`
- `d53d1e808`：增加图像请求参数 `N / AspectRatio / Resolution / ImageList / ElementList`
- `31f38c5f6`、`b41372bbb`：Seedance 适配
- `7bd74ab99`：`ExecutionExpiresAfter`
- `6ead664f4`：Kling TTS
- `47bd49895`：identify-face
- `96ac12065`：kling-lip-sync relay
- `7fee583bd`：fix upstream type change

## 风险和待处理项

- 代码里仍有 `TODO: debug only` 日志：
  - `zhipu image request body`
  - `zhipu video request body`
  - `zhipu video response body`
  这些日志可能输出请求体或上游响应，生产环境需要确认是否保留。
- `/v1/elements` 和 `/v1/video/identify-face` 的主 relay 分发链路已补齐，并通过聚焦测试验证。
- `upstream/main..main` 里还看到 `controller/discord.go`、`controller/linuxdo.go`、`controller/oidc.go` 被删除，这不像 `zhipu_4v` 相关改动，建议单独列为非 zhipu 本地差异，不纳入本说明的核心范围。

## 已验证测试

- `go test ./relay/common -run 'TestGenRelayInfoSupportsElementAndIdentifyFaceFormats|TestRelayInfoGetFinalRequestRelayFormat' -v`
- `go test ./controller -run TestRelayHandlerRoutesElementAndIdentifyFaceAwayFromTextHelper -v`
