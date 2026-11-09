// Model names and default resolutions come from Yueqi's /v1/models.
// Its descriptions and request_spec disagree about reference capabilities, so
// preserve media fields and let the upstream validate each model's inputs.
const MODELS = {
  "BD Seedance2.5": { defaultResolution: "720p" },
  "kling-4.0": { defaultResolution: "720p" },
  "sd2.5新": { defaultResolution: "720p" },
  "seedance-2.0-fast-通用": { defaultResolution: "720p" },
  "seedance2.5无限并发版": { defaultResolution: "720p" },
  "vidu-q4": { defaultResolution: "720p" },
  "vidu-q4-nopeak": { defaultResolution: "720p" },
  "w3-c1": { defaultResolution: "720p" },
  "wan3.0-vedio-deal": { defaultResolution: "480p" },
  "wan3.0-video-nsfw": { defaultResolution: "480p" },
  "wan3.0-video-prime-deal": { defaultResolution: "720p" },
  "新seedance2.0-933": { defaultResolution: "720p" },
};
// Explicit public model names are video protocol bindings as well as driver
// aliases. This keeps routing independent of cross-channel alias inference,
// which can become ambiguous when another channel maps the same public name
// to a different model. Channel model_mapping still takes precedence.
const MODEL_ALIASES = {
  "seedance2.5-per": "BD Seedance2.5",
  "seedance-2.0-fast-per": "seedance-2.0-fast-通用",
  "seedance2.5": "seedance2.5无限并发版",
  "seedance2.0-9:16-per": "新seedance2.0-933",
  "vidu-q4-per": "vidu-q4",
  "wan3.0-vedio-deal-per": "wan3.0-vedio-deal",
  "wan3.0-video-prime": "wan3.0-video-prime-deal",
  "new-seedance2.5-per": "sd2.5新",
  "wan3-c1-per": "w3-c1",
};
const VIDEO_USAGE_SCHEMA = {
  requests: {
    type: "number",
    unit: "count",
    description: { en: "Video generation request unit price", zh: "视频生成按次单价" },
  },
  resolution: {
    enum: ["720p"],
    allowCustom: true,
    description: { en: "Output video resolution", zh: "输出视频分辨率" },
  },
  reference_seconds: {
    type: "number",
    unit: "second",
    optional: true,
    description: { en: "Reference video unit price", zh: "参考视频单价" },
  },
  seconds: {
    type: "number",
    unit: "second",
    description: { en: "Video generation duration unit price", zh: "视频生成时长单价" },
  },
};
const CONTAINERS = ["values", "metadata", "extra", "parameters"];
const VIDEO_FIELDS = [
  "video_url",
  "video_urls",
  "videoUrls",
  "video_refs",
  "video_references",
  "videoReferences",
  "reference_videos",
  "referenceVideos",
  "videos",
];
const MEDIA_FIELDS = [
  "image",
  "images",
  "image_url",
  "image_urls",
  "imageUrls",
  "image_refs",
  "image_references",
  "imageReferences",
  "reference_images",
  "referenceImages",
  "input_reference",
  "video_url",
  "video_urls",
  "videoUrls",
  "video_refs",
  "video_references",
  "videoReferences",
  "reference_videos",
  "referenceVideos",
  "videos",
  "audio_url",
  "audio_urls",
  "audioUrls",
  "audio_refs",
  "audio_references",
  "audioReferences",
  "reference_audios",
  "referenceAudios",
  "audios",
  "references",
  "first_image",
  "last_image",
  "first_frame_url",
  "last_frame_url",
];

export const meta = {
  apiVersion: 1,
  key: "yueqi",
  name: "Yueqi Video",
  icon: "text:YQ",
  description: {
    en: "Seedance, Wan, Kling, and Vidu video generation through Yueqi with frame and media references",
    zh: "通过 Yueqi 调用 Seedance、Wan、Kling 和 Vidu 视频模型，支持首尾帧和参考素材",
  },
  version: "1.1.2",
  author: { name: "QuantumNous" },
  website: "https://yueqi.icu/docs",
  baseUrl: "https://yueqi.icu",
  fetchMode: "per_task",
  upstreams: ["vendor", "new_api"],
  // An accepted creation has no documented idempotency key. Never retry an
  // ambiguous submission, even on another channel; poll its original id.
  requiredCapabilities: ["submit-no-retry@1", "media-duration@1"],
  models: Object.keys(MODELS).concat(Object.keys(MODEL_ALIASES)),
  // Every model reports the same usage facts. Only the administrator's expression
  // chooses which quantity is charged and its unit price.
  usageSchema: VIDEO_USAGE_SCHEMA,
  protocols: ["openai_video"],
};

