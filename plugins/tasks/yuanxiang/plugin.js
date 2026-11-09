// Yuanxiang's authenticated /v1/models catalog, checked 2026-11-09.
// Defaults are sent explicitly so upstream defaults cannot change the facts.
const VIDEO_MODELS = {
  "SD 2.5 · 官方直连（满血版）按秒": { seconds: 5, resolution: "720p" },
  "SD 2.0 满血版 按秒": { seconds: 5, resolution: "720p", version: "标准" },
  "SD 2.5 · 特价按次": { seconds: 5, resolution: "1080p" },
  "Sd 2.0 · 特价按次 · 漫剧短剧专线": { seconds: 5, resolution: "720p" },
  "SD 2.0 · 特价按次 · 电商专线": { seconds: 5, resolution: "720p", quality: "真人模特" },
  "Muse · 特价按次 ·【原生】": { seconds: 10, resolution: "720p" },
  "MiniMax H3 · 特价按秒": { seconds: 5, resolution: "480p" },
  "MiniMax H3 · 特价按次": { seconds: 4, resolution: "768p" },
  "Wan 3.0 ·  官方直连 · 特价按秒": { seconds: 5, resolution: "480p" },
};
const IMAGE_MODELS = {
  // The API currently accepts low only; the public docs also advertise higher
  // qualities which the authenticated API rejects before task creation.
  "GT Image 2.5": { resolution: "1k", quality: "low", version: "flare" },
};
const MODEL_MAPPING = Object.fromEntries(Object.keys(VIDEO_MODELS).concat(Object.keys(IMAGE_MODELS)).map((model) => [model + "-YX", model]));
const VIDEO_NAMES = Object.keys(MODEL_MAPPING).filter((name) => Object.prototype.hasOwnProperty.call(VIDEO_MODELS, MODEL_MAPPING[name]));
const IMAGE_NAMES = Object.keys(MODEL_MAPPING).filter((name) => Object.prototype.hasOwnProperty.call(IMAGE_MODELS, MODEL_MAPPING[name]));
const DURATION_TIERS = ["5", "10", "15", "20", "30", "40", "50", "60"];
const VIDEO_USAGE_SCHEMA = {
  requests: { type: "number", unit: "count", description: { en: "Video generation request unit price", zh: "视频生成按次单价" } },
  seconds: { type: "number", unit: "second", description: { en: "Video generation duration unit price", zh: "视频生成时长单价" } },
  tokens: { type: "number", unit: "token", optional: true, description: { en: "Video generation token unit price", zh: "视频生成 Token 单价" } },
  reference_seconds: { type: "number", unit: "second", optional: true, description: { en: "Reference video unit price", zh: "参考视频单价" } },
  resolution: { enum: ["720p"], allowCustom: true, description: { en: "Output video resolution", zh: "输出视频分辨率" } },
  // An expression can branch on resolution or duration_tier independently.
  // Duration values are exact submitted seconds; multiply requests, rather
  // than seconds, when each duration tier has a total per-video price.
  duration_tier: {
    enum: DURATION_TIERS,
    allowCustom: true,
    description: { en: "Requested video duration", zh: "请求视频时长档位" },
    enumLabels: Object.fromEntries(DURATION_TIERS.map((seconds) => [seconds, { en: seconds + "s", zh: seconds + "秒" }])),
  },
  video_input: {
    enum: ["none", "video"],
    enumLabels: { none: { en: "No reference video", zh: "无参考视频" }, video: { en: "With reference video", zh: "有参考视频" } },
    description: { en: "Reference video input", zh: "参考视频输入" },
  },
};
const IMAGE_USAGE_SCHEMA = {
  image_count: { type: "number", unit: "count", unitLabel: { en: "image", zh: "张" }, description: { en: "Image generation unit price", zh: "图片生成单价" } },
  resolution: { enum: ["1k", "2k", "4k"], allowCustom: true, description: { en: "Output image resolution", zh: "输出图片分辨率" } },
};
const CONTAINERS = ["values", "metadata", "extra", "parameters"];
const IMAGE_FIELDS = ["image", "images", "image_url", "image_urls", "imageUrls", "image_refs", "image_references", "reference_images", "input_reference", "first_image", "last_image", "first_frame_url", "last_frame_url"];
const VIDEO_FIELDS = ["video", "videos", "video_url", "video_urls", "videoUrls", "video_refs", "video_references", "reference_videos"];
const AUDIO_FIELDS = ["audio", "audios", "audio_url", "audio_urls", "audioUrls", "audio_refs", "audio_references", "reference_audios"];
const QUANTITY_FIELDS = ["n", "count", "batch_size", "batch_count", "num_videos", "num_images"];

