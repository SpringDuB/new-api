export const meta = {
  apiVersion: 1,
  key: "xogpu",
  name: "XOGPU",
  description: {
    en: "XOGPU video generation with text, frames, references, and lip sync",
    zh: "XOGPU 视频生成，支持文本、首尾帧、参考素材和口型同步",
  },
  version: "1.1.0",
  author: { name: "QuantumNous" },
  website: "https://xogpu.com",
  baseUrl: "https://xogpu.com",
  fetchMode: "per_task",
  upstreams: ["vendor", "new_api"],
  // Uses only existing API v1 hooks so it can be uploaded independently.
  // Creation is not idempotent; administrators must disable submission retries
  // in the host's existing settings when the host cannot suppress them per task.
  models: [
    "xogpu-video-text-480p",
    "xogpu-video-text-768p",
    "xogpu-video-frames-480p",
    "xogpu-video-frames-768p",
    "xogpu-video-multiref-quality-480p",
    "xogpu-video-multiref-quality-768p",
    "xogpu-video-multiref-fast-480p",
    "xogpu-video-multiref-fast-768p",
    "xogpu-video-multiref-hd-480p",
    "xogpu-video-multiref-hd-768p",
    "xogpu-video-multiref-hd-1080p",
    "xogpu-video-lipsync-480p",
    "xogpu-video-lipsync-768p",
    "xogpu-video-lipsync-1080p",
  ],
  usageSchema: {
    // Bill the submitted duration (audio_duration for lip sync), default 5.
    // Output metadata never replaces this quantity, even on completion.
    seconds: {
      type: "number",
      unit: "second",
      description: { en: "Video generation unit price", zh: "视频生成单价" },
    },
  },
  protocols: ["openai_video"],
};

function decimalText(value, field) {
  if (typeof value === "number" && Number.isSafeInteger(value) && value >= 0) return String(value);
  if (typeof value !== "string" || !/^\d+$/.test(value)) throw new Error(field + " must be a decimal integer");
  return value.replace(/^0+(?=\d)/, "");
}

