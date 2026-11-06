const MODEL = "runninghub-minimax-h3";
const WORKFLOW_ID = "2111027801062203393";
const ASPECT_RATIOS = {
  "1:1": "1:1 (Square)",
  "2:3": "2:3 (Portrait Photo)",
  "3:2": "3:2 (Photo)",
  "3:4": "3:4 (Portrait Standard)",
  "4:3": "4:3 (Standard)",
  "16:9": "16:9 (Widescreen)",
  "9:16": "9:16 (Portrait Widescreen)",
  "21:9": "21:9 (Ultrawide)",
};
const IMAGE_NODES = ["408", "409", "410", "411", "412", "413", "414", "415", "416"];
const AUDIO_NODES = ["14", "15", "48"];
const VIDEO_NODES = ["25", "26", "27"];

export const meta = {
  apiVersion: 1,
  key: "runninghub-h3",
  name: "RunningHub MiniMax H3",
  icon: "text:RH",
  description: {
    en: "MiniMax H3 video generation with reference images, audio, and video through a dedicated RunningHub workflow",
    zh: "通过专用 RunningHub 工作流调用 MiniMax H3，支持图片、音频和视频参考",
  },
  version: "1.0.2",
  author: { name: "QuantumNous" },
  website: "https://www.runninghub.cn",
  baseUrl: "https://www.runninghub.cn",
  fetchMode: "per_task",
  requiredCapabilities: ["submit-no-retry@1"],
  models: ["runninghub-minimax-h3"],
  usageSchema: {
    // The first five reference images are included; only additional images
    // contribute to this fact. Configure each unit price in the host's model
    // pricing editor, which stores the resulting expression in USD.
    extra_images: {
      type: "number",
      unit: "count",
      unitLabel: { en: "image", zh: "张" },
      description: { en: "Additional reference image unit price", zh: "超额参考图单价" },
    },
    input_audios: {
      type: "number",
      unit: "count",
      unitLabel: { en: "audio", zh: "个" },
      description: { en: "Reference audio unit price", zh: "参考音频单价" },
    },
    input_video_seconds: {
      type: "number",
      unit: "second",
      description: { en: "Reference video unit price", zh: "参考视频单价" },
    },
    seconds: {
      type: "number",
      unit: "second",
      description: { en: "Video generation unit price", zh: "视频生成单价" },
    },
  },
  usageExamples: [
    { label: "768p · 10s", facts: { extra_images: 0, input_audios: 0, input_video_seconds: 0, seconds: 10 } },
    { label: "768p · 10s · multimodal", facts: { extra_images: 1, input_audios: 1, input_video_seconds: 5, seconds: 10 } },
  ],
  routes: [
    { method: "POST", path: "/runninghub-h3/v1/videos", type: "submit", decode: "createVideo", render: "video" },
    { method: "GET", path: "/runninghub-h3/v1/videos/:task_id", type: "query", render: "video" },
  ],
  protocols: ["openai_video"],
};

function secondsValue(value, field, minimum, maximum) {
  if (typeof value !== "number" && (typeof value !== "string" || !/^\d+(?:\.\d+)?$/.test(value.trim())))
    throw new Error(field + " must be a decimal number");
  const seconds = Number(value);
  if (!Number.isFinite(seconds) || seconds < minimum || seconds > maximum)
    throw new Error(field + " must be between " + minimum + " and " + maximum);
  return seconds;
}

function referenceMedia(value, field, files) {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    const file = (files || []).find(function (item) { return item.ref === value.__fileRef; });
    if (!file || value.encoding !== "dataUrl") throw new Error(field + " contains an invalid uploaded file reference");
    for (const key of Object.keys(value)) {
      if (key !== "__fileRef" && key !== "encoding") throw new Error("unsupported uploaded file field: " + key);
    }
    if (!Number.isSafeInteger(file.size) || file.size <= 0 || file.size > 64 * 1024 * 1024) throw new Error("uploaded file must be between 1 byte and 64 MiB");
    return { __fileRef: file.ref, encoding: "dataUrl" };
  }
  // RunningHub accepts its uploaded resource names, public URLs, and data URIs.
  // References are forwarded only to the provider, never opened by the host.
  if (typeof value !== "string" || !value.trim() || value.trim().toLowerCase() === "none" || /[\u0000-\u001f\u007f]/.test(value))
    throw new Error(field + " must contain a non-empty media reference");
  return value.trim();
}