export const meta = {
  apiVersion: 1,
  key: "yuanxiang",
  name: "Yuanxiang AI",
  icon: "text:YX",
  description: { en: "Yuanxiang video and image generation with canvas parameter mapping and configurable reference video pricing", zh: "元象视频与图片生成，支持画布参数映射和可选参考视频计费" },
  version: "1.0.4",
  author: { name: "QuantumNous" },
  website: "https://yuanxiangai.cc/api-docs",
  baseUrl: "https://yuanxiangai.cc",
  fetchMode: "per_task",
  upstreams: ["vendor", "new_api"],
  requiredCapabilities: ["submit-no-retry@1", "media-duration@1", "json-file-upload@1"],
  models: Object.keys(MODEL_MAPPING),
  usageSchema: VIDEO_USAGE_SCHEMA,
  usageExamples: [
    { label: "720p · 10s", facts: { requests: 1, seconds: 10, tokens: estimateVideoTokens(10, "720p"), reference_seconds: 0, resolution: "720p", duration_tier: "10", video_input: "none" } },
    { label: "1080p · 20s", facts: { requests: 1, seconds: 20, tokens: estimateVideoTokens(20, "1080p"), reference_seconds: 0, resolution: "1080p", duration_tier: "20", video_input: "none" } },
  ],
  usageProfiles: [{ models: IMAGE_NAMES, schema: IMAGE_USAGE_SCHEMA }],
  protocols: [{ name: "openai_video", models: VIDEO_NAMES }, { name: "openai_image", models: IMAGE_NAMES }],
  routes: [
    { method: "POST", path: "/yuanxiang/v1/videos", type: "submit", decode: "createVideo", render: "task" },
    { method: "GET", path: "/yuanxiang/v1/videos/:task_id", type: "query", render: "task" },
    { method: "POST", path: "/yuanxiang/v1/images/generations", type: "submit", decode: "createImage", render: "task" },
    { method: "GET", path: "/yuanxiang/v1/images/:task_id", type: "query", render: "task" },
  ],
};

function modelDetails(ctx) {
  const requested = ctx.upstreamModel || ctx.model;
  const model = Object.prototype.hasOwnProperty.call(MODEL_MAPPING, requested) ? MODEL_MAPPING[requested] : requested;
  if (Object.prototype.hasOwnProperty.call(VIDEO_MODELS, model)) return { model: model, kind: "video", defaults: VIDEO_MODELS[model] };
  if (Object.prototype.hasOwnProperty.call(IMAGE_MODELS, model)) return { model: model, kind: "image", defaults: IMAGE_MODELS[model] };
  throw new Error("unsupported Yuanxiang model");
}

function secondsValue(value) {
  if (typeof value !== "number" && (typeof value !== "string" || !/^\d+$/.test(value.trim()))) throw new Error("duration must be an integer between 1 and 60");
  const seconds = Number(value);
  if (!Number.isInteger(seconds) || seconds < 1 || seconds > 60) throw new Error("duration must be between 1 and 60");
  return seconds;
}

function conditionValue(value, field) {
  if (typeof value !== "string" || !value.trim() || value.length > 64 || Array.from(value).some((character) => {
    const code = character.charCodeAt(0);
    return code < 32 || code === 127;
  })) throw new Error(field + " must be a short non-empty string");
  return value.trim();
}