function secondsValue(value) {
  if (typeof value !== "number" && (typeof value !== "string" || !/^\d+(?:\.\d+)?$/.test(value.trim())))
    throw new Error("seconds must be a decimal number between 1 and 60");
  const seconds = Number(value);
  // Yueqi's public model metadata caps output duration at 60 seconds. The
  // upstream enforces any narrower model limits. Auto duration is unsupported.
  if (!Number.isFinite(seconds) || seconds < 1 || seconds > 60) throw new Error("seconds must be between 1 and 60");
  return seconds;
}

// Submission and metering share normalization, including billing aliases in
// passthrough containers. Media objects keep their own duration and role data.
function videoSubmission(ctx) {
  const requestedModel = ctx.upstreamModel || ctx.model;
  const model = Object.prototype.hasOwnProperty.call(MODEL_ALIASES, requestedModel) ? MODEL_ALIASES[requestedModel] : requestedModel;
  if (!Object.prototype.hasOwnProperty.call(MODELS, model)) throw new Error("unsupported Yueqi video model");
  const profile = MODELS[model];
  const input = ctx.requestBody;
  if (!input || typeof input !== "object" || Array.isArray(input)) throw new Error("request body must be an object");
  const scopes = [{ ...input }];
  let seconds, resolution, ratio;
  for (let i = 0; i < scopes.length; i++) {
    const scope = scopes[i];
    for (const container of CONTAINERS) {
      if (scope[container] === undefined) continue;
      if (!scope[container] || typeof scope[container] !== "object" || Array.isArray(scope[container])) throw new Error(container + " must be an object");
      scope[container] = { ...scope[container] };
      scopes.push(scope[container]);
      if (scopes.length > 64) throw new Error("too many video parameter containers");
    }
    if (i > 0 && scope.model !== undefined && scope.model !== model) throw new Error("nested model must match the channel model");
    for (const field of ["seconds", "duration"]) {
      if (scope[field] === undefined) continue;
      const value = secondsValue(scope[field]);
      if (seconds !== undefined && seconds !== value) throw new Error("seconds conflicts with duration");
      seconds = value;
    }
    // Video clients use size for either a resolution label or an aspect
    // ratio. Rewrite only parameter containers, leaving media metadata intact.
    const sizeIsRatio = typeof scope.size === "string" && scope.size.includes(":");
    let scopeResolution, scopeRatio;
    for (const field of ["size", "resolution", "resolution_name"]) {
      if (scope[field] === undefined) continue;
      if (field === "size" && sizeIsRatio) continue;
      const value = typeof scope[field] === "string" ? scope[field].trim().toLowerCase() : scope[field];
      if (typeof value !== "string" || !/^[1-9]\d{1,3}p$/.test(value)) throw new Error("resolution must be a label such as 480p, 720p or 1080p");
      if (resolution !== undefined && resolution !== value) throw new Error("conflicting video resolution parameters");
      scopeResolution = resolution = value;
    }
    for (const field of sizeIsRatio ? ["aspect_ratio", "ratio", "size"] : ["aspect_ratio", "ratio"]) {
      if (scope[field] === undefined) continue;
      const value = typeof scope[field] === "string" ? scope[field].trim() : scope[field];
      if (typeof value !== "string" || !/^\d{1,2}:\d{1,2}$/.test(value) || value.split(":").some((part) => Number(part) === 0))
        throw new Error("aspect_ratio must be a positive width:height ratio");
      if (ratio !== undefined && ratio !== value) throw new Error("ratio conflicts with aspect_ratio");
      scopeRatio = ratio = value;
    }
    if (scopeResolution !== undefined) scope.resolution = scopeResolution;
    if (scopeRatio !== undefined) scope.aspect_ratio = scopeRatio;
    delete scope.resolution_name;
    delete scope.size;
    delete scope.ratio;
    // Canvas clients wrap image URLs in input_reference; Yueqi's reference
    // requests take an images array. Keep signed URLs byte-for-byte intact.
    const reference = scope.input_reference;
    if (reference && typeof reference === "object" && !Array.isArray(reference) && reference.image_url !== undefined) {
      const image = reference.image_url;
      const url = image && typeof image === "object" && !Array.isArray(image) ? image.url : image;
      if (typeof url !== "string" || !/^https?:\/\/[^\s]+$/.test(url)) throw new Error("input_reference.image_url must be a public image URL");
      if (scope.images !== undefined && !Array.isArray(scope.images)) throw new Error("images must be an array");
      const images = [...(scope.images || [])];
      if (!images.includes(url)) images.unshift(url);
      scope.images = images;
      delete scope.input_reference;
    }
    for (const field of ["n", "count", "batch_size", "batch_count", "num_videos"]) {
      if (scope[field] !== undefined && scope[field] !== 1 && scope[field] !== "1") throw new Error("Yueqi video requests must generate exactly one video");
    }
    for (const field of ["quality", "max_tokens", "max_output_tokens", "max_completion_tokens"]) {
      if (scope[field] !== undefined) throw new Error(field + " is not supported by this video plugin");
    }
  }
  const prompt = scopes.find((scope) => scope.prompt !== undefined)?.prompt;
  if (typeof prompt !== "string" || !prompt.trim()) throw new Error("prompt is required");
  const body = Object.assign({}, scopes[0], {
    model: model,
    prompt: prompt,
    seconds: seconds === undefined ? 5 : seconds,
    resolution: resolution || profile.defaultResolution,
  });
  if (ratio !== undefined) body.aspect_ratio = ratio;
  let action = "text_to_video";
  for (const scope of scopes) {
    if (MEDIA_FIELDS.some((field) => scope[field] !== undefined)) action = "reference_to_video";
    if (["first_image", "last_image", "first_frame_url", "last_frame_url"].some((field) => scope[field] !== undefined)) {
      action = "frames_to_video";
      break;
    }
  }
  const files = ctx.files || [];
  for (const file of files) {
    const field = file.field.replace(/\[\]$/, "");
    if (!MEDIA_FIELDS.includes(field)) throw new Error("unsupported video file field: " + file.field);
    if (!(file.size > 0)) throw new Error("video reference files must not be empty");
    if (["first_image", "last_image", "first_frame_url", "last_frame_url"].includes(field)) action = "frames_to_video";
    else if (action === "text_to_video") action = "reference_to_video";
  }
  return { body: body, action: action };
}