// Shared by submission and metering so the billed facts always describe the
// validated request actually sent upstream, after channel model mapping.
function videoSubmission(ctx) {
  const model = ctx.upstreamModel || ctx.model;
  if (!meta.models.includes(model)) throw new Error("unsupported XOGPU video model");
  const match = /^xogpu-video-(.+)-(480p|768p|1080p)$/.exec(model);
  const mode = match[1];
  const quality = match[2];
  const lipsync = mode === "lipsync";
  const req = ctx.requestBody || {};
  if (typeof req !== "object" || Array.isArray(req)) throw new Error("request body must be an object");
  for (const field of Object.keys(req)) {
    if (!["model", "prompt", "duration", "audio_duration", "resolution", "seed", "seconds", "size"].includes(field))
      throw new Error("unsupported field: " + field);
  }
  const durationField = lipsync ? "audio_duration" : "duration";
  const forbiddenDuration = lipsync ? "duration" : "audio_duration";
  if (Object.prototype.hasOwnProperty.call(req, forbiddenDuration)) throw new Error(forbiddenDuration + " is not supported by this model");
  let duration = decimalText(req[durationField] === undefined ? 5 : req[durationField], durationField);
  if (Object.prototype.hasOwnProperty.call(req, "seconds")) {
    const openaiSeconds = decimalText(req.seconds, "seconds");
    if (Object.prototype.hasOwnProperty.call(req, durationField) && openaiSeconds !== duration) throw new Error("seconds conflicts with " + durationField);
    duration = openaiSeconds;
  }
  const seconds = Number(duration);
  const maxSeconds = mode === "multiref-hd" ? 10 : 15;
  if (!Number.isInteger(seconds) || seconds < 1 || seconds > maxSeconds) throw new Error(durationField + " must be between 1 and " + maxSeconds);

  const parts = [{ name: "model", value: model }];
  if (lipsync) {
    if (Object.prototype.hasOwnProperty.call(req, "prompt")) throw new Error("prompt is not supported by lip sync");
  } else {
    if (typeof req.prompt !== "string" || !req.prompt.trim()) throw new Error("prompt is required");
    const prompt = req.prompt.trim();
    if (Array.from(prompt).length > 10000) throw new Error("prompt must not exceed 10000 characters");
    parts.push({ name: "prompt", value: prompt });
  }
  parts.push({ name: durationField, value: duration });

  let resolution = req.resolution === undefined ? quality + "竖" : req.resolution;
  if (Object.prototype.hasOwnProperty.call(req, "size")) {
    // OpenAI dimensions select the layout; output quality is fixed by the
    // mapped XOGPU model, so size cannot silently change the billed model.
    const layouts = {
      "720x1280": "竖",
      "1280x720": "横",
      "1024x1792": "竖",
      "1792x1024": "横",
      "1080x1920": "竖",
      "1920x1080": "横",
      "1024x1024": "(1:1)",
    };
    if (typeof req.size !== "string" || !Object.prototype.hasOwnProperty.call(layouts, req.size)) throw new Error("unsupported video size");
    const mappedResolution = quality + layouts[req.size];
    if (Object.prototype.hasOwnProperty.call(req, "resolution") && resolution !== mappedResolution) throw new Error("size conflicts with resolution");
    resolution = mappedResolution;
  }
  const resolutions = [quality + "横", quality + "竖"];
  if (!lipsync && mode !== "multiref-hd") resolutions.push(quality + "(1:1)");
  if (!resolutions.includes(resolution)) throw new Error("resolution must match the model quality and supported aspect ratios");
  parts.push({ name: "resolution", value: resolution });
  if (Object.prototype.hasOwnProperty.call(req, "seed")) {
    if (lipsync || mode === "text") throw new Error("seed is not supported by this model");
    const seed = decimalText(req.seed, "seed");
    // Keep this as text: XOGPU's maximum seed exceeds JavaScript's safe integer.
    if (seed.length > 16) throw new Error("seed must be between 0 and 9999999999999999");
    parts.push({ name: "seed", value: seed });
  }

  const files = ctx.files || [];
  const inputReferences = files.filter((file) => file.field === "input_reference");
  if (inputReferences.length > 1) throw new Error("input_reference must be provided once");
  const referenceField = mode === "frames" ? "first_frame" : "reference_images";
  if (inputReferences.length && files.some((file) => file.field === referenceField))
    throw new Error("input_reference cannot be combined with " + referenceField);
  const counts = { reference_images: 0, reference_audios: 0, first_frame: 0, last_frame: 0 };
  let totalBytes = 0;
  // Reserve space for bounded text fields, part headers, and boundaries. UTF-8
  // uses at most three bytes per UTF-16 code unit in a supplied MIME header.
  let requestBytes = 64 * 1024;
  for (const file of files) {
    const field = file.field === "input_reference" ? referenceField : file.field;
    if (!Object.prototype.hasOwnProperty.call(counts, field)) throw new Error("unsupported file field: " + field);
    const audio = field === "reference_audios";
    const extension = /\.([^.\\/]+)$/.exec(file.filename || "");
    const ext = extension ? extension[1].toLowerCase() : "";
    if (!(audio ? ["mp3", "wav", "mp4", "flac"] : ["jpg", "jpeg", "png", "webp"]).includes(ext)) throw new Error("unsupported file format: " + file.field);
    const limit = (audio ? 50 : 20) * 1024 * 1024;
    if (!Number.isSafeInteger(file.size) || file.size <= 0 || file.size > limit) throw new Error("invalid file size: " + file.field);
    if (typeof file.ref !== "string" || !file.ref) throw new Error("uploaded file reference is required");
    totalBytes += file.size;
    if (totalBytes > 95 * 1024 * 1024) throw new Error("total media must not exceed 95 MiB");
    requestBytes += file.size + String(file.mimeType || "").length * 3 + 512;
    if (requestBytes > 100 * 1024 * 1024) throw new Error("multipart request must not exceed 100 MiB");
    counts[field]++;
    // Normalize filenames so arbitrary client names cannot enlarge MIME headers.
    parts.push({ name: field, fileRef: file.ref, filename: field + "-" + counts[field] + "." + ext });
  }
  if (mode === "frames") {
    if (counts.first_frame !== 1 || counts.last_frame !== 1 || counts.reference_images || counts.reference_audios)
      throw new Error("frames requires exactly one first_frame and one last_frame");
  } else {
    if (counts.first_frame || counts.last_frame) throw new Error("frame files are only supported by frames models");
    if (mode === "text" && (counts.reference_images || counts.reference_audios)) throw new Error("text models do not accept media");
    if (lipsync && (counts.reference_images !== 1 || counts.reference_audios !== 1)) throw new Error("lip sync requires exactly one image and one audio");
    if (["multiref-quality", "multiref-fast"].includes(mode) && counts.reference_images < 1) throw new Error("at least one reference image is required");
    if (counts.reference_images > 9 || counts.reference_audios > 3) throw new Error("at most 9 images and 3 audios are allowed");
  }
  return { parts, seconds, action: mode === "text" || (mode === "multiref-hd" && !counts.reference_images) ? "text_to_video" : "image_to_video" };
}