// Used by both building and metering. Flatten parameter containers, validate
// conflicting aliases, and forward only the documented vendor body members.
function submission(ctx) {
  const profile = modelDetails(ctx);
  const input = ctx.requestBody;
  if (!input || typeof input !== "object" || Array.isArray(input)) throw new Error("request body must be an object");
  const scopes = [input];
  const values = {};
  const options = {};
  const body = { model: profile.model, wait: false };
  const references = { images: [], videos: [], audios: [] };
  for (let i = 0; i < scopes.length; i++) {
    const scope = scopes[i];
    for (const container of CONTAINERS) {
      if (scope[container] === undefined) continue;
      if (!scope[container] || typeof scope[container] !== "object" || Array.isArray(scope[container])) throw new Error(container + " must be an object");
      scopes.push(scope[container]);
      if (scopes.length > 64) throw new Error("too many parameter containers");
    }
    if (i > 0 && scope.model !== undefined && scope.model !== ctx.model && scope.model !== ctx.upstreamModel && scope.model !== profile.model)
      throw new Error("nested model must match the channel model");
    for (const field of QUANTITY_FIELDS) {
      if (scope[field] !== undefined && scope[field] !== 1 && scope[field] !== "1") throw new Error("Yuanxiang requests must generate exactly one result");
    }
    for (const field of ["max_tokens", "max_output_tokens", "max_completion_tokens"]) {
      if (scope[field] !== undefined) throw new Error(field + " is not supported by this plugin");
    }
    if (scope.options !== undefined) {
      if (!scope.options || typeof scope.options !== "object" || Array.isArray(scope.options)) throw new Error("options must be an object");
      for (const field of Object.keys(scope.options)) {
        if (CONTAINERS.concat(QUANTITY_FIELDS, ["model", "seconds", "duration", "resolution", "resolution_name", "size", "max_tokens", "max_output_tokens", "max_completion_tokens"]).includes(field))
          throw new Error("unsupported options field: " + field);
        if (Object.prototype.hasOwnProperty.call(options, field) && JSON.stringify(options[field]) !== JSON.stringify(scope.options[field])) throw new Error("conflicting options." + field);
        Object.defineProperty(options, field, { value: scope.options[field], enumerable: true, configurable: true, writable: true });
      }
    }
    const sizeIsRatio = typeof scope.size === "string" && (scope.size.includes(":") || ["adaptive", "auto"].includes(scope.size));
    for (const field of ["prompt", "seconds", "duration", "resolution", "resolution_name", "size", "aspect_ratio", "ratio", "quality", "version", "channel"]) {
      if (scope[field] === undefined) continue;
      let key = field, value = scope[field];
      if (["seconds", "duration"].includes(field)) {
        key = "duration";
        value = secondsValue(value);
      } else if (["resolution", "resolution_name"].includes(field) || (field === "size" && !sizeIsRatio)) {
        key = "resolution";
        value = conditionValue(value, field).toLowerCase();
        if (!/^(?:[1-9]\d{1,3}p|(?:0\.5|[1-8])k)$/.test(value)) throw new Error("resolution must be a label such as 720p, 1080p, 1k or 4k");
      } else if (["ratio", "aspect_ratio", "size"].includes(field)) {
        key = "ratio";
        value = conditionValue(value, field);
        if (!["auto", "adaptive"].includes(value) && (!/^\d{1,2}:\d{1,2}$/.test(value) || value.split(":").some((part) => Number(part) === 0))) throw new Error("ratio must be a positive width:height ratio");
      } else if (field === "prompt") {
        if (typeof value !== "string" || !value.trim()) throw new Error("prompt is required");
      } else value = conditionValue(value, field);
      if (Object.prototype.hasOwnProperty.call(values, key) && values[key] !== value) throw new Error("conflicting " + key + " parameters");
      values[key] = value;
    }
    for (const field of new Set(["image_ids", "image_b64s", "video_b64s", "audio_b64s"].concat(IMAGE_FIELDS, VIDEO_FIELDS, AUDIO_FIELDS))) {
      if (scope[field] === undefined) continue;
      const kind = field.startsWith("video") || VIDEO_FIELDS.includes(field) ? "videos" : field.startsWith("audio") || AUDIO_FIELDS.includes(field) ? "audios" : "images";
      for (const item of Array.isArray(scope[field]) ? scope[field] : [scope[field]]) {
        references[kind].push({ value: item, field: field, scope: i });
      }
    }
  }
  if (values.prompt === undefined) throw new Error("prompt is required");
  body.prompt = values.prompt;
  if (values.ratio !== undefined) body.ratio = values.ratio;
  if (values.channel !== undefined) body.channel = values.channel;
  const resolution = values.resolution || profile.defaults.resolution.toLowerCase();
  body.resolution = resolution;
  for (const field of ["quality", "version"]) {
    if (options[field] !== undefined) {
      const value = conditionValue(options[field], field);
      if (values[field] !== undefined && values[field] !== value) throw new Error("conflicting " + field + " parameters");
      values[field] = value;
    }
    if (values[field] === undefined && profile.defaults[field] !== undefined) values[field] = profile.defaults[field];
  }
  if (profile.kind === "video") {
    body.duration = values.duration === undefined ? profile.defaults.seconds : values.duration;
    if (values.quality !== undefined) options.quality = values.quality;
    // Muse continuation produces the fixed 20-second tier regardless of the
    // requested text-generation tier. Normalize before reservation and submit.
    if (profile.model === "Muse · 特价按次 ·【原生】" && references.videos.length) body.duration = 20;
  } else {
    if (values.duration !== undefined) throw new Error("image requests do not accept video duration");
    if (values.quality !== undefined && values.quality !== "default") body.quality = values.quality;
    delete options.quality;
  }
  if (values.version !== undefined) options.version = values.version;
  for (const file of ctx.files || []) {
    const field = file.field.replace(/\[\]$/, "");
    const kind = VIDEO_FIELDS.includes(field) || field === "video_b64s" || field === "input_reference" && String(file.mimeType || "").startsWith("video/") ? "videos"
      : AUDIO_FIELDS.includes(field) || field === "audio_b64s" || field === "input_reference" && String(file.mimeType || "").startsWith("audio/") ? "audios"
      : IMAGE_FIELDS.includes(field) || field === "image_b64s" ? "images" : "";
    if (!kind) throw new Error("unsupported reference file field: " + file.field);
    if (!Number.isSafeInteger(file.size) || file.size <= 0) throw new Error("reference files must not be empty");
    references[kind].push({ fileRef: file.ref });
  }
  for (const kind of ["images", "videos", "audios"]) {
    if (references[kind].length > (kind === "images" ? 30 : 10)) throw new Error("too many reference " + kind);
    if (profile.kind === "image" && kind !== "images" && references[kind].length) throw new Error("image models accept only reference images");
    for (const ref of references[kind]) {
      if (ref.fileRef) {
        if (kind !== "images") {
          const field = kind === "videos" ? "video_b64s" : "audio_b64s";
          (body[field] || (body[field] = [])).push({ __fileRef: ref.fileRef, encoding: "base64" });
        }
        continue;
      }
      let value = ref.value;
      if (value && typeof value === "object" && !Array.isArray(value)) value = value.url || value.image_url || value.video_url || value.audio_url;
      if (value && typeof value === "object") value = value.url;
      if (typeof value !== "string" || !value.trim()) throw new Error("invalid reference " + kind);
      value = value.trim();
      const url = /^https?:\/\/[^\s]+$/.test(value);
      if (kind === "images" && ref.field !== "image_ids" && !url) throw new Error("reference images require public image_urls or image_ids from Yuanxiang /v1/files");
      const field = ref.field === "image_ids" ? "image_ids" : kind === "images" ? url ? "image_urls" : "image_b64s" : kind === "videos" ? "video_b64s" : "audio_b64s";
      if (url && kind !== "images") throw new Error("reference videos and audio require base64 data or uploaded files");
      if (value.startsWith("data:")) {
        const comma = value.indexOf(",");
        if (comma < 0 || !value.slice(0, comma).endsWith(";base64")) throw new Error("media data URLs must use base64");
        value = value.slice(comma + 1);
      }
      if (value.length > 89478488) throw new Error("reference media must not exceed 64 MiB");
      ref.value = value;
      (body[field] || (body[field] = [])).push(value);
    }
  }
  if (profile.model === "Muse · 特价按次 ·【原生】" && references.videos.length) body.duration = 20;
  if (Object.keys(options).length) body.options = options;
  return { body: body, profile: profile, references: references };
}