// Both request construction and metering use this validated representation.
// No metadata or vendor body can override a node or a billable quantity.
function workflowSubmission(ctx) {
  if ((ctx.upstreamModel || ctx.model) !== MODEL) throw new Error("unsupported RunningHub MiniMax H3 model");
  const source = ctx.requestBody || {};
  if (typeof source !== "object" || Array.isArray(source)) throw new Error("request body must be an object");
  const req = {};
  for (const field of Object.keys(source)) {
    if (!["model", "prompt", "seconds", "duration", "aspect_ratio", "size", "resolution", "resolution_name", "images", "input_reference", "audios", "videos", "input", "parameters"].includes(field))
      throw new Error("unsupported field: " + field);
    if (field !== "input" && field !== "parameters") req[field] = source[field];
  }
  for (const container of ["input", "parameters"]) {
    if (source[container] === undefined) continue;
    const nested = source[container];
    if (!nested || typeof nested !== "object" || Array.isArray(nested)) throw new Error(container + " must be an object");
    const allowed = container === "input" ? ["prompt", "media"] : ["duration", "seconds", "ratio", "resolution", "resolution_name", "prompt_extend", "watermark"];
    for (const key of Object.keys(nested)) {
      if (!allowed.includes(key)) throw new Error("unsupported " + container + " field: " + key);
      const value = nested[key];
      if (key === "prompt_extend" || key === "watermark") {
        if (value !== false) throw new Error(container + "." + key + " only supports false for this workflow");
        continue;
      }
      if (key === "media") {
        if (!Array.isArray(value)) throw new Error("input.media must be an array");
        if (value.length > 15) throw new Error("input.media must contain at most 15 references");
        if (["images", "input_reference", "audios", "videos"].some(function (field) { return req[field] !== undefined; }))
          throw new Error("input.media cannot be combined with top-level media fields");
        req.images = [];
        req.audios = [];
        req.videos = [];
        let firstFrame;
        for (const media of value) {
          if (!media || typeof media !== "object" || Array.isArray(media)) throw new Error("input.media entries must be objects");
          for (const field of Object.keys(media)) {
            if (!["type", "url", "duration", "seconds"].includes(field)) throw new Error("unsupported input.media field: " + field);
          }
          const url = referenceMedia(media.url, "input.media.url");
          if (["first_frame", "image", "reference_image"].includes(media.type)) {
            if (media.duration !== undefined || media.seconds !== undefined) throw new Error("image references do not accept duration");
            if (media.type === "first_frame") {
              if (firstFrame !== undefined) throw new Error("input.media accepts at most one first_frame");
              firstFrame = url;
            } else req.images.push(url);
          } else if (["audio", "reference_audio"].includes(media.type)) {
            if (media.duration !== undefined || media.seconds !== undefined) throw new Error("audio references do not accept duration");
            req.audios.push(url);
          } else if (["video", "reference_video"].includes(media.type)) {
            const seconds = secondsValue(media.seconds === undefined ? media.duration : media.seconds, "video seconds", 2, 15);
            if (media.seconds !== undefined && media.duration !== undefined && seconds !== secondsValue(media.duration, "video duration", 2, 15))
              throw new Error("video seconds conflicts with duration");
            req.videos.push({ url: url, seconds: seconds });
          } else throw new Error("unsupported input.media type");
        }
        if (firstFrame !== undefined) req.images.unshift(firstFrame);
        continue;
      }
      const field = key === "ratio" ? "aspect_ratio" : key;
      if (req[field] !== undefined) {
        const duration = field === "duration" || field === "seconds";
        if (duration ? secondsValue(req[field], field, 3, 15) !== secondsValue(value, container + "." + key, 3, 15) : req[field] !== value)
          throw new Error(container + "." + key + " conflicts with " + field);
      }
      req[field] = value;
    }
  }
  let seconds = 10;
  if (req.duration !== undefined) seconds = secondsValue(req.duration, "duration", 3, 15);
  if (Object.prototype.hasOwnProperty.call(req, "seconds")) {
    const explicit = secondsValue(req.seconds, "seconds", 3, 15);
    if (req.duration !== undefined && explicit !== seconds) throw new Error("seconds conflicts with duration");
    seconds = explicit;
  }
  let ratio = req.aspect_ratio === undefined ? "16:9" : req.aspect_ratio;
  if (req.size !== undefined) {
    if (typeof req.size !== "string") throw new Error("size must be an aspect ratio or pixel dimensions");
    let sizeRatio = req.size;
    const dimensions = /^(\d{1,5})x(\d{1,5})$/.exec(req.size);
    if (dimensions) {
      const width = Number(dimensions[1]);
      const height = Number(dimensions[2]);
      sizeRatio = Object.keys(ASPECT_RATIOS).find(function (candidate) {
        const parts = candidate.split(":");
        return width > 0 && height > 0 && width * Number(parts[1]) === height * Number(parts[0]);
      });
    }
    if (!Object.prototype.hasOwnProperty.call(ASPECT_RATIOS, sizeRatio)) throw new Error("unsupported video size");
    if (req.aspect_ratio !== undefined && ratio !== sizeRatio) throw new Error("size conflicts with aspect_ratio");
    ratio = sizeRatio;
  }
  if (typeof ratio !== "string" || !Object.prototype.hasOwnProperty.call(ASPECT_RATIOS, ratio)) throw new Error("unsupported aspect_ratio");
  // Every supplied resolution is normalized to the workflow's fixed 768p
  // setting: node 252 megapixels = 1, with its existing multiple = 32.
  if (req.resolution !== undefined && typeof req.resolution !== "string") throw new Error("resolution must be a string");
  if (req.resolution_name !== undefined && typeof req.resolution_name !== "string") throw new Error("resolution_name must be a string");
  const nodes = [
    { nodeId: "252", fieldName: "megapixels", fieldValue: 1 },
    { nodeId: "252", fieldName: "aspect_ratio", fieldValue: ASPECT_RATIOS[ratio] },
    { nodeId: "259", fieldName: "value", fieldValue: seconds },
  ];
  if (req.prompt !== undefined) {
    if (typeof req.prompt !== "string" || !req.prompt.trim() || Array.from(req.prompt).length > 10000)
      throw new Error("prompt must contain between 1 and 10000 characters");
    nodes.push({ nodeId: "263", fieldName: "text", fieldValue: req.prompt.trim() });
  }
  let images = req.images === undefined ? [] : req.images;
  if (req.input_reference !== undefined) {
    if (req.images !== undefined) throw new Error("input_reference cannot be combined with images");
    images = [req.input_reference];
  }
  const audios = req.audios === undefined ? [] : req.audios;
  const videos = req.videos === undefined ? [] : req.videos;
  if (!Array.isArray(images) || images.length > IMAGE_NODES.length) throw new Error("images must be an array of at most 9 references");
  if (!Array.isArray(audios) || audios.length > AUDIO_NODES.length) throw new Error("audios must be an array of at most 3 references");
  if (!Array.isArray(videos) || videos.length > VIDEO_NODES.length) throw new Error("videos must be an array of at most 3 references");
  const references = [
    { values: images, nodeIds: IMAGE_NODES, field: "image", input: "ref_images.ref_image_" },
    { values: audios, nodeIds: AUDIO_NODES, field: "audio", input: "ref_audios.ref_audio_" },
    { values: videos, nodeIds: VIDEO_NODES, field: "video", input: "ref_videos.ref_video_" },
  ];
  let videoSeconds = 0;
  for (const reference of references) {
    for (let i = 0; i < reference.nodeIds.length; i++) {
      if (i >= reference.values.length) {
        // Disconnect unused Autogrow slots at the conditioning node, rather
        // than leaving the export's example images or invalid empty loaders
        // attached. MiniMaxH3ReferenceToVideo skips null reference values.
        nodes.push({ nodeId: "265", fieldName: reference.input + i, fieldValue: null });
        continue;
      }
      const value = reference.values[i];
      let media = value;
      if (reference.field === "video") {
        if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("each video must contain url and seconds");
        for (const field of Object.keys(value)) {
          if (field !== "url" && field !== "seconds") throw new Error("unsupported video field: " + field);
        }
        const requestedSeconds = secondsValue(value.seconds, "video seconds", 2, 15);
        const frameCap = Math.floor(requestedSeconds * 24);
        // Bill the requested reference clip at 24 fps, never an untrusted
        // client duration detached from the loaded frames. Longer sources
        // are trimmed; a shorter source still uses the reserved clip price.
        videoSeconds += frameCap / 24;
        nodes.push(
          { nodeId: reference.nodeIds[i], fieldName: "force_rate", fieldValue: 24 },
          { nodeId: reference.nodeIds[i], fieldName: "frame_load_cap", fieldValue: frameCap },
          { nodeId: reference.nodeIds[i], fieldName: "skip_first_frames", fieldValue: 0 },
          { nodeId: reference.nodeIds[i], fieldName: "select_every_nth", fieldValue: 1 }
        );
        media = value.url;
      }
      nodes.push({ nodeId: reference.nodeIds[i], fieldName: reference.field, fieldValue: referenceMedia(media, reference.field, ctx.files) });
    }
  }
  return {
    nodes: nodes,
    instanceType: videos.length ? "plus" : "default",
    action: videos.length || audios.length || images.length > 1 ? "reference_to_video" : images.length ? "image_to_video" : "text_to_video",
    facts: { extra_images: Math.max(0, images.length - 5), input_audios: audios.length, input_video_seconds: videoSeconds, seconds: seconds },
  };
}

