// Yuanxiang model contracts verified against authenticated /v1/models and /v1/pricing on 2026-11-25.
// Defaults are sent explicitly so upstream defaults cannot change the facts.
const SD20_PER_REQUEST = "seedance-2-0-per-request";
const SD25_PER_REQUEST = "seedance-2-5-per-request";
const SD20_MANJU = "Sd 2.0 · 特价按次 · 漫剧短剧专线";
const SD20_ECOMMERCE = "SD 2.0 · 特价按次 · 电商专线";
const SD20_FULL = "SD2.0 · 即梦满血版";
const SD25_SPECIAL = "SD 2.5 · 特价按次";
const SHARED_MODELS = {
  [SD20_PER_REQUEST]: SD20_MANJU,
  [SD25_PER_REQUEST]: SD25_SPECIAL,
};
// Historical image tasks use a different query endpoint and completion facts.
const HISTORICAL_IMAGE_MODELS = new Set([
  "GT Image 2.5", "GT Image 2.5 · 特价按张", "GT Image 2.6 Flare", "GT Image 2 · OpenAI", "Banana 2",
]);
const VIDEO_NAMES = Object.keys(SHARED_MODELS);
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
const CONTAINERS = ["values", "metadata", "extra", "parameters"];
const IMAGE_FIELDS = ["image", "images", "image_url", "image_urls", "imageUrls", "image_refs", "image_references", "reference_images", "input_reference", "first_image", "last_image", "first_frame_url", "last_frame_url"];
const VIDEO_FIELDS = ["video", "videos", "video_url", "video_urls", "videoUrls", "video_refs", "video_references", "reference_videos"];
const AUDIO_FIELDS = ["audio", "audios", "audio_url", "audio_urls", "audioUrls", "audio_refs", "audio_references", "reference_audios"];
const QUANTITY_FIELDS = ["n", "count", "batch_size", "batch_count", "num_videos", "num_images"];
const SHARED_FIELDS = new Set([
  "model", "prompt", "seconds", "duration", "resolution", "resolution_name", "size", "aspect_ratio", "ratio", "quality", "options",
  "image_ids", "image_b64s", "video_b64s", "audio_b64s",
  ...CONTAINERS, ...IMAGE_FIELDS, ...VIDEO_FIELDS, ...AUDIO_FIELDS, ...QUANTITY_FIELDS,
]);

export const meta = {
  apiVersion: 1,
  key: "yuanxiang",
  name: "Yuanxiang AI",
  icon: "text:YX",
  description: { en: "Yuanxiang Seedance 2.0 and 2.5 video generation with per-request pricing", zh: "元象 Seedance 2.0 和 2.5 视频生成，按次计费" },
  version: "1.2.0",
  author: { name: "QuantumNous" },
  website: "https://yuanxiangai.cc/api-docs",
  baseUrl: "https://yuanxiangai.cc",
  fetchMode: "per_task",
  upstreams: ["vendor", "new_api"],
  requiredCapabilities: ["submit-no-retry@1", "media-duration@1", "json-file-upload@1"],
  models: VIDEO_NAMES,
  // Customer prices are in CNY for the site's 1:1 billing setup.
  modelDefaults: {
    [SD20_PER_REQUEST]: {
      displayName: "Seedance 2.0 · 按次",
      billingExpr: 'tier("720p", u("requests") * 2.9)',
    },
    [SD25_PER_REQUEST]: {
      displayName: "Seedance 2.5 · 按次",
      billingExpr: 'tier("720p", u("requests") * 6.0)',
    },
  },
  usageSchema: {
    ...VIDEO_USAGE_SCHEMA,
    image_count: { type: "number", unit: "count", optional: true, unitLabel: { en: "image", zh: "张" }, description: { en: "Image generation unit price", zh: "图片生成单价" } },
  },
  usageExamples: [
    { label: "720p · 10s", facts: { requests: 1, seconds: 10, tokens: estimateVideoTokens(10, "720p"), reference_seconds: 0, resolution: "720p", duration_tier: "10", video_input: "none", image_count: 0 } },
    { label: "1080p · 20s", facts: { requests: 1, seconds: 20, tokens: estimateVideoTokens(20, "1080p"), reference_seconds: 0, resolution: "1080p", duration_tier: "20", video_input: "none", image_count: 0 } },
  ],
  usageProfiles: [{
    models: VIDEO_NAMES,
    schema: {
      ...VIDEO_USAGE_SCHEMA,
      resolution: { ...VIDEO_USAGE_SCHEMA.resolution, allowCustom: false },
    },
    examples: [
      { label: "720p · 10s", facts: { requests: 1, seconds: 10, tokens: estimateVideoTokens(10, "720p"), reference_seconds: 0, resolution: "720p", duration_tier: "10", video_input: "none" } },
    ],
  }],
  protocols: [{ name: "openai_video", models: VIDEO_NAMES }],
  routes: [
    { method: "POST", path: "/yuanxiang/v1/videos", type: "submit", decode: "createVideo", render: "task", models: VIDEO_NAMES },
    { method: "GET", path: "/yuanxiang/v1/videos/:task_id", type: "query", render: "task" },
    { method: "GET", path: "/yuanxiang/v1/images/:task_id", type: "query", render: "task" },
  ],
};