function apiRoot(ctx) {
  return ctx.baseUrl.replace(/\/$/, "") + (ctx.upstream && ctx.upstream.kind === "new_api" ? "/yuanxiang" : "");
}

export function buildSubmitRequest(ctx) {
  const request = submission(ctx);
  const headers = { Authorization: "Bearer " + ctx.apiKey, "Content-Type": "application/json" };
  const gateway = ctx.upstream && ctx.upstream.kind === "new_api";
  if (gateway) request.body.model = request.profile.model + "-YX";
  for (const reference of request.references.images) {
    if (!reference.fileRef || gateway) continue;
    const uploaded = utils.uploadFile({ url: ctx.baseUrl.replace(/\/$/, "") + "/v1/files", headers: { Authorization: "Bearer " + ctx.apiKey }, fileRef: reference.fileRef, jsonField: "data_b64" });
    const id = uploaded.body && uploaded.body.file_id;
    if (uploaded.status < 200 || uploaded.status >= 300 || typeof id !== "string" || !id.trim()) throw new Error("Yuanxiang reference image upload failed");
    (request.body.image_ids || (request.body.image_ids = [])).push(id);
  }
  // Scope idempotency to this gateway task, rather than a client-provided id
  // shared by other gateway users of the same upstream API key.
  request.body.external_id = ctx.publicTaskId || utils.uuid();
  const descriptor = { url: apiRoot(ctx) + (request.profile.kind === "image" ? "/v1/images/generations" : "/v1/videos"), method: "POST", headers: headers, body: request.body, action: request.profile.kind === "image" ? "image_generation" : "video_generation" };
  if (gateway && (ctx.files || []).length) {
    // Another gateway uploads its own request files to the vendor. Preserve
    // those files as multipart rather than calling its unrelated /v1/files.
    for (const field of ["video_b64s", "audio_b64s"]) {
      if (request.body[field]) request.body[field] = request.body[field].filter((value) => typeof value === "string");
      if (request.body[field] && !request.body[field].length) delete request.body[field];
    }
    const parts = Object.keys(request.body).map((name) => ({ name: name, value: typeof request.body[name] === "object" ? JSON.stringify(request.body[name]) : String(request.body[name]) }));
    for (const file of ctx.files) parts.push({ name: file.field, fileRef: file.ref, filename: file.filename });
    delete descriptor.body;
    delete headers["Content-Type"];
    descriptor.bodyType = "multipart";
    descriptor.parts = parts;
  }
  return descriptor;
}