export function buildSubmitRequest(ctx) {
  const submission = videoSubmission(ctx);
  return {
    url: ctx.baseUrl + "/v1/videos",
    method: "POST",
    headers: { Authorization: "Bearer " + ctx.apiKey },
    bodyType: "multipart",
    parts: submission.parts,
    action: submission.action,
  };
}

export function extractUsage(ctx) {
  return { seconds: videoSubmission(ctx).seconds };
}

export function parseSubmitResponse(ctx, resp) {
  const body = resp.body;
  if (!body || typeof body !== "object" || Array.isArray(body)) throw new Error("XOGPU returned an invalid task response");
  if (body.error || body.code) throw new Error((body.error && body.error.message) || body.message || "XOGPU rejected the submission");
  if (typeof body.id !== "string" || !body.id.trim()) throw new Error("XOGPU returned an empty task id");
  return { taskId: body.id.trim(), taskData: body };
}

export function buildQueryRequest(ctx) {
  return { url: ctx.baseUrl + "/v1/videos/" + encodeURIComponent(ctx.taskId), method: "GET", headers: { Authorization: "Bearer " + ctx.apiKey } };
}

export function parseTaskResult(ctx, body) {
  const statuses = { queued: "QUEUED", in_progress: "IN_PROGRESS", completed: "SUCCESS", failed: "FAILURE" };
  const status = body && Object.prototype.hasOwnProperty.call(statuses, body.status) ? statuses[body.status] : "UNKNOWN";
  const result = { status };
  if (status === "UNKNOWN") result.reason = "Unrecognized XOGPU task status";
  if (status === "FAILURE") result.reason = (body.error && body.error.message) || body.message || "Video generation task failed";
  if (body && typeof body.progress === "number" && Number.isFinite(body.progress) && body.progress >= 0 && body.progress <= 100)
    result.progress = body.progress + "%";
  return result;
}

export function listArtifacts(task) {
  return task.status === "SUCCESS" ? [{ key: "video", type: "video", mimeType: "video/mp4" }] : [];
}

export function buildContentRequest(ctx) {
  if (ctx.artifactKey !== "video") throw new Error("artifact_not_found");
  return {
    url: ctx.baseUrl + "/v1/videos/" + encodeURIComponent(ctx.upstreamTaskId) + "/content",
    // Only GET is documented upstream; the host omits the body for HEAD clients.
    method: "GET",
    headers: { Authorization: "Bearer " + ctx.apiKey },
  };
}

export const protocols = {
  openai_video: {
    decodeRequest: function (ctx) {
      if (!ctx.body || ctx.body.kind !== "multipart") throw new Error("multipart body required");
      const req = {};
      for (const field of Object.keys(ctx.body.fields || {})) {
        const values = ctx.body.fields[field];
        if (values.length !== 1) throw new Error(field + " must be provided once");
        Object.defineProperty(req, field, { value: values[0], enumerable: true, writable: true, configurable: true });
      }
      // Mode validation runs in the driver after channel mapping. A local alias
      // has no XOGPU mode until ctx.upstreamModel is available there.
      return { kind: "submit", model: ctx.model, requestBody: req };
    },
    render: function (ctx, task) {
      return task.data && typeof task.data === "object" && !Array.isArray(task.data) ? task.data : {};
    },
  },
};