export function buildSubmitRequest(ctx) {
  const submission = workflowSubmission(ctx);
  for (const node of submission.nodes) {
    if (!node.fieldValue || typeof node.fieldValue !== "object" || !node.fieldValue.__fileRef) continue;
    if (typeof utils.uploadFile !== "function") throw new Error("Update the gateway to support RunningHub file uploads, or use the New API video interface with media URLs");
    const response = utils.uploadFile({
      url: ctx.baseUrl.replace(/\/+$/, "") + "/openapi/v2/media/upload/binary",
      headers: { Authorization: "Bearer " + ctx.apiKey },
      fileRef: node.fieldValue.__fileRef,
    });
    const body = response.body;
    if (response.status < 200 || response.status >= 300 || !body || body.code !== 0 || !body.data || typeof body.data.fileName !== "string" || !body.data.fileName.trim())
      throw new Error("RunningHub media upload failed");
    node.fieldValue = body.data.fileName;
  }
  return {
    url: ctx.baseUrl.replace(/\/+$/, "") + "/openapi/v2/run/workflow/" + WORKFLOW_ID,
    method: "POST",
    headers: { Authorization: "Bearer " + ctx.apiKey, "Content-Type": "application/json" },
    body: { addMetadata: true, nodeInfoList: submission.nodes, instanceType: submission.instanceType, usePersonalQueue: false },
    action: submission.action,
  };
}