// Read only bounded ISO BMFF headers in a Base64 reference, without decoding or
// retaining the whole video. mdhd + hdlr select the video track; movie/audio
// duration and user-supplied duration metadata cannot establish billed seconds.
class InlineVideo {
  constructor(value) {
    if (!/^[A-Za-z0-9+/]+={0,2}$/.test(value) || value.length % 4 === 1) throw new Error("reference video must contain valid MP4 base64 data");
    this.data = value;
    this.length = Math.floor(value.replace(/=+$/, "").length * 3 / 4);
  }
  uint(offset, length) {
    if (offset < 0 || offset + length > this.length) throw new Error("truncated reference video header");
    const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let result = 0;
    for (let i = 0; i < length; i++) {
      const index = offset + i;
      const group = Math.floor(index / 3) * 4;
      const position = index % 3;
      const a = alphabet.indexOf(this.data[group + position]);
      const b = alphabet.indexOf(this.data[group + position + 1]);
      result = result * 256 + ((a << (2 + position * 2) | b >> (4 - position * 2)) & 255);
    }
    if (!Number.isSafeInteger(result)) throw new Error("reference video header exceeds safe numeric limits");
    return result;
  }
  type(offset) {
    return String.fromCharCode(this.uint(offset, 1), this.uint(offset + 1, 1), this.uint(offset + 2, 1), this.uint(offset + 3, 1));
  }
  seconds() {
    const pending = [{ start: 0, end: this.length, track: null }];
    const tracks = [];
    let boxes = 0;
    for (let i = 0; i < pending.length; i++) {
      const container = pending[i];
      for (let offset = container.start; offset < container.end;) {
        if (++boxes > 512 || offset + 8 > container.end) throw new Error("invalid reference video box layout");
        let size = this.uint(offset, 4), header = 8;
        const type = this.type(offset + 4);
        if (size === 1) { size = this.uint(offset + 8, 8); header = 16; }
        if (size === 0) size = container.end - offset;
        if (size < header || offset + size > container.end) throw new Error("invalid reference video box size");
        const start = offset + header, end = offset + size;
        if (type === "moov" || type === "trak" || type === "mdia") {
          let track = container.track;
          if (type === "trak") { track = {}; tracks.push(track); }
          pending.push({ start: start, end: end, track: track });
        } else if (container.track && type === "hdlr") {
          if (start + 12 > end) throw new Error("invalid reference video handler");
          container.track.handler = this.type(start + 8);
        } else if (container.track && type === "mdhd") {
          const version = this.uint(start, 1);
          if (version !== 0 && version !== 1) throw new Error("unsupported reference video duration header");
          const timescaleOffset = start + (version === 1 ? 20 : 12);
          const durationLength = version === 1 ? 8 : 4;
          if (timescaleOffset + 4 + durationLength > end) throw new Error("truncated reference video duration header");
          const scale = this.uint(timescaleOffset, 4);
          const duration = this.uint(timescaleOffset + 4, durationLength);
          container.track.seconds = scale > 0 ? duration / scale : 0;
        }
        offset = end;
      }
    }
    let seconds = 0;
    for (const track of tracks) if (track.handler === "vide") seconds = Math.max(seconds, track.seconds || 0);
    return seconds;
  }
}