export function buildSubmitRequest(ctx) {
  const submission = videoSubmission(ctx);
  const headers = { Authorization: "Bearer " + ctx.apiKey };
  const url = ctx.baseUrl.replace(/\/$/, "") + "/v1/videos";
  if ((ctx.files || []).length) {
    const parts = [];
    for (const name of Object.keys(submission.body)) {
      const value = submission.body[name];
      if (value === undefined) continue;
      parts.push({ name: name, value: value !== null && typeof value === "object" ? JSON.stringify(value) : String(value) });
    }
    for (const file of ctx.files) parts.push({ name: file.field, fileRef: file.ref, filename: file.filename });
    return { url: url, method: "POST", headers: headers, bodyType: "multipart", parts: parts, action: submission.action };
  }
  headers["Content-Type"] = "application/json";
  return { url: url, method: "POST", headers: headers, body: submission.body, action: submission.action };
}

export function extractUsage(ctx) {
  const body = videoSubmission(ctx).body;
  // Legacy per-call pricing must stay per call rather than multiplying every
  // numeric fact. Task expressions select requests, seconds, or both explicitly.
  if (ctx.usagePurpose === "billing_ratios") return { requests: 1 };
  const referenceSeconds = (ctx.usageKeys || []).includes("reference_seconds") ? referenceVideoSeconds(ctx, body) : 0;
  return { requests: 1, seconds: body.seconds, resolution: body.resolution, reference_seconds: referenceSeconds };
}

