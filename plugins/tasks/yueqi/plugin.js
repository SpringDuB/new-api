const PUBLIC_MODEL = "seedance-2.5-超低价";
const VENDOR_MODEL = "kk-sd25";
const REQUEST_USAGE_SCHEMA = {
  requests: {
    type: "number",
    unit: "count",
    description: { en: "Video generation request unit price", zh: "视频生成按次单价" },
  },
};
const CONTAINERS = ["values", "metadata", "extra", "parameters"];

export const meta = {
  apiVersion: 1,
  key: "yueqi",
  name: "Yueqi Video",
  icon: "text:YQ",
  description: {
    en: "Seedance 2.5 text-to-video generation through Yueqi",
    zh: "通过 Yueqi 调用 Seedance 2.5 文生视频",
  },
  version: "1.3.0",
  author: { name: "QuantumNous" },
  website: "https://yueqi.icu/docs",
  baseUrl: "https://yueqi.icu",
  fetchMode: "per_task",
  upstreams: ["vendor", "new_api"],
  // An accepted creation has no documented idempotency key. Never retry an
  // ambiguous submission, even on another channel; poll its original id.
  requiredCapabilities: ["submit-no-retry@1"],
  models: [PUBLIC_MODEL],
  modelDefaults: {
    [PUBLIC_MODEL]: {
      displayName: PUBLIC_MODEL,
      vendor: "ByteDance",
      description: { en: "720p text-to-video generation only", zh: "仅支持 720p 文生视频" },
      icon: "Doubao.Color",
      tags: ["video", "text-to-video", "720p"],
      // Same platform CNY/USD 1:1 pricing convention as LecAPI.
      billingExpr: 'tier("base", u("requests") * 3)',
    },
  },
  // Older in-flight tasks can still settle by the resolution returned by Yueqi.
  usageSchema: {
    ...REQUEST_USAGE_SCHEMA,
    resolution: { enum: ["720p"], allowCustom: true, description: { en: "Output video resolution", zh: "输出视频分辨率" } },
  },
  usageProfiles: [{ models: [PUBLIC_MODEL], schema: REQUEST_USAGE_SCHEMA }],
  protocols: ["openai_video"],
};

function secondsValue(value) {
  if (typeof value !== "number" && (typeof value !== "string" || !/^\d+(?:\.\d+)?$/.test(value.trim())))
    throw new Error("seconds must be a decimal number between 1 and 60");
  const seconds = Number(value);
  // Retain the Yueqi duration bound; the upstream enforces narrower limits.
  if (!Number.isFinite(seconds) || seconds < 1 || seconds > 60) throw new Error("seconds must be between 1 and 60");
  return seconds;
}

// Normalize client and nested parameters to the fields in Yueqi's text request spec.
function videoSubmission(ctx) {
  const requestedModel = ctx.upstreamModel || ctx.model;
  if (requestedModel !== PUBLIC_MODEL && requestedModel !== VENDOR_MODEL) throw new Error("unsupported Yueqi video model");
  if (ctx.files?.length) throw new Error(PUBLIC_MODEL + " only supports text-to-video without uploaded files");
  const input = ctx.requestBody;
  if (!input || typeof input !== "object" || Array.isArray(input)) throw new Error("request body must be an object");
  const scopes = [input];
  const body = { model: ctx.upstream?.kind === "new_api" ? requestedModel : VENDOR_MODEL };
  for (let i = 0; i < scopes.length; i++) {
    for (const [field, inputValue] of Object.entries(scopes[i])) {
      let value = inputValue;
      let target = field;
      if (CONTAINERS.includes(field)) {
        if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error(field + " must be an object");
        scopes.push(value);
        if (scopes.length > 64) throw new Error("too many video parameter containers");
        continue;
      }
      if (field === "model") {
        if (i > 0 && value !== ctx.model && value !== requestedModel && value !== body.model) throw new Error("nested model must match the channel model");
        continue;
      }
      if (["n", "count", "batch_size", "batch_count", "num_videos"].includes(field)) {
        if (value !== 1 && value !== "1") throw new Error("Yueqi video requests must generate exactly one video");
        continue;
      }
      if (field === "mode" && ["text", "text_to_video"].includes(value)) continue;
      if (field === "seconds") target = "duration";
      if (field === "resolution_name") target = "resolution";
      if (field === "ratio") target = "aspect_ratio";
      if (field === "size") target = typeof value === "string" && value.includes(":") ? "aspect_ratio" : "resolution";
      switch (target) {
        case "prompt":
          if (typeof value !== "string" || !value.trim()) throw new Error("prompt is required");
          break;
        case "duration":
          value = secondsValue(value);
          break;
        case "resolution":
          value = typeof value === "string" ? value.trim().toLowerCase() : value;
          if (value !== "720p") throw new Error(PUBLIC_MODEL + " only supports 720p resolution");
          break;
        case "aspect_ratio":
          value = typeof value === "string" ? value.trim() : value;
          if (typeof value !== "string" || !/^\d{1,2}:\d{1,2}$/.test(value) || value.split(":").some((part) => Number(part) === 0))
            throw new Error("aspect_ratio must be a positive width:height ratio");
          break;
        default:
          throw new Error(field + " is not supported by this text-to-video model");
      }
      if (body[target] !== undefined && body[target] !== value) throw new Error("conflicting " + target + " parameters");
      body[target] = value;
    }
  }
  if (body.prompt === undefined) throw new Error("prompt is required");
  body.duration ??= 5;
  body.resolution ??= "720p";
  return body;
}

export function buildSubmitRequest(ctx) {
  return {
    url: ctx.baseUrl.replace(/\/$/, "") + "/v1/videos",
    method: "POST",
    headers: { Authorization: "Bearer " + ctx.apiKey, "Content-Type": "application/json" },
    body: videoSubmission(ctx),
    action: "text_to_video",
  };
}

export function extractUsage(ctx) {
  videoSubmission(ctx);
  return { requests: 1 };
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
  const model = ctx.upstreamModel || ctx.model;
  if (!model || model === PUBLIC_MODEL || model === VENDOR_MODEL) return null;
  // Only older tasks need actual output duration and resolution for settlement.
  const body = videoObject(response);
  const facts = {};
  if (body.seconds !== undefined) facts.seconds = secondsValue(body.seconds);
  else if (body.duration !== undefined) facts.seconds = secondsValue(body.duration);
  let resolution = body.resolution || body.resolution_name || body.size;
  if (typeof resolution === "string") resolution = resolution.trim().toLowerCase();
  if (typeof resolution === "string" && /^(?:[1-9]\d{1,3}p|2k)$/.test(resolution)) facts.resolution = resolution;
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
        if (ctx.body.files?.length) throw new Error(PUBLIC_MODEL + " only supports text-to-video without uploaded files");
        body = {};
        for (const name of Object.keys(ctx.body.fields || {})) {
          const values = ctx.body.fields[name];
          if (values.length !== 1) throw new Error(name + " must be provided once");
          let value = values[0];
          if (CONTAINERS.includes(name)) {
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