// Match the existing video-token estimate used by the Doubao plugin: output
// seconds × pixels × 24 fps / 1024. This is a fallback estimate, not a vendor
// usage claim. Use the tier's maximum 16:9 canvas regardless of aspect ratio.
// Submission bounds duration to 1..60 and resolution to at most 9999p / 8k.
function estimateVideoTokens(seconds, resolution) {
  let width, height;
  if (resolution.endsWith("p")) {
    height = Number(resolution.slice(0, -1));
    width = Math.ceil(height * 16 / 9);
  } else if (resolution === "4k" || resolution === "8k") {
    height = Number(resolution.slice(0, -1)) * 540;
    width = height * 16 / 9;
  } else {
    width = Number(resolution.slice(0, -1)) * 1024;
    height = Math.ceil(width * 9 / 16);
  }
  return seconds * width * height * 24 / 1024;
}

export function extractUsage(ctx) {
  const request = submission(ctx);
  if (ctx.usagePurpose === "billing_ratios") return request.profile.kind === "image" ? { image_count: 1 } : { requests: 1 };
  // Retired conditions keep their first declared values for stored expressions.
  // Vendor options still pass through in the submission body. Resolution and
  // duration remain independent facts so each model's expression can select
  // one or more conditions without changing the client request.
  const facts = { resolution: request.body.resolution.toLowerCase(), quality: "default", version: "default", channel: "auto" };
  if (request.profile.kind === "image") return { ...facts, quality: "low", image_count: 1 };
  let referenceSeconds = 0;
  if ((ctx.usageKeys || []).includes("reference_seconds")) {
    for (const reference of request.references.videos) {
      const seconds = reference.fileRef ? utils.mediaDuration({ fileRef: reference.fileRef }) : new InlineVideo(reference.value).seconds();
      if (typeof seconds !== "number" || !Number.isFinite(seconds) || seconds <= 0 || seconds > 3600) throw new Error("reference video has an invalid measured duration");
      referenceSeconds += seconds;
      if (referenceSeconds > 3600) throw new Error("reference video duration total must not exceed 3600 seconds");
    }
  }
  return { ...facts, requests: 1, seconds: request.body.duration, tokens: estimateVideoTokens(request.body.duration, request.body.resolution), duration_tier: String(request.body.duration), video_input: request.references.videos.length ? "video" : "none", reference_seconds: referenceSeconds, has_reference_video: false };
}