function referenceVideoSeconds(ctx, body) {
  const scopes = [body];
  const references = [];
  for (let i = 0; i < scopes.length; i++) {
    const scope = scopes[i];
    for (const container of CONTAINERS) {
      if (scope[container]) scopes.push(scope[container]);
      if (scopes.length > 64) throw new Error("too many video parameter containers");
    }
    for (const field of VIDEO_FIELDS) {
      if (scope[field] === undefined) continue;
      references.push(...(Array.isArray(scope[field]) ? scope[field] : [scope[field]]));
    }
    for (const reference of Array.isArray(scope.references) ? scope.references : []) {
      if (
        reference &&
        typeof reference === "object" &&
        (String(reference.type || "").includes("video") || String(reference.mime_type || reference.mimeType || "").startsWith("video/"))
      )
        references.push(reference);
    }
  }
  const sources = [];
  for (const reference of references) {
    let url = reference;
    if (reference && typeof reference === "object") url = reference.url || reference.video_url || reference.videoUrl;
    if (url && typeof url === "object") url = url.url;
    if (typeof url !== "string" || !/^https?:\/\/[^\s]+$/.test(url)) throw new Error("reference video billing requires a public video URL or uploaded file");
    if (!sources.some((source) => source.url === url)) sources.push({ url: url });
  }
  for (const file of ctx.files || []) {
    const field = file.field.replace(/\[\]$/, "");
    if (VIDEO_FIELDS.includes(field) || (field === "input_reference" && String(file.mimeType || "").startsWith("video/"))) sources.push({ fileRef: file.ref });
  }
  if (sources.length > 16) throw new Error("reference video billing supports at most 16 videos");
  let seconds = 0;
  for (const source of sources) {
    const measured = utils.mediaDuration(source);
    if (typeof measured !== "number" || !Number.isFinite(measured) || measured <= 0 || measured > 3600)
      throw new Error("reference video has an invalid measured duration");
    seconds += measured;
    if (seconds > 3600) throw new Error("reference video duration total must not exceed 3600 seconds");
  }
  return seconds;
}

function videoObject(body) {
  if (!body || typeof body !== "object" || Array.isArray(body)) throw new Error("invalid Yueqi video response");
  if (body.success === false || (body.code !== undefined && ![0, "0", 200, "200", "success"].includes(body.code)))
    throw new Error("Yueqi video request failed");
  return body.status === undefined &&
    body.id === undefined &&
    body.task_id === undefined &&
    body.data &&
    typeof body.data === "object" &&
    !Array.isArray(body.data)
    ? body.data
    : body;
}

function resultURL(body) {
  for (const value of [body.url, body.download_url, body.video_url, body.metadata && body.metadata.url]) {
    if (typeof value === "string" && /^https?:\/\/[^\s]+$/.test(value)) return value;
  }
  return "";
}

export function parseSubmitResponse(ctx, response) {
  const body = videoObject(response.body);
  const id = body.id || body.task_id;
  if (typeof id !== "string" || !id.trim()) throw new Error("Yueqi did not return a video task id");
  return { taskId: id, taskData: response.body };
}

export function buildQueryRequest(ctx) {
  return {
    url: ctx.baseUrl.replace(/\/$/, "") + "/v1/videos/" + encodeURIComponent(ctx.taskId),
    method: "GET",
    headers: { Authorization: "Bearer " + ctx.apiKey },
  };
}

export function parseTaskResult(ctx, response) {
  const body = videoObject(response);
  const id = body.id || body.task_id;
  if (id !== undefined && id !== ctx.taskId) return { status: "UNKNOWN", reason: "Yueqi returned a different video task id" };
  const statuses = {
    queued: "QUEUED",
    pending: "QUEUED",
    not_start: "NOT_START",
    submitted: "SUBMITTED",
    in_progress: "IN_PROGRESS",
    processing: "IN_PROGRESS",
    running: "IN_PROGRESS",
    completed: "SUCCESS",
    succeeded: "SUCCESS",
    success: "SUCCESS",
    failed: "FAILURE",
    failure: "FAILURE",
    cancelled: "FAILURE",
    canceled: "FAILURE",
  };
  const key = String(body.status || "").toLowerCase();
  if (!Object.prototype.hasOwnProperty.call(statuses, key)) return { status: "UNKNOWN", reason: "Unrecognized Yueqi video task status" };
  const status = statuses[key];
  if (status === "FAILURE") return { status: status, reason: (body.error && body.error.message) || body.fail_reason || "Yueqi video generation failed" };
  const storage = String(body.storage_status || "").toLowerCase();
  if (status === "SUCCESS" && ["failed", "failure", "error"].includes(storage)) return { status: "FAILURE", reason: "Yueqi video storage failed" };
  const url = resultURL(body);
  // A generation can report completed/100 before its signed result is ready.
  // Keep the original task polling rather than settling an undeliverable video.
  if (status === "SUCCESS" && !url) return { status: "IN_PROGRESS", progress: "99%" };
  if (status === "SUCCESS") return { status: status, progress: "100%" };
  const progress = Number(String(body.progress === undefined ? "" : body.progress).replace(/%$/, ""));
  return Number.isFinite(progress) && progress >= 0 ? { status: status, progress: Math.min(progress, 99) + "%" } : { status: status };
}