function isHistoricalImage(ctx) {
  const model = ctx.upstreamModel || ctx.model || "";
  return HISTORICAL_IMAGE_MODELS.has(model.replace(/-YX$/, ""));
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
  const model = ctx.upstreamModel || ctx.model;
  if (!Object.hasOwn(SHARED_MODELS, model)) throw new Error("unsupported Yuanxiang model for new tasks");
  const input = ctx.requestBody;
  if (!input || typeof input !== "object" || Array.isArray(input)) throw new Error("request body must be an object");
  const scopes = [input];
  const values = {};
  const options = {};
  const body = { model: SHARED_MODELS[model], wait: false };
  const references = { images: [], videos: [], audios: [] };
  for (let i = 0; i < scopes.length; i++) {
    const scope = scopes[i];
    for (const field of Object.keys(scope)) {
      if (!SHARED_FIELDS.has(field)) throw new Error(field + " is not supported by this Yuanxiang model");
    }
    for (const container of CONTAINERS) {
      if (scope[container] === undefined) continue;
      if (!scope[container] || typeof scope[container] !== "object" || Array.isArray(scope[container])) throw new Error(container + " must be an object");
      scopes.push(scope[container]);
      if (scopes.length > 64) throw new Error("too many parameter containers");
    }
    if (i > 0 && scope.model !== undefined && scope.model !== ctx.model && scope.model !== model && scope.model !== body.model)
      throw new Error("nested model must match the channel model");
    for (const field of QUANTITY_FIELDS) {
      if (scope[field] !== undefined && scope[field] !== 1 && scope[field] !== "1") throw new Error("Yuanxiang requests must generate exactly one result");
    }
    if (scope.options !== undefined) {
      if (!scope.options || typeof scope.options !== "object" || Array.isArray(scope.options)) throw new Error("options must be an object");
      for (const field of Object.keys(scope.options)) {
        if (model !== SD20_PER_REQUEST || field !== "quality") throw new Error("unsupported Yuanxiang per-request option: " + field);
        if (Object.hasOwn(options, field) && options[field] !== scope.options[field]) throw new Error("conflicting options." + field);
        options[field] = scope.options[field];
      }
    }
    const sizeIsRatio = typeof scope.size === "string" && (scope.size.includes(":") || ["adaptive", "auto"].includes(scope.size));
    for (const field of ["prompt", "seconds", "duration", "resolution", "resolution_name", "size", "aspect_ratio", "ratio", "quality"]) {
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
  body.resolution = values.resolution ?? "720p";
  body.duration = values.duration ?? 5;
  if (Object.hasOwn(options, "quality")) {
    options.quality = conditionValue(options.quality, "quality");
    if (values.quality !== undefined && values.quality !== options.quality) throw new Error("conflicting quality parameters");
  }
  const quality = options.quality ?? values.quality;
  if (quality !== undefined && (model !== SD20_PER_REQUEST || quality !== "真人模特")) throw new Error("unsupported Yuanxiang per-request option: quality");
  for (const file of ctx.files || []) {
    const field = file.field.replace(/\[\]$/, "");
    const kind = VIDEO_FIELDS.includes(field) || field === "video_b64s" || field === "input_reference" && String(file.mimeType || "").startsWith("video/") ? "videos"
      : AUDIO_FIELDS.includes(field) || field === "audio_b64s" || field === "input_reference" && String(file.mimeType || "").startsWith("audio/") ? "audios"
      : IMAGE_FIELDS.includes(field) || field === "image_b64s" ? "images" : "";
    if (!kind) throw new Error("unsupported reference file field: " + file.field);
    if (!Number.isSafeInteger(file.size) || file.size <= 0) throw new Error("reference files must not be empty");
    references[kind].push({ fileRef: file.ref });
  }
  if (body.resolution !== "720p") throw new Error("Yuanxiang per-request Seedance requires 720p");
  const ratio = body.ratio ?? "16:9";
  const wide = ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"];
  const standard = wide.slice(0, 3);
  if (model === SD25_PER_REQUEST) {
    if (!wide.includes(ratio) || body.duration < 4 || body.duration > 30 ||
        references.images.length > 30 || references.videos.length > 10 || references.audios.length > 10 ||
        Array.from(body.prompt).length > 4000)
      throw new Error("unsupported Yuanxiang Seedance 2.5 duration, aspect ratio or references");
    // Chained gateways run the same selection with their own vendor credentials.
    if (ctx.upstream?.kind !== "new_api") body.channel = body.duration >= 5 && standard.includes(ratio) ? "line:stable-longsupply-item" : "line:economy-sd25";
  } else {
    const discreteDuration = [5, 10, 15].includes(body.duration);
    const promptLength = Array.from(body.prompt).length;
    if (references.videos.length || references.images.length > 9)
      throw new Error("Yuanxiang Seedance 2.0 does not support reference videos or more than nine images");
    if (quality === "真人模特") {
      if (!discreteDuration || !standard.includes(ratio) || references.audios.length || promptLength > 24400)
        throw new Error("unsupported Yuanxiang Seedance 2.0 e-commerce duration, aspect ratio or references");
      body.model = SD20_ECOMMERCE;
      body.options = { quality };
    } else if (discreteDuration && wide.slice(0, 5).includes(ratio) && !references.audios.length && promptLength <= 8000) {
      body.model = SD20_MANJU;
    } else if (standard.includes(ratio) && body.duration >= 5 && body.duration <= 15 && references.audios.length <= 3 && promptLength <= 4000) {
      body.model = SD20_FULL;
    } else throw new Error("unsupported Yuanxiang Seedance 2.0 duration, aspect ratio or references");
  }
  for (const kind of ["images", "videos", "audios"]) {
    if (references[kind].length > (kind === "images" ? 30 : 10)) throw new Error("too many reference " + kind);
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
  return { body, model, references };
}

function apiRoot(ctx) {
  return ctx.baseUrl.replace(/\/$/, "") + (ctx.upstream && ctx.upstream.kind === "new_api" ? "/yuanxiang" : "");
}

export function buildSubmitRequest(ctx) {
  const request = submission(ctx);
  const headers = { Authorization: "Bearer " + ctx.apiKey, "Content-Type": "application/json" };
  const gateway = ctx.upstream && ctx.upstream.kind === "new_api";
  if (gateway) request.body.model = request.model;
  for (const reference of request.references.images) {
    if (!reference.fileRef || gateway) continue;
    const uploaded = utils.uploadFile({ url: ctx.baseUrl.replace(/\/$/, "") + "/v1/files", headers: { Authorization: "Bearer " + ctx.apiKey }, fileRef: reference.fileRef, jsonField: "data_b64" });
    const id = uploaded.body && uploaded.body.file_id;
    if (uploaded.status < 200 || uploaded.status >= 300 || typeof id !== "string" || !id.trim()) throw new Error("Yuanxiang reference image upload failed");
    (request.body.image_ids || (request.body.image_ids = [])).push(id);
  }
  // Scope idempotency to this gateway task, rather than a client-provided id
  // shared by other gateway users of the same upstream API key.
  if (gateway) delete request.body.wait;
  else request.body.external_id = ctx.publicTaskId || utils.uuid();
  const descriptor = { url: apiRoot(ctx) + "/v1/videos", method: "POST", headers, body: request.body, action: "video_generation" };
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
  if (ctx.usagePurpose === "billing_ratios") return { requests: 1 };
  // Retired conditions keep their first declared values for stored expressions.
  // Vendor options still pass through in the submission body. Resolution and
  // duration remain independent facts so each model's expression can select
  // one or more conditions without changing the client request.
  const facts = { resolution: request.body.resolution.toLowerCase(), quality: "default", version: "default", channel: "auto" };
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
    const delivered = isHistoricalImage(ctx) ? imageEntries(body).length > 0 : typeof body.video_url === "string" && /^https?:\/\/[^\s]+$/.test(body.video_url);
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
  return { url: apiRoot(ctx) + (isHistoricalImage(ctx) ? "/v1/images/" : "/v1/videos/") + encodeURIComponent(ctx.taskId), method: "GET", headers: { Authorization: "Bearer " + ctx.apiKey } };
}

export function extractUsageOnComplete(ctx, result, body) {
  if (result.status !== "SUCCESS") return {};
  if (!isHistoricalImage(ctx)) {
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
  if (task.data && task.data.object === "image") {
    const entries = imageEntries(task.data);
    return entries.filter((entry) => entry.url).map((entry, i) => ({ key: "image-" + (i + 1), type: "image", mimeType: "image/png" })).concat(
      entries.filter((entry) => !entry.url && entry.b64_json).map((entry, i) => ({ key: "image-b64-" + (i + 1), type: "image", mimeType: "image/png" })),
    );
  }
  return task.data && task.data.video_url ? [{ key: "video", type: "video", mimeType: "video/mp4" }] : [];
}

export function buildContentRequest(ctx) {
  const image = /^image-(b64-)?([1-9]\d*)$/.exec(ctx.artifactKey);
  if (ctx.artifactKey !== "video" && !image) throw new Error("artifact_not_found");
  // A gateway's native presenter already supplies a credentialless host URL.
  if (ctx.upstream && ctx.upstream.kind === "new_api") {
    const entry = image ? imageEntries(ctx.data).filter((item) => image[1] ? !item.url && item.b64_json : item.url)[Number(image[2]) - 1] : undefined;
    const url = image ? entry?.url || (entry?.b64_json ? "data:image/png;base64," + entry.b64_json : "") : ctx.data && ctx.data.video_url;
    if (typeof url !== "string" || !url.startsWith("data:image/png;base64,") && !/^https?:\/\/[^\s]+$/.test(url)) throw new Error("artifact_not_found");
    return { url: url, method: "GET", credentialless: true };
  }
  // The authenticated /content endpoint redirects to storage. Refresh through
  // the JSON task query so channel credentials never accompany that redirect.
  const response = utils.fetch(buildQueryRequest({ ...ctx, taskId: ctx.upstreamTaskId }));
  if (response.status < 200 || response.status >= 300) throw new Error("Yuanxiang result refresh failed (HTTP " + response.status + ")");
  const body = responseObject(response.body);
  if (body.id !== ctx.upstreamTaskId) throw new Error("Yuanxiang result refresh returned a different task id");
  if (parseTaskResult({ ...ctx, taskId: ctx.upstreamTaskId }, body).status !== "SUCCESS") throw new Error("Yuanxiang result is not available");
  const entry = image ? imageEntries(body).filter((item) => image[1] ? !item.url && item.b64_json : item.url)[Number(image[2]) - 1] : undefined;
  const url = image ? entry?.url || (entry?.b64_json ? "data:image/png;base64," + entry.b64_json : "") : body.video_url;
  if (typeof url !== "string" || !url.startsWith("data:image/png;base64,") && !/^https?:\/\/[^/?#@\s\\]+(?:[/?][^#\s\\]*)?$/.test(url)) throw new Error("Yuanxiang result URL is not available");
  return { url: url, method: "GET", credentialless: true };
}

function decodeRequest(ctx) {
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
  if (ctx.upstreamModel || Object.hasOwn(SHARED_MODELS, knownModel))
    submission({ model, upstreamModel: knownModel, requestBody: body, files: ctx.body.files });
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
  createVideo: decodeRequest,
  task: renderTask,
};

export const protocols = {
  openai_video: {
    decodeRequest,
    render: function (ctx, task) { return task.data || {}; },
  },
};