function responseObject(body) {
  if (!body || typeof body !== "object" || Array.isArray(body)) throw new Error("invalid Yuanxiang task response");
  return body;
}

function imageEntries(body) {
  if (!body || !body.data) return [];
  const entries = Array.isArray(body.data) ? body.data : [body.data];
  return entries.filter((entry) => entry && typeof entry === "object" && (typeof entry.url === "string" && entry.url.trim() || typeof entry.b64_json === "string" && entry.b64_json.trim()));
}

export function parseTaskResult(ctx, response) {
  const body = responseObject(response);
  if (body.id !== undefined && body.id !== ctx.taskId) return { status: "UNKNOWN", reason: "Yuanxiang returned a different task id" };
  const statuses = { queued: "QUEUED", running: "IN_PROGRESS", succeeded: "SUCCESS", failed: "FAILURE", cancelled: "FAILURE" };
  const status = statuses[body.status];
  if (!status) return { status: "UNKNOWN", reason: "Unrecognized Yuanxiang task status" };
  if (status === "FAILURE") return { status: status, reason: typeof body.error === "string" ? body.error : body.error && body.error.message || "Yuanxiang generation failed" };
  if (status === "SUCCESS") {
    const delivered = modelDetails(ctx).kind === "image" ? imageEntries(body).length > 0 : typeof body.video_url === "string" && /^https?:\/\/[^\s]+$/.test(body.video_url);
    return delivered ? { status: "SUCCESS", progress: "100%" } : { status: "IN_PROGRESS", progress: "99%" };
  }
  const progress = Number(String(body.progress === undefined ? "" : body.progress).replace(/%$/, ""));
  return Number.isFinite(progress) && progress >= 0 ? { status: status, progress: Math.min(progress, 99) + "%" } : { status: status };
}

export function parseSubmitResponse(ctx, response) {
  const body = responseObject(response.body);
  const id = body.id;
  if (typeof id !== "string" || !id.trim()) throw new Error("Yuanxiang did not return a task id");
  const result = { taskId: id, taskData: body };
  const parsed = parseTaskResult({ ...ctx, taskId: id }, body);
  if (parsed.status === "SUCCESS" || parsed.status === "FAILURE") result.immediate = parsed;
  return result;
}

export function buildQueryRequest(ctx) {
  return { url: apiRoot(ctx) + (modelDetails(ctx).kind === "image" ? "/v1/images/" : "/v1/videos/") + encodeURIComponent(ctx.taskId), method: "GET", headers: { Authorization: "Bearer " + ctx.apiKey } };
}

export function extractUsageOnComplete(ctx, result, body) {
  if (result.status !== "SUCCESS") return {};
  if (modelDetails(ctx).kind === "video") {
    const usage = body && body.usage;
    if (!usage || typeof usage !== "object" || Array.isArray(usage)) return {};
    // Follow Doubao's video billing: prefer completion tokens, then total
    // tokens. Explicit zero is real usage; missing or malformed counters leave
    // the frozen local estimate in place.
    for (const field of ["completion_tokens", "total_tokens", "output_tokens"]) {
      const value = usage[field];
      if (typeof value !== "number" && (typeof value !== "string" || !/^\d+$/.test(value))) continue;
      const tokens = Number(value);
      if (Number.isSafeInteger(tokens) && tokens >= 0) return { tokens: tokens };
    }
    return {};
  }
  const entries = imageEntries(body);
  const count = Math.max(entries.filter((entry) => typeof entry.url === "string" && entry.url.trim()).length, entries.filter((entry) => typeof entry.b64_json === "string" && entry.b64_json.trim()).length);
  return count > 0 && count <= 128 ? { image_count: count } : {};
}

export function listArtifacts(task) {
  if (task.status !== "SUCCESS") return [];
  if (task.data && task.data.object === "image") return imageEntries(task.data).filter((entry) => entry.url).map((entry, i) => ({ key: "image-" + (i + 1), type: "image", mimeType: "image/png" }));
  return task.data && task.data.video_url ? [{ key: "video", type: "video", mimeType: "video/mp4" }] : [];
}