export function extractUsage(ctx) {
  if (ctx.usagePurpose === "billing_ratios") return null;
  return workflowSubmission(ctx).facts;
}

export function parseSubmitResponse(ctx, response) {
  const body = response.body;
  if (!body || typeof body !== "object" || Array.isArray(body)) throw new Error("RunningHub returned an invalid task response");
  if (body.errorCode && String(body.errorCode) !== "0") throw new Error(body.errorMessage || "RunningHub rejected the submission");
  if (typeof body.taskId !== "string" || !body.taskId.trim()) throw new Error("RunningHub returned an empty task id");
  const result = parseTaskResult({}, body);
  if (result.status === "UNKNOWN") throw new Error(result.reason);
  const parsed = { taskId: body.taskId.trim(), taskData: body };
  if (result.status === "SUCCESS" || result.status === "FAILURE") parsed.immediate = result;
  return parsed;
}

export function buildQueryRequest(ctx) {
  return {
    url: ctx.baseUrl.replace(/\/+$/, "") + "/openapi/v2/query",
    method: "POST",
    headers: { Authorization: "Bearer " + ctx.apiKey, "Content-Type": "application/json" },
    body: { taskId: ctx.taskId },
  };
}

function outputVideo(data) {
  if (!data || !Array.isArray(data.results)) return null;
  const mimeTypes = { mp4: "video/mp4", webm: "video/webm", mov: "video/quicktime", mkv: "video/x-matroska" };
  for (const result of data.results) {
    if (!result || typeof result !== "object") continue;
    if (result.nodeId !== undefined && String(result.nodeId) !== "418") continue;
    if (typeof result.outputType !== "string" || !Object.prototype.hasOwnProperty.call(mimeTypes, result.outputType.toLowerCase())) continue;
    if (typeof result.url === "string" && /^https?:\/\//i.test(result.url)) return { url: result.url, mimeType: mimeTypes[result.outputType.toLowerCase()] };
  }
  return null;
}

export function parseTaskResult(ctx, body) {
  if (!body || typeof body !== "object" || Array.isArray(body)) return { status: "UNKNOWN", reason: "Invalid RunningHub task response" };
  if (ctx.taskId && body.taskId && ctx.taskId !== body.taskId) return { status: "UNKNOWN", reason: "RunningHub returned a different task id" };
  const statuses = { QUEUED: "QUEUED", RUNNING: "IN_PROGRESS", SUCCESS: "SUCCESS", FAILED: "FAILURE" };
  const status = Object.prototype.hasOwnProperty.call(statuses, body.status) ? statuses[body.status] : "UNKNOWN";
  if (status === "UNKNOWN") return { status: status, reason: "Unrecognized RunningHub task status" };
  if (status === "FAILURE") return { status: status, progress: "100%", reason: body.errorMessage || "RunningHub video generation failed" };
  if (body.errorCode && String(body.errorCode) !== "0") return { status: "UNKNOWN", reason: body.errorMessage || "RunningHub query failed" };
  if (status === "SUCCESS") {
    const video = outputVideo(body);
    if (!video) return { status: "UNKNOWN", reason: "RunningHub completed without an output video" };
    return { status: status, progress: "100%", url: video.url };
  }
  return { status: status, progress: status === "QUEUED" ? "0%" : "50%" };
}

export function extractUsageOnComplete() {
  // usage.taskCostTime is GPU runtime, not input or output media duration.
  // Preserve the bounded submitted quantities; upstream money is not usage.
  return null;
}

export function listArtifacts(task) {
  const video = outputVideo(task.data);
  return task.status === "SUCCESS" && video ? [{ key: "video", type: "video", mimeType: video.mimeType }] : [];
}

export function buildContentRequest(ctx) {
  const video = outputVideo(ctx.data);
  if (ctx.artifactKey !== "video" || !video) throw new Error("artifact_not_found");
  // RunningHub links expire after 24 hours. The gateway proxies downloads but
  // does not extend the provider's retention or send API credentials to its CDN.
  return { url: video.url, method: ctx.clientRequest.method, credentialless: true };
}

function decodeVideo(ctx) {
  if (!ctx.body || !["json", "multipart"].includes(ctx.body.kind)) throw new Error("JSON or multipart body required");
  let req;
  if (ctx.body.kind === "json") {
    req = ctx.body.value;
    if (!req || typeof req !== "object" || Array.isArray(req)) throw new Error("JSON object required");
  } else {
    req = {};
    const fields = ctx.body.fields || {};
    for (const name of Object.keys(fields)) {
      const values = fields[name];
      const isMediaList = ["images", "audios", "videos"].includes(name);
      if (!isMediaList && values.length !== 1) throw new Error(name + " must be provided once");
      let value = values[0];
      if (name === "input" || name === "parameters") {
        try {
          value = JSON.parse(value);
        } catch (e) {
          throw new Error(name + " must contain a JSON object");
        }
      } else if (isMediaList) {
        // Repeated fields preserve reference order. A JSON array in a text
        // field is also accepted, without decoding signed URL query strings.
        value = [];
        for (const text of values) {
          if (name !== "videos" && !text.trim().startsWith("[")) {
            value.push(text);
            continue;
          }
          let parsed;
          try {
            parsed = JSON.parse(text);
          } catch (e) {
            throw new Error(name + " must contain valid JSON media references");
          }
          if (Array.isArray(parsed)) {
            for (const entry of parsed) value.push(entry);
          } else if (name === "videos" && parsed && typeof parsed === "object") {
            value.push(parsed);
          } else {
            throw new Error(name + " must contain a JSON array or video object");
          }
        }
      }
      // Preserve every field for workflowSubmission's allowlist, including
      // names such as __proto__, rather than invoking object setters.
      Object.defineProperty(req, name, { value: value, enumerable: true, writable: true });
    }
    for (const file of ctx.body.files || []) {
      if (file.field !== "input_reference") throw new Error("unsupported file field: " + file.field);
      if (req.input_reference !== undefined || req.images !== undefined || req.input !== undefined)
        throw new Error("input_reference file cannot be combined with other image references");
      if (typeof file.ref !== "string" || !file.ref || !Number.isSafeInteger(file.size) || file.size <= 0 || file.size > 64 * 1024 * 1024)
        throw new Error("input_reference file must be between 1 byte and 64 MiB");
      if (!["image/png", "image/jpeg", "image/webp"].includes(String(file.mimeType || "").toLowerCase()))
        throw new Error("input_reference must be a PNG, JPEG, or WebP image");
      req.input_reference = { __fileRef: file.ref, encoding: "dataUrl" };
    }
  }
  const model = ctx.model || req.model || MODEL;
  if (typeof model !== "string" || !model.trim()) throw new Error("model is required");
  return { kind: "submit", model: model, requestBody: req };
}

function renderVideo(ctx, task) {
  const statuses = { NOT_START: "queued", SUBMITTED: "queued", QUEUED: "queued", IN_PROGRESS: "in_progress", SUCCESS: "completed", FAILURE: "failed" };
  const progress = Number(String(task.progress || "0").replace("%", ""));
  const output = {
    id: task.task_id,
    object: "video",
    model: (task.properties && task.properties.origin_model_name) || ctx.model || MODEL,
    status: statuses[task.status] || "unknown",
    progress: Number.isFinite(progress) ? Math.max(0, Math.min(100, progress)) : 0,
    created_at: task.created_at,
    resolution: "768p",
  };
  const video = outputVideo(task.data);
  if (video && task.status === "SUCCESS") {
    output.url = video.url;
    if (task.updated_at) output.completed_at = task.updated_at;
  }
  if (task.status === "FAILURE") output.error = { code: "task_failed", message: task.fail_reason || "RunningHub video generation failed" };
  return output;
}

export const native = { createVideo: decodeVideo, video: renderVideo };
export const protocols = { openai_video: { decodeRequest: decodeVideo, render: renderVideo } };