export function extractUsageOnComplete(ctx, result, response) {
  if (result.status !== "SUCCESS") return null;
  const body = videoObject(response);
  const facts = {};
  // Only output video duration replaces seconds; generation wall time and
  // reference duration never replace it. Measured reference_seconds stays frozen.
  // A per-call expression ignores this fact and continues charging once.
  if (body.seconds !== undefined) facts.seconds = secondsValue(body.seconds);
  else if (body.duration !== undefined) facts.seconds = secondsValue(body.duration);
  let resolution = body.resolution || body.resolution_name || body.size;
  if (typeof resolution === "string") resolution = resolution.trim().toLowerCase();
  if (typeof resolution === "string" && /^[1-9]\d{1,3}p$/.test(resolution)) facts.resolution = resolution;
  return facts;
}

export function listArtifacts(task) {
  return task.status === "SUCCESS" ? [{ key: "video", type: "video", mimeType: "video/mp4" }] : [];
}

export function buildContentRequest(ctx) {
  if (ctx.artifactKey !== "video") throw new Error("artifact_not_found");
  // Refresh expiring signed URLs through the authenticated task endpoint.
  // Credentials never accompany the signed URL or an object-storage/CDN URL.
  const response = utils.fetch({
    url: ctx.baseUrl.replace(/\/$/, "") + "/v1/videos/" + encodeURIComponent(ctx.upstreamTaskId),
    method: "GET",
    headers: { Authorization: "Bearer " + ctx.apiKey },
  });
  if (response.status < 200 || response.status >= 300) throw new Error("Yueqi video result could not be refreshed");
  const body = videoObject(response.body);
  const id = body.id || body.task_id;
  if (id !== undefined && id !== ctx.upstreamTaskId) throw new Error("Yueqi returned a different video task id");
  const url = resultURL(body);
  if (!url) throw new Error("Yueqi video result is not available");
  // Only GET is documented; the host handles a downstream HEAD response.
  return { url: url, method: "GET", credentialless: true };
}

export const protocols = {
  openai_video: {
    decodeRequest: function (ctx) {
      if (!ctx.body || !["json", "multipart"].includes(ctx.body.kind)) throw new Error("JSON or multipart body required");
      let body;
      if (ctx.body.kind === "json") {
        body = ctx.body.value;
        if (!body || typeof body !== "object" || Array.isArray(body)) throw new Error("JSON object required");
      } else {
        body = {};
        for (const name of Object.keys(ctx.body.fields || {})) {
          const values = ctx.body.fields[name];
          if (values.length !== 1 && !MEDIA_FIELDS.includes(name)) throw new Error(name + " must be provided once");
          let value = values.length === 1 ? values[0] : values;
          if (typeof value === "string" && (CONTAINERS.includes(name) || MEDIA_FIELDS.includes(name)) && /^[[{]/.test(value.trim())) {
            try {
              value = JSON.parse(value);
            } catch {
              throw new Error(name + " must contain valid JSON");
            }
          }
          Object.defineProperty(body, name, { value: value, enumerable: true });
        }
      }
      // Model mapping and billing validation run in the driver, where the
      // executing channel's actual upstream model is available.
      return { kind: "submit", model: ctx.model, requestBody: body };
    },
    render: function (ctx, task) {
      return task.data ? videoObject(task.data) : {};
    },
  },
};