export function buildContentRequest(ctx) {
  const image = /^image-([1-9]\d*)$/.exec(ctx.artifactKey);
  if (ctx.artifactKey !== "video" && !image) throw new Error("artifact_not_found");
  // The authenticated same-origin endpoint refreshes signed storage URLs. The
  // host strips credentials if that endpoint redirects to untrusted storage.
  if (ctx.upstream && ctx.upstream.kind === "new_api") {
    const url = image ? (imageEntries(ctx.data)[Number(image[1]) - 1] || {}).url : ctx.data && ctx.data.video_url;
    if (typeof url !== "string" || !/^https?:\/\/[^\s]+$/.test(url)) throw new Error("artifact_not_found");
    return { url: url, method: "GET", credentialless: true };
  }
  return { url: ctx.baseUrl.replace(/\/$/, "") + (image ? "/v1/images/" : "/v1/videos/") + encodeURIComponent(ctx.upstreamTaskId) + "/content", method: "GET", headers: { Authorization: "Bearer " + ctx.apiKey } };
}

function decodeRequest(ctx, kind) {
  if (!ctx.body || !["json", "multipart"].includes(ctx.body.kind)) throw new Error("JSON or multipart body required");
  let body;
  if (ctx.body.kind === "json") {
    body = ctx.body.value;
    if (!body || typeof body !== "object" || Array.isArray(body)) throw new Error("JSON object required");
  } else {
    body = {};
    for (const name of Object.keys(ctx.body.fields || {})) {
      const values = ctx.body.fields[name];
      const media = IMAGE_FIELDS.concat(VIDEO_FIELDS, AUDIO_FIELDS, ["image_ids", "image_b64s", "video_b64s", "audio_b64s"]).includes(name);
      if (values.length !== 1 && !media) throw new Error(name + " must be provided once");
      let value = values.length === 1 ? values[0] : values;
      if (typeof value === "string" && (media || CONTAINERS.includes(name) || name === "options") && /^[[{]/.test(value.trim())) {
        try { value = JSON.parse(value); } catch { throw new Error(name + " must contain valid JSON"); }
      }
      Object.defineProperty(body, name, { value: value, enumerable: true });
    }
  }
  const model = ctx.model || body.model;
  const knownModel = ctx.upstreamModel || model;
  // Channel aliases can be decoded before the executing channel's mapping is
  // known. The driver validates the final model before any billable work.
  if (Object.prototype.hasOwnProperty.call(MODEL_MAPPING, knownModel) || Object.prototype.hasOwnProperty.call(VIDEO_MODELS, knownModel) || Object.prototype.hasOwnProperty.call(IMAGE_MODELS, knownModel)) {
    if (modelDetails({ model: model, upstreamModel: ctx.upstreamModel }).kind !== kind) throw new Error("model does not support this media endpoint");
  }
  return { kind: "submit", model: model, requestBody: body };
}

function renderTask(ctx, task) {
  const body = { ...(task.data || {}), id: task.task_id };
  const statuses = { NOT_START: "queued", SUBMITTED: "queued", QUEUED: "queued", IN_PROGRESS: "running", SUCCESS: "succeeded", FAILURE: "failed" };
  body.status = statuses[task.status] || "queued";
  if (task.status === "FAILURE") body.error = { message: task.fail_reason || "Yuanxiang generation failed" };
  if (ctx.artifacts && ctx.artifacts.video) body.video_url = ctx.artifacts.video.url;
  if (ctx.artifacts && ctx.artifacts["image-1"]) body.data = [{ url: ctx.artifacts["image-1"].url }];
  return body;
}

export const native = {
  createVideo: function (ctx) { return decodeRequest(ctx, "video"); },
  createImage: function (ctx) { return decodeRequest(ctx, "image"); },
  task: renderTask,
};

export const protocols = {
  openai_video: {
    decodeRequest: function (ctx) { return decodeRequest(ctx, "video"); },
    render: function (ctx, task) { return task.data || {}; },
  },
  openai_image: {
    decodeRequest: function (ctx) { return decodeRequest(ctx, "image"); },
    render: function (ctx, task) { return { data: imageEntries(task.data) }; },
  },
};
