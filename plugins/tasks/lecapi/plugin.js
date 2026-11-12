// LecAPI public media catalog and documentation snapshot: 2026-11-12.
// Sources: https://api.lecapi.net/api/media-models/catalog, /api/pricing, and /docs.
// Static media kinds take precedence over inconsistent /v1/models labels.
// User pricing policy: published CNY price +20%, platform CNY/USD units 1:1.
// Each expression contains the final marked-up numeric coefficient, with no FX
// conversion or hidden multiplier. Rounding only removes binary float noise.
// MF30ref uses its explicit 720p tier (5.5), rather than the stale 5-CNY summary.
// TinySnow Image 2 follows its Base64-only docs rather than the url capability.
// Missing or unverifiable prices remain unconfigured.
const MODELS = {
  "lec-ac-image-2-5-flare": {
    kind: "image",
    name: "Image 2.5 Flare（AC）",
    icon: "OpenAI",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 32000 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K"] },
      output_format: { type: "enum", default: "jpeg", values: ["jpeg", "png"] },
      response_format: { type: "enum", default: "url", values: ["url", "b64_json"] },
      images: {
        type: "array",
        constraints: {
          max_items: 9,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["image/png", "image/jpeg", "image/webp"],
          max_bytes: 52428799,
          allow_files: true,
        },
      },
    },
    billing: { billing_unit: "request", resolution_prices: { "1K": 0.05, "2K": 0.1, "4K": 0.1 }, price_per_request: 0.05 },
  },
  "lec-ac-image-2-5-sunburst": {
    kind: "image",
    name: "Image 2.5 Sunburst（AC）",
    icon: "OpenAI",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 32000 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K"] },
      output_format: { type: "enum", default: "jpeg", values: ["jpeg", "png"] },
      response_format: { type: "enum", default: "url", values: ["url", "b64_json"] },
      images: {
        type: "array",
        constraints: {
          max_items: 9,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["image/png", "image/jpeg", "image/webp"],
          max_bytes: 52428799,
          allow_files: true,
        },
      },
    },
    billing: { billing_unit: "request", resolution_prices: { "1K": 0.05, "2K": 0.1, "4K": 0.1 }, price_per_request: 0.05 },
  },
  "lec-ac-seedance-2-0-fast-2-720p": {
    kind: "video",
    name: "Seedance 2.0 Fast 933（AC）",
    icon: "Doubao.Color",
    tags: ["fast", "933", "720p", "image", "video", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "string", default: "16:9" },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      duration: { type: "integer", default: 15, values: [15], min: 1, max: 3600 },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request" },
  },
  "lec-h3video-2k": {
    kind: "video",
    name: "MiniMax-H3-2K",
    icon: "Minimax.Color",
    tags: ["full", "1440p", "image", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"], allow_data_url: true, allow_files: true } },
      duration: { type: "integer", default: 15, values: [15], min: 1, max: 3600 },
      resolution: { type: "enum", default: "1440p", values: ["1440p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "1440p": 2.5 }, price_per_request: 2.5 },
  },
  "lec-ac-image-2": {
    kind: "image",
    name: "Image 2（AC）",
    icon: "OpenAI",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "enum", default: "16:9", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K"] },
      output_format: { type: "enum", default: "jpeg", values: ["jpeg", "png"] },
      images: {
        type: "array",
        constraints: {
          max_items: 9,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["image/png", "image/jpeg", "image/webp"],
          max_bytes: 52428799,
          allow_files: true,
        },
      },
    },
    billing: { billing_unit: "request", resolution_prices: { "1K": 0.08, "2K": 0.08, "4K": 0.08 }, price_per_request: 0.08 },
  },
  "lec-aj-wan-3-0": {
    kind: "video",
    name: "Wan 3.0（AJ）",
    icon: "AlibabaCloud.Color",
    tags: ["fast", "multi", "image", "video", "audio", "per_second"],
    parameters: {
      prompt: { type: "string" },
      duration: { type: "integer", default: 5, min: 2, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "adaptive", values: ["adaptive", "16:9", "4:3", "1:1", "3:4", "9:16"] },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p", "1080p"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "1080p": 0.61, "480p": 0.18, "720p": 0.32 }, equivalent_second_price: 0.32 },
  },
  "lec-aj-wan-3-0-prime": {
    kind: "video",
    name: "Wan 3.0 Prime（AJ）",
    icon: "AlibabaCloud.Color",
    tags: ["full", "multi", "image", "video", "audio", "per_second"],
    parameters: {
      prompt: { type: "string" },
      duration: { type: "integer", default: 5, min: 2, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "adaptive", values: ["adaptive", "16:9", "4:3", "1:1", "3:4", "9:16"] },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p", "1080p"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "1080p": 0.98, "480p": 0.25, "720p": 0.5 }, equivalent_second_price: 0.5 },
  },
  "lec-am-wan-3-0": {
    kind: "video",
    name: "Wan 3.0（AM）",
    icon: "AlibabaCloud.Color",
    tags: ["1080p", "image", "video", "audio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 30, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["adaptive", "16:9", "9:16", "1:1", "3:4", "4:3"] },
      audio: { type: "boolean", default: true },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
      documents: { type: "array", constraints: { max_items: 1, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "1080p", values: ["1080p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "1080p": 2 }, price_per_request: 2 },
    max_reference_total: 20,
  },
  "lec-ap-muse-video-720-10s": {
    kind: "video",
    name: "Muse Video（AP）",
    icon: "Video",
    tags: ["per_second", "720p", "image"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 10000 },
      duration: { type: "integer", required: true, default: 5, values: [5, 10, 20, 30, 40, 50, 60], min: 5, max: 60 },
      images: { type: "array", constraints: { max_items: 4, url_schemes: ["https"] } },
      first_frame: { type: "string", constraints: { url_schemes: ["https"] } },
      last_frame: { type: "string", constraints: { url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 0, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 0, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "second", resolution_prices: { "720p": 0.05 }, equivalent_second_price: 0.05 },
  },
  "lec-ap-seedance-2-0": {
    kind: "video",
    name: "Seedance 2.0（AP）",
    icon: "Doubao.Color",
    tags: ["per_request", "480p", "image", "audio", "supported"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 5, min: 5, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "480p", values: ["480p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "480p": 1.3 }, price_per_request: 1.3 },
  },
  "lec-ap-seedance-2-5-yz": {
    kind: "video",
    name: "Seedance 2.5（AP）",
    icon: "Doubao.Color",
    tags: ["per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 5, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "4:5", "5:4", "21:9", "9:21"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p", "1080p"] },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "1080p": 1.75, "480p": 0.32, "720p": 0.45 }, equivalent_second_price: 0.32 },
  },
  "lec-haya-seedance-2-0-vc-720p": {
    kind: "video",
    name: "Seedance 2.0 VC（HAYA）",
    icon: "Doubao.Color",
    tags: ["full", "720p", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 10000 },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "4:3", "3:4", "1:1"] },
      images: {
        type: "array",
        constraints: {
          max_items: 9,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["image/png", "image/jpeg", "image/webp", "image/gif"],
          max_bytes: 20971520,
          allow_files: true,
        },
      },
      videos: {
        type: "array",
        constraints: {
          max_items: 3,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["video/mp4", "video/webm", "video/quicktime"],
          max_bytes: 20971520,
          allow_files: true,
        },
      },
      audios: {
        type: "array",
        constraints: {
          max_items: 3,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["audio/aac", "audio/flac", "audio/mp4", "audio/mpeg", "audio/ogg", "audio/wav", "audio/x-wav"],
          max_bytes: 20971520,
          allow_files: true,
        },
      },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 2.5 }, price_per_request: 2.5 },
  },
  "lec-haya-seedance-2-0-xg-720": {
    kind: "video",
    name: "Seedance 2.0 满血 XG（HAYA）",
    icon: "Doubao.Color",
    tags: ["full", "XG", "720p", "image", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 15, step: 1 },
      aspect_ratio: { type: "string", default: "16:9" },
      images: {
        type: "array",
        constraints: {
          max_items: 9,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["image/png", "image/jpeg", "image/webp", "image/gif"],
          max_bytes: 20971520,
          allow_files: true,
        },
      },
      audios: {
        type: "array",
        constraints: {
          max_items: 3,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["audio/aac", "audio/flac", "audio/mp4", "audio/mpeg", "audio/ogg", "audio/wav", "audio/x-wav"],
          max_bytes: 20971520,
          allow_files: true,
        },
      },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 3.5 }, price_per_request: 3.5 },
  },
  "lec-jl-wan-3-0-1055-720p": {
    kind: "video",
    name: "Wan 3.0 1055（JL）",
    icon: "AlibabaCloud.Color",
    tags: ["full", "1055", "720p", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"] } },
      videos: { type: "array", constraints: { max_items: 5, url_schemes: ["http", "https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["http", "https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 4 }, price_per_request: 4 },
  },
  "lec-jm-seedance-2-5": {
    kind: "video",
    name: "Seedance 2.5（JM）",
    icon: "Doubao.Color",
    tags: ["per_second", "supported", "image", "audio"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 4, max: 30, step: 1 },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["21:9", "16:9", "4:3", "1:1", "3:4", "9:16"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "480p": 0.22, "720p": 0.3 }, equivalent_second_price: 0.3 },
  },
  "lec-mf-gemini-omni-1-2-flash": {
    kind: "video",
    name: "Gemini Omni 1.2 Flash（MF）",
    icon: "Gemini.Color",
    tags: ["flash", "per_request", "multi", "image", "video"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 5, min: 3, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      resolution: { type: "enum", default: "720p", values: ["360p", "720p", "1080p", "4k"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["http", "https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "1080p": 1.5, "360p": 1.5, "4k": 1.5, "720p": 1.5 }, price_per_request: 1.5 },
  },
  "lec-mf-seedance-2-5-30ref": {
    kind: "video",
    name: "Seedance 2.5 30参（MF）",
    icon: "Doubao.Color",
    tags: ["30-ref", "per_request", "720p", "image", "video", "audio", "native"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 5.5 }, price_per_request: 5 },
    max_reference_total: 30,
    max_reference_video_seconds: 15,
  },
  "lec-mf-video-upscale": {
    kind: "video",
    name: "火山-视频超分（MF）",
    icon: "Doubao.Color",
    tags: ["per_second", "multi", "video"],
    parameters: {
      resolution: { type: "enum", default: "720p", values: ["720p", "1080p", "2k", "4k"] },
      fps: { type: "enum", default: 30, values: [30, 60, 120, 240] },
      videos: { type: "array", required: true, constraints: { min_items: 1, max_items: 1, url_schemes: ["http", "https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "1080p": 0.06, "2k": 0.14, "4k": 0.22, "720p": 0.04 }, equivalent_second_price: 0.04 },
    source_duration: true,
  },
  "lec-mg-seedance-2-5-6": {
    kind: "video",
    name: "Seedance 2.5 六号（MG）",
    icon: "Doubao.Color",
    tags: ["multi", "image", "video", "audio", "per_second"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 20000 },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 30, step: 1 },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["21:9", "16:9", "4:3", "1:1", "3:4", "9:16"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "480p": 0.3, "720p": 0.37 }, equivalent_second_price: 0.37 },
  },
  "lec-mm-seedance-2-5": {
    kind: "video",
    name: "Seedance 2.5（MM）",
    icon: "Doubao.Color",
    tags: ["30-ref", "per_request", "720p", "image", "video", "audio"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["auto", "16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 4.5 }, price_per_request: 4.5 },
    max_reference_total: 50,
  },
  "lec-mm-seedance-2-5-full": {
    kind: "video",
    name: "Seedance 2.5 满血（MM）",
    icon: "Doubao.Color",
    tags: ["30-ref", "full", "per_request", "720p", "image", "video", "audio"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 5, min: 5, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["auto", "16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 4.5 }, price_per_request: 4.5 },
    max_reference_total: 50,
  },
  "lec-nw-image-2": {
    kind: "image",
    name: "Image 2（NW）",
    icon: "OpenAI",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "enum", default: "1:1", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K"] },
      quality: { type: "enum", default: "auto", values: ["auto", "low", "medium", "high"] },
      images: { type: "array", constraints: { url_schemes: ["http", "https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "1K": 0.02, "2K": 0.03, "4K": 0.05 }, price_per_request: 0.02 },
  },
  "lec-nw-image-2-5": {
    kind: "image",
    name: "Image 2.5（NW）",
    icon: "OpenAI",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "enum", default: "1:1", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K"] },
      quality: { type: "enum", default: "auto", values: ["auto", "low", "medium", "high"] },
      images: { type: "array", constraints: { url_schemes: ["http", "https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "1K": 0.02, "2K": 0.03, "4K": 0.05 }, price_per_request: 0.02 },
  },
  "lec-nw-image-2-6": {
    kind: "image",
    name: "Image 2.6（NW）",
    icon: "OpenAI",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "enum", default: "1:1", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K"] },
      quality: { type: "enum", default: "auto", values: ["auto", "low", "medium", "high"] },
      images: { type: "array", constraints: { url_schemes: ["http", "https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "1K": 0.04, "2K": 0.05, "4K": 0.07 }, price_per_request: 0.04 },
  },
  "lec-nw-nano-banana-2-1": {
    kind: "image",
    name: "Nano Banana 2.1（NW）",
    icon: "Gemini.Color",
    tags: ["image", "per_request"],
    parameters: {
      images: {
        type: "array",
        constraints: {
          max_items: 14,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["image/png", "image/jpeg", "image/webp"],
          max_bytes: 20971520,
          allow_files: true,
        },
      },
      prompt: { type: "string", required: true },
    },
    billing: { billing_unit: "request", price_per_request: 0.132, price_per_image: 0.132 },
  },
  "lec-nzt-wan-3-0-prime": {
    kind: "video",
    name: "Wan 3.0 Prime（NZT）",
    icon: "AlibabaCloud.Color",
    tags: ["multi", "per_request", "image"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 2, max: 30, step: 1 },
      resolution: { type: "enum", default: "1080p", values: ["480p", "720p", "1080p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "1080p": 2.5, "480p": 2.5, "720p": 2.5 }, price_per_request: 2.5 },
  },
  "lec-sl-midjourney": {
    kind: "image",
    name: "Midjourney（SL）",
    icon: "Midjourney",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      action: { type: "enum", default: "imagine", values: ["imagine", "edits", "blend"] },
      aspect_ratio: { type: "enum", default: "1:1", values: ["1:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "21:9"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"] } },
      version: { type: "enum", default: "8.1", values: ["8.1", "8.2"] },
      image_size: { type: "enum", default: "1", values: ["0.25", "0.5", "1", "2"] },
      style: { type: "boolean" },
      hd: { type: "boolean" },
      draft: { type: "boolean" },
      tile: { type: "boolean" },
      seed: { type: "number" },
      stylize: { type: "number" },
      chaos: { type: "number" },
      weird: { type: "number" },
      original_task_id: { type: "string" },
    },
    billing: { billing_unit: "request", action_prices: { blend: 0.48, edits: 0.48, imagine: 0.45 }, price_per_request: 0.45 },
  },
  "lec-sl-midjourney-niji": {
    kind: "image",
    name: "Midjourney Niji（SL）",
    icon: "Midjourney",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string" },
      action: { type: "enum", default: "imagine", values: ["imagine", "edits", "blend", "hd_upscale"] },
      aspect_ratio: { type: "enum", default: "1:1", values: ["1:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "21:9"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"] } },
      image_size: { type: "enum", default: "1", values: ["0.25", "0.5", "1", "2"] },
      style: { type: "boolean" },
      seed: { type: "number" },
      stylize: { type: "number" },
      chaos: { type: "number" },
      weird: { type: "number" },
      original_task_id: { type: "string" },
      index: { type: "integer", min: 0, max: 3, step: 1 },
    },
    billing: { billing_unit: "request", action_prices: { blend: 0.48, edits: 0.48, hd_upscale: 0.48, imagine: 0.45 }, price_per_request: 0.45 },
  },
  "lec-vp-seedance-2-0-903": {
    kind: "video",
    name: "Seedance 2.0 满血 903（VP）",
    icon: "Doubao.Color",
    tags: ["full", "903", "720p", "image", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "3:4", "4:3", "1:1"] },
      images: {
        type: "array",
        constraints: { max_items: 9, url_schemes: ["https"], mime_types: ["image/png", "image/jpeg", "image/webp"], max_bytes: 20971520 },
      },
      audios: {
        type: "array",
        constraints: {
          max_items: 3,
          url_schemes: ["https"],
          mime_types: ["audio/aac", "audio/flac", "audio/mp4", "audio/mpeg", "audio/ogg", "audio/wav", "audio/x-wav"],
          max_bytes: 20971520,
        },
      },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 3.5 }, price_per_request: 3.5 },
  },
  "lec-vp-seedance-2-0-933-s2": {
    kind: "video",
    name: "Seedance 2.0 满血 933（VP）",
    icon: "Doubao.Color",
    tags: ["full", "933", "720p", "image", "video", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      images: {
        type: "array",
        constraints: { max_items: 9, url_schemes: ["https"], mime_types: ["image/png", "image/jpeg", "image/webp"], max_bytes: 20971520 },
      },
      videos: {
        type: "array",
        constraints: { max_items: 3, url_schemes: ["https"], mime_types: ["video/mp4", "video/webm", "video/quicktime"], max_bytes: 20971520 },
      },
      audios: {
        type: "array",
        constraints: {
          max_items: 3,
          url_schemes: ["https"],
          mime_types: ["audio/aac", "audio/flac", "audio/mp4", "audio/mpeg", "audio/ogg", "audio/wav", "audio/x-wav"],
          max_bytes: 20971520,
        },
      },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 4.5 }, price_per_request: 4.5 },
  },
  "lec-vp-seedance-2-5-301010-49": {
    kind: "video",
    name: "Seedance 2.5 301010（VP）",
    icon: "Doubao.Color",
    tags: ["full", "301010", "720p", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1"] },
      images: {
        type: "array",
        constraints: { max_items: 30, url_schemes: ["https"], mime_types: ["image/png", "image/jpeg", "image/webp"], max_bytes: 20971520 },
      },
      videos: {
        type: "array",
        constraints: { max_items: 10, url_schemes: ["https"], mime_types: ["video/mp4", "video/webm", "video/quicktime"], max_bytes: 20971520 },
      },
      audios: {
        type: "array",
        constraints: {
          max_items: 10,
          url_schemes: ["https"],
          mime_types: ["audio/aac", "audio/flac", "audio/mp4", "audio/mpeg", "audio/ogg", "audio/wav", "audio/x-wav"],
          max_bytes: 20971520,
        },
      },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 5.5 }, price_per_request: 5.5 },
  },
  "lec-ms-minimax-h3": {
    kind: "video",
    name: "MiniMax-H3 生成（MS）",
    icon: "Minimax.Color",
    tags: ["full", "per_second"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 10000 },
      resolution: { type: "enum", default: "768P", values: ["480p", "768P", "2K"] },
      duration: { type: "integer", default: 5, min: 4, max: 30 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["http", "https"] } },
      reference_video_seconds: { type: "integer", default: 1, min: 1, max: 15 },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["http", "https"] } },
    },
    billing: {
      billing_unit: "second",
      resolution_prices: { "2K": 0.17, "480p": 0.08, "768P": 0.11 },
      equivalent_second_price: 0.11,
      duration_prices: { "480p": { short: 0.08, long: 0.14 }, "768P": { short: 0.11, long: 0.2 }, "2K": { short: 0.17 } },
      reference_prices: { "768P": 0.5, "2K": 0.8 },
      extra_image_price: 0.2,
    },
  },
  "lec-ms-minimax-h3-upscale": {
    kind: "video",
    name: "MiniMax-H3 再生成（MS）",
    icon: "Minimax.Color",
    tags: ["per_second"],
    parameters: { source_task_id: { type: "string", required: true }, resolution: { type: "enum", default: "2K", values: ["2K"] } },
    billing: { billing_unit: "second", resolution_prices: { "2K": 0.08 }, equivalent_second_price: 0.08 },
  },
  "lec-minimax-h3": {
    kind: "video",
    name: "MiniMax-H3",
    icon: "Minimax.Color",
    tags: ["full", "720p", "image", "video", "audio", "supported", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 10, min: 6, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 9, url_schemes: ["https"], allow_data_url: true, allow_files: true } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["http", "https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["http", "https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "second", resolution_prices: { "720p": 0.15 }, equivalent_second_price: 0.15 },
  },
  "lec-minimax-h3-768p": {
    kind: "video",
    name: "MiniMax-H3",
    icon: "Minimax.Color",
    tags: ["full", "768p", "image", "audio", "discount", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 5, min: 1, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "9:16", values: ["16:9", "9:16"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "768p", values: ["768p"] },
    },
    billing: { billing_unit: "second", resolution_prices: { "768p": 0.05 }, equivalent_second_price: 0.05 },
  },
  "lec-ad-minimax-h3-replace": {
    kind: "video",
    name: "MiniMax-H3 单角色替换（AD）",
    icon: "Minimax.Color",
    tags: ["full", "768p", "image", "video", "per_second"],
    parameters: {
      prompt: { type: "string" },
      subjects: { type: "array", constraints: { min_items: 1, max_items: 1 } },
      duration: { type: "integer", default: 5, min: 5, max: 25, step: 1 },
      aspect_ratio: { type: "enum", default: "9:16", values: ["16:9", "9:16"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 1, url_schemes: ["https"] } },
      videos: { type: "array", required: true, constraints: { min_items: 1, max_items: 1, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "768p", values: ["768p"] },
    },
    billing: { billing_unit: "second", resolution_prices: { "768p": 0.15 }, equivalent_second_price: 0.15 },
  },
  "lec-ad-minimax-h3-replace-multi": {
    kind: "video",
    name: "MiniMax-H3 多角色替换（AD）",
    icon: "Minimax.Color",
    tags: ["full", "768p", "image", "video", "per_second"],
    parameters: {
      prompt: { type: "string" },
      subjects: { type: "array", constraints: { min_items: 1, max_items: 9 } },
      duration: { type: "integer", default: 5, min: 5, max: 25, step: 1 },
      aspect_ratio: { type: "enum", default: "9:16", values: ["16:9", "9:16"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", required: true, constraints: { min_items: 1, max_items: 1, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "768p", values: ["768p"] },
    },
    billing: { billing_unit: "second", resolution_prices: { "768p": 0.15 }, equivalent_second_price: 0.15 },
  },
  "lec-muse-zr-700": {
    kind: "video",
    name: "Muse 真人（YJ）",
    icon: "Video",
    tags: ["full", "720p", "image", "supported", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 15, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "2:3", "3:2"] },
      images: { type: "array", constraints: { max_items: 7, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 0, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "second", resolution_prices: { "720p": 0.05 }, equivalent_second_price: 0.05 },
  },
  "lec-seedance-2-0-full-933-720p": {
    kind: "video",
    name: "Seedance 2.0 满血 933（HN）",
    icon: "Doubao.Color",
    tags: ["full", "720p", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 5.2 }, price_per_request: 5.2 },
  },
  "lec-seedance-2-0-fast-933-720p": {
    kind: "video",
    name: "Seedance 2.0 Fast 933（HN）",
    icon: "Doubao.Color",
    tags: ["fast", "720p", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 4 }, price_per_request: 4 },
  },
  "lec-vg-seedance-2-0-full-933": {
    kind: "video",
    name: "Seedance 2.0 满血 933（VG）",
    icon: "Doubao.Color",
    tags: ["full", "933", "720p", "image", "audio", "face", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      videos: { type: "array", constraints: { max_items: 0, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 2 }, price_per_request: 2 },
  },
  "lec-seedance-2-0-mini-b1-480p": {
    kind: "video",
    name: "Seedance 2.0 Mini（HN）",
    icon: "Doubao.Color",
    tags: ["mini", "480p", "image", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "480p", values: ["480p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "480p": 1.5 }, price_per_request: 1.5 },
  },
  "lec-seedance-2-0-mini-c9-933-720p": {
    kind: "video",
    name: "Seedance 2.0 Mini 933（HN）",
    icon: "Doubao.Color",
    tags: ["mini", "933", "720p", "image", "video", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 2.5 }, price_per_request: 2.5 },
  },
  "lec-vg-seedance-2-0-mini-cc": {
    kind: "video",
    name: "Seedance 2.0 Mini（VG）",
    icon: "Doubao.Color",
    tags: ["mini", "720p", "image", "audio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 5, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 1 }, price_per_request: 1 },
  },
  "lec-ty-seedance-2-0-full-933-w-720p": {
    kind: "video",
    name: "Seedance 2.0 满血 933 W（TY）",
    icon: "Doubao.Color",
    tags: ["full", "933", "720p", "image", "video", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "4:3", "3:4", "1:1"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 9, url_schemes: ["http", "https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["http", "https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["http", "https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 3 }, price_per_request: 3 },
  },
  "lec-vg-seedance-2-5-pa": {
    kind: "video",
    name: "Seedance 2.5 PA（VG）",
    icon: "Doubao.Color",
    tags: ["full", "multi", "image", "audio", "face", "supported", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 15, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
    },
    billing: { billing_unit: "second", resolution_prices: { "480p": 0.25, "720p": 0.35 }, equivalent_second_price: 0.35 },
  },
  "lec-vg-seedance-2-5-wd": {
    kind: "video",
    name: "Seedance 2.5 WD（VG）",
    icon: "Doubao.Color",
    tags: ["full", "720p", "image", "video", "audio", "face", "supported", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "second", resolution_prices: { "720p": 0.7 }, equivalent_second_price: 0.7 },
  },
  "lec-md-seedance-2-0-900-720p": {
    kind: "video",
    name: "Seedance 2.0 Mini 900（MD）",
    icon: "Video.Color",
    tags: ["full", "900", "720p", "image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 15, values: [5, 10, 15], min: 1, max: 3600 },
      aspect_ratio: { type: "enum", required: true, default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"], allow_data_url: true, allow_files: true } },
      process_face: { type: "boolean" },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 1.2 }, price_per_request: 1.2 },
  },
  "lec-mj-seedance-2-0-full-933-jd": {
    kind: "video",
    name: "Seedance 2.0 满血 933（MJ）",
    icon: "Doubao.Color",
    tags: ["full", "933", "720p", "image", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", required: true, default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 0, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 4 }, price_per_request: 4 },
  },
  "lec-mj-seedance-2-5-bd-720p": {
    kind: "video",
    name: "Seedance 2.5（MJ）",
    icon: "Doubao.Color",
    tags: ["full", "720p", "image", "video", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 30, step: 1 },
      aspect_ratio: { type: "enum", required: true, default: "16:9", values: ["16:9", "9:16", "1:1"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 5.5 }, price_per_request: 5.5 },
  },
  "lec-mj-seedance-2-5-zd": {
    kind: "video",
    name: "Seedance 2.5（MJ）",
    icon: "Doubao.Color",
    tags: ["full", "multi", "image", "video", "audio", "supported", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", required: true, default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "480p": 0.25, "720p": 0.35 }, equivalent_second_price: 0.35 },
  },
  "lec-gt-seedance-2-0-mini": {
    kind: "video",
    name: "Seedance 2.0 Mini 933（官渠）",
    icon: "Doubao.Color",
    tags: ["mini", "933", "multi", "image", "video", "audio", "face", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 5, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "480p": 0.28, "720p": 0.43 }, equivalent_second_price: 0.28 },
  },
  "lec-gt-seedance-2-0-fast": {
    kind: "video",
    name: "Seedance 2.0 Fast 933（官渠）",
    icon: "Doubao.Color",
    tags: ["fast", "933", "multi", "image", "video", "audio", "face", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 5, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "480p": 0.33, "720p": 0.55 }, equivalent_second_price: 0.33 },
  },
  "lec-gt-seedance-2-0-full": {
    kind: "video",
    name: "Seedance 2.0 满血 933（官渠）",
    icon: "Doubao.Color",
    tags: ["full", "933", "multi", "image", "video", "audio", "face", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 5, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p", "1080p"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "1080p": 1.55, "480p": 0.45, "720p": 0.75 }, equivalent_second_price: 0.45 },
  },
  "lec-gt-seedance-2-5-720p": {
    kind: "video",
    name: "Seedance 2.5 满血（官渠）",
    icon: "Doubao.Color",
    tags: ["full", "multi", "image", "video", "audio", "face", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p", "1080p"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: {
      billing_unit: "second",
      resolution_prices: { "1080p": 2.1, "480p": 0.67, "720p": 1.3 },
      video_reference_prices: { "1080p": 2.5, "480p": 0.9, "720p": 1.65 },
      equivalent_second_price: 1.3,
    },
  },
  "lec-ac-seedream-5-0-lite": {
    kind: "image",
    name: "Seedream 5.0 Lite（AC）",
    icon: "Doubao.Color",
    tags: ["lite", "image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "enum", default: "16:9", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K"] },
      output_format: { type: "enum", default: "jpeg", values: ["jpeg", "png"] },
      images: {
        type: "array",
        constraints: {
          max_items: 9,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["image/png", "image/jpeg", "image/webp"],
          max_bytes: 52428799,
          allow_files: true,
        },
      },
    },
    billing: { billing_unit: "request", resolution_prices: { "1K": 0.1, "2K": 0.1, "4K": 0.1 }, price_per_request: 0.1 },
  },
  "lec-ac-seedream-5-0-pro": {
    kind: "image",
    name: "Seedream 5.0 Pro（AC）",
    icon: "Doubao.Color",
    tags: ["pro", "image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "enum", default: "16:9", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K"] },
      output_format: { type: "enum", default: "jpeg", values: ["jpeg", "png"] },
      images: {
        type: "array",
        constraints: {
          max_items: 9,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["image/png", "image/jpeg", "image/webp"],
          max_bytes: 52428799,
          allow_files: true,
        },
      },
    },
    billing: { billing_unit: "request", resolution_prices: { "1K": 0.15, "2K": 0.15, "4K": 0.15 }, price_per_request: 0.15 },
  },
  "lec-ty-seedream-5-pro": {
    kind: "image",
    name: "Seedream 5 Pro（TY）",
    icon: "Doubao.Color",
    tags: ["pro", "image", "aspect_ratio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K"] },
      aspect_ratio: { type: "enum", default: "1:1", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      images: { type: "array", constraints: { max_items: 8, url_schemes: ["http", "https"], min_items: 0 } },
    },
    billing: { billing_unit: "request", price: 0.1 },
  },
  "lec-ty-face-processing-1-0": {
    kind: "video",
    name: "天悦人脸处理 1.0（TY）",
    icon: "Video",
    tags: ["image", "supported", "per_request"],
    parameters: {
      mode: { type: "enum", required: true, default: "黑白素描1", values: ["黑白素描1", "黑白素描2", "黑白素描3", "黑白白描", "3D建模网", "平铺网格"] },
      aspect_ratio: { type: "enum", required: true, default: "16:9", values: ["16:9", "9:16", "1:1"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 1, url_schemes: ["http", "https"] } },
      duration: { type: "integer", default: 1, values: [1], min: 1, max: 3600 },
    },
    billing: { billing_unit: "request", price_per_request: 0.55 },
  },
  "lec-tinysnow-banana-3-1": {
    kind: "image",
    name: "Nano Banana 2（TS）",
    icon: "Gemini.Color",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "enum", default: "16:9", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K"] },
      images: {
        type: "array",
        constraints: { url_schemes: ["https"], allow_data_url: true, mime_types: ["image/png", "image/jpeg", "image/webp"], allow_files: true },
      },
    },
    billing: { billing_unit: "request", price_per_request: 0.15 },
  },
  "lec-tinysnow-banana-3": {
    kind: "image",
    name: "Nano Banana Pro（TS）",
    icon: "Gemini.Color",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "enum", default: "16:9", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K"] },
      images: {
        type: "array",
        constraints: { url_schemes: ["https"], allow_data_url: true, mime_types: ["image/png", "image/jpeg", "image/webp"], allow_files: true },
      },
    },
    billing: { billing_unit: "request", price_per_request: 0.2 },
  },
  "lec-tinysnow-image-2": {
    kind: "image",
    name: "Image 2（TS）",
    icon: "OpenAI",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      images: { type: "array", constraints: { url_schemes: ["https"], allow_data_url: true, allow_files: true } },
      n: { type: "integer", default: 1, min: 1, max: 10 },
      aspect_ratio: { type: "enum", default: "1:1", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K", "auto"] },
      quality: { type: "enum", default: "auto", values: ["auto", "low", "medium", "high"] },
      response_format: { type: "enum", default: "b64_json", values: ["b64_json"] },
    },
    billing: { billing_unit: "request", price_per_image: 0.05, price_per_request: 0.05 },
  },
  "lec-tinysnow-image-2-5-flare": {
    kind: "image",
    name: "Image 2.5 Flare（TS）",
    icon: "OpenAI",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      images: { type: "array", constraints: { url_schemes: ["https"], allow_data_url: true, allow_files: true } },
      n: { type: "integer", default: 1, min: 1, max: 10 },
      aspect_ratio: { type: "enum", default: "1:1", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K", "auto"] },
      quality: { type: "enum", default: "auto", values: ["auto", "low", "medium", "high"] },
    },
    billing: { billing_unit: "request", price_per_image: 0.05, price_per_request: 0.05 },
  },
  "lec-tinysnow-image-2-5-sunburst": {
    kind: "image",
    name: "Image 2.5 Sunburst（TS）",
    icon: "OpenAI",
    tags: ["image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      images: { type: "array", constraints: { url_schemes: ["https"], allow_data_url: true, allow_files: true } },
      n: { type: "integer", default: 1, min: 1, max: 10 },
      aspect_ratio: { type: "enum", default: "1:1", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K", "4K", "auto"] },
      quality: { type: "enum", default: "auto", values: ["auto", "low", "medium", "high"] },
    },
    billing: { billing_unit: "request", price_per_image: 0.05, price_per_request: 0.05 },
  },
  "lec-vv-seedance-2-5-r2": {
    kind: "video",
    name: "Seedance 2.5 R2（VV）",
    icon: "Doubao.Color",
    tags: ["full", "r2", "480p", "image", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 30, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      images: {
        type: "array",
        required: true,
        constraints: { min_items: 1, max_items: 30, url_schemes: ["https"], mime_types: ["image/png", "image/jpeg", "image/webp"] },
      },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "480p", values: ["480p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "480p": 5.3 }, price_per_request: 5.3 },
  },
  "lec-vv-seedance-2-5-r2-plus": {
    kind: "video",
    name: "Seedance 2.5 R2 Plus（VV）",
    icon: "Doubao.Color",
    tags: ["full", "r2-plus", "720p", "image", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 30, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      images: {
        type: "array",
        required: true,
        constraints: { min_items: 1, max_items: 30, url_schemes: ["https"], mime_types: ["image/png", "image/jpeg", "image/webp"] },
      },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 6.3 }, price_per_request: 6.3 },
  },
  "lec-wan-3-0": {
    kind: "video",
    name: "Wan 3.0（渠道1）",
    icon: "AlibabaCloud.Color",
    tags: ["full", "480p", "720p", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      images: {
        type: "array",
        constraints: { max_items: 10, url_schemes: ["https"], mime_types: ["image/png", "image/jpeg", "image/webp"], max_bytes: 15728640 },
      },
      videos: { type: "array", constraints: { max_items: 5, url_schemes: ["https"], mime_types: ["video/mp4"], max_bytes: 62914560 } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["https"], mime_types: ["audio/mpeg", "audio/wav"], max_bytes: 52428800 } },
      duration: { type: "integer", default: 30, values: [30], min: 1, max: 3600 },
    },
    billing: { billing_unit: "request", resolution_prices: { "480p": 2.5, "720p": 3 }, price_per_request: 2.5 },
  },
  "lec-wan-3-0-h2": {
    kind: "video",
    name: "Wan 3.0（HN）",
    icon: "AlibabaCloud.Color",
    tags: ["full", "multi", "image", "audio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 4, min: 2, max: 20, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "480p": 2.5, "720p": 3 }, price_per_request: 3 },
  },
  "lec-image-2": {
    kind: "image",
    name: "Image 2",
    icon: "OpenAI.Color",
    tags: ["image"],
    parameters: {
      prompt: { type: "string", required: true },
      images: { type: "array", constraints: { url_schemes: ["https"] } },
      n: { type: "integer", default: 1, min: 1, max: 10 },
      aspect_ratio: { type: "enum", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"], default: "1:1" },
      resolution: { type: "enum", values: ["1K", "2K", "4K"], default: "1K" },
      quality: { type: "enum", values: ["auto", "low", "medium", "high"], default: "auto" },
      response_format: { type: "enum", values: ["url"], default: "url" },
    },
    billing: { billing_unit: "request", price_per_request: 0.1 },
  },
  image2: {
    kind: "image",
    name: "Image generation",
    icon: "OpenAI.Color",
    tags: ["image"],
    parameters: {
      prompt: { type: "string", required: true },
      images: { type: "array", constraints: { url_schemes: ["https"] } },
      size: { type: "string", default: "1024x1024" },
      resolution: { type: "enum", values: ["1K", "2K", "4K"] },
      quality: { type: "enum", values: ["standard", "thinking"], default: "standard" },
      response_format: { type: "enum", values: ["url"], default: "url" },
    },
    billing: {},
  },
  "nano-banana-2": {
    kind: "image",
    name: "Image generation Pro",
    icon: "Gemini.Color",
    tags: ["image"],
    parameters: {
      prompt: { type: "string", required: true },
      images: { type: "array", constraints: { url_schemes: ["https"] } },
      aspect_ratio: { type: "enum", values: ["16:9", "9:16", "1:1", "2:3", "3:2"] },
      size: { type: "string" },
      resolution: { type: "enum", values: ["1K", "2K", "4K"] },
      response_format: { type: "enum", values: ["url"], default: "url" },
    },
    billing: {},
  },
  "lec-vp-seedance-2-9-301010": {
    kind: "video",
    name: "Seedance 2.9 301010 (VP)",
    icon: "Doubao.Color",
    tags: ["video"],
    parameters: { prompt: { type: "string", required: true } },
    billing: {},
    unverified: true,
  },
  "lec-omni-video-redraw": {
    kind: "video",
    name: "Omni Video Redraw",
    icon: "Gemini.Color",
    tags: ["video_redraw"],
    parameters: {
      prompt: { type: "string", required: true },
      size: { type: "enum", values: ["1280x720", "720x1280", "1024x1024"], default: "1280x720" },
      images: { type: "array", constraints: { url_schemes: ["https"], max_items: 3 } },
      videos: { type: "array", constraints: { url_schemes: ["https"], max_items: 1, max_bytes: 5242880 } },
    },
    billing: { billing_unit: "request", price_per_request: 1.5 },
  },
};

// Public identities stay independent from LecAPI's exact wire model names.
// Reject a collision instead of silently dropping one of the catalog models.
const MODEL_MAPPING = {};
const PUBLIC_MODEL_NAMES = {};
const MODEL_LOOKUP = {};
for (const vendorModel of Object.keys(MODELS)) {
  // Reuse the exact spelling already shared by Hailuo's host binding.
  const publicModel = vendorModel === "lec-minimax-h3" ? "MiniMax-H3" : vendorModel.startsWith("lec-") ? vendorModel.slice(4) : vendorModel;
  if (Object.prototype.hasOwnProperty.call(MODEL_MAPPING, publicModel)) throw new Error("LecAPI public model id collision: " + publicModel);
  MODEL_MAPPING[publicModel] = vendorModel;
  PUBLIC_MODEL_NAMES[vendorModel] = publicModel;
  for (const name of [publicModel, vendorModel]) {
    const lookup = name.toLowerCase();
    if (Object.prototype.hasOwnProperty.call(MODEL_LOOKUP, lookup) && MODEL_LOOKUP[lookup] !== vendorModel)
      throw new Error("LecAPI public model id collision: " + publicModel);
    MODEL_LOOKUP[lookup] = vendorModel;
  }
}
const IMAGE_NAMES = Object.keys(MODEL_MAPPING).filter((name) => MODELS[MODEL_MAPPING[name]].kind === "image");
const VIDEO_NAMES = Object.keys(MODEL_MAPPING).filter((name) => MODELS[MODEL_MAPPING[name]].kind === "video");
const ASYNC_IMAGE_MODELS = ["lec-sl-midjourney", "lec-sl-midjourney-niji", "lec-ty-seedream-5-pro"];
const ASYNC_IMAGE_NAMES = ASYNC_IMAGE_MODELS.map((name) => PUBLIC_MODEL_NAMES[name]);
const SYNC_IMAGE_NAMES = IMAGE_NAMES.filter((name) => !ASYNC_IMAGE_NAMES.includes(name));
const CONTAINERS = ["metadata", "extra", "parameters", "values"];
const MEDIA_FIELDS = ["images", "videos", "audios", "documents", "first_frame", "last_frame"];
const QUANTITIES = ["n", "count", "batch_size", "batch_count", "num_images", "num_videos"];
const RESOLUTIONS = ["unspecified", ...new Set(Object.values(MODELS).flatMap((model) => model.parameters.resolution?.values || []))];
const VIDEO_USAGE_SCHEMA = {
  requests: { type: "number", unit: "count", description: { en: "Video generation request unit price", zh: "视频生成按次单价" } },
  seconds: { type: "number", unit: "second", description: { en: "Video generation unit price", zh: "视频生成单价" } },
  reference_seconds: { type: "number", unit: "second", optional: true, description: { en: "Reference video unit price", zh: "参考视频单价" } },
  extra_images: {
    type: "number",
    unit: "count",
    optional: true,
    unitLabel: { en: "image", zh: "张" },
    description: { en: "Additional reference image unit price", zh: "额外参考图单价" },
  },
  resolution: { enum: RESOLUTIONS, description: { en: "Output video resolution", zh: "输出视频分辨率" } },
  duration_tier: {
    enum: ["short", "long"],
    description: { en: "Output duration tier", zh: "输出时长档位" },
    enumLabels: { short: { en: "Up to 15 seconds", zh: "不超过15秒" }, long: { en: "Over 15 seconds", zh: "超过15秒" } },
  },
  video_input: {
    enum: ["none", "video"],
    description: { en: "Reference video input", zh: "参考视频输入" },
    enumLabels: { none: { en: "No reference video", zh: "无参考视频" }, video: { en: "With reference video", zh: "有参考视频" } },
  },
};
const IMAGE_USAGE_SCHEMA = {
  requests: { type: "number", unit: "count", description: { en: "Image generation request unit price", zh: "图片生成按次单价" } },
  image_count: { type: "number", unit: "count", unitLabel: { en: "image", zh: "张" }, description: { en: "Image generation unit price", zh: "图片生成单价" } },
  resolution: { enum: RESOLUTIONS, description: { en: "Output image resolution", zh: "输出图片分辨率" } },
  action: {
    enum: ["imagine", "edits", "blend", "hd_upscale"],
    description: { en: "Generate, edit, blend or upscale images", zh: "生成、编辑、融合或放大图片" },
  },
};

function defaultExpression(profile) {
  const billing = profile.billing;
  const quantity = billing.price_per_image !== undefined ? "image_count" : billing.billing_unit === "second" ? "seconds" : "requests";
  const priceWithFee = (price) => Number((price * 1.2).toFixed(12));
  const leaf = (name, price) => "tier(" + JSON.stringify(name) + ", u(" + JSON.stringify(quantity) + ") * " + priceWithFee(price) + ")";
  if (billing.duration_prices) {
    const branches = [];
    for (const resolution of Object.keys(billing.duration_prices)) {
      for (const band of Object.keys(billing.duration_prices[resolution])) {
        const reference = billing.reference_prices[resolution] || 0;
        branches.push(
          'u("resolution") == ' +
            JSON.stringify(resolution) +
            ' && u("duration_tier") == ' +
            JSON.stringify(band) +
            " ? tier(" +
            JSON.stringify(resolution + "-" + band) +
            ', u("seconds") * ' +
            priceWithFee(billing.duration_prices[resolution][band]) +
            ' + u("reference_seconds") * ' +
            priceWithFee(reference) +
            ' + u("extra_images") * ' +
            priceWithFee(billing.extra_image_price) +
            ")"
        );
      }
    }
    // 2K's final supported row is the fallback; request validation rejects its
    // unsupported long-duration combination before any reservation or submit.
    const final = branches.pop();
    branches.push(final.slice(final.indexOf(" ? ") + 3));
    return branches.join(" : ");
  }
  if (billing.action_prices) {
    const actions = Object.keys(billing.action_prices);
    return actions
      .map(
        (action, index) =>
          (index === actions.length - 1 ? "" : 'u("action") == ' + JSON.stringify(action) + " ? ") + leaf(action, billing.action_prices[action])
      )
      .join(" : ");
  }
  if (billing.resolution_prices) {
    const resolutions = Object.keys(billing.resolution_prices).filter(
      (value) => !profile.parameters.resolution?.values || profile.parameters.resolution.values.includes(value)
    );
    if (billing.video_reference_prices) {
      const branches = [];
      for (let index = 0; index < resolutions.length; index++) {
        const resolution = resolutions[index],
          last = index === resolutions.length - 1;
        const condition = 'u("resolution") == ' + JSON.stringify(resolution);
        branches.push(
          (last ? "" : condition + " && ") + 'u("video_input") == "video" ? ' + leaf(resolution + "-reference", billing.video_reference_prices[resolution])
        );
        branches.push((last ? "" : condition + " ? ") + leaf(resolution, billing.resolution_prices[resolution]));
      }
      return branches.join(" : ");
    }
    return resolutions
      .map((resolution, index) => {
        const expression = leaf(resolution, billing.resolution_prices[resolution]);
        return (index === resolutions.length - 1 ? "" : 'u("resolution") == ' + JSON.stringify(resolution) + " ? ") + "(" + expression + ")";
      })
      .join(" : ");
  }
  const price = billing.price_per_image ?? billing.price_per_request ?? billing.equivalent_second_price ?? billing.price;
  return price === undefined ? "" : leaf(quantity, price);
}

function modelDescription(profile) {
  const english = [],
    chinese = [];
  for (const [field, parameter] of Object.entries(profile.parameters)) {
    let details =
      parameter.values?.join(" / ") ||
      (parameter.min !== undefined || parameter.max !== undefined ? String(parameter.min ?? 0) + "–" + String(parameter.max ?? "") : parameter.type);
    if (parameter.default !== undefined) details += " [" + parameter.default + "]";
    if (parameter.required) details += " *";
    const constraints = parameter.constraints;
    if (constraints) {
      if (constraints.max_items !== undefined) details += "; " + (constraints.min_items || 0) + "–" + constraints.max_items;
      if (constraints.url_schemes) details += "; " + constraints.url_schemes.join("/").toUpperCase();
      if (constraints.allow_data_url) details += "; data URL";
      if (constraints.allow_files) details += "; multipart";
      if (constraints.mime_types) details += "; " + constraints.mime_types.join(", ");
      if (constraints.max_bytes) details += "; ≤" + constraints.max_bytes + " bytes";
    }
    if (parameter.maxLength) details += "; ≤" + parameter.maxLength + " characters";
    english.push(field + ": " + details);
    chinese.push(field + ": " + details.replace("characters", "字").replace("bytes", "字节"));
  }
  const suffix = profile.unverified
    ? "; This model is listed by /v1/models but has no published parameter or pricing card; only a prompt can be submitted"
    : "; * required; brackets show defaults; references retain their order";
  const suffixZH = profile.unverified
    ? "；该模型在 /v1/models 中可见，但没有公开参数或价格卡片，当前仅接受提示词"
    : "；* 为必填，中括号为默认值，参考素材保持顺序";
  if (profile.max_reference_total) {
    english.push("Combined references ≤" + profile.max_reference_total);
    chinese.push("参考素材合计≤" + profile.max_reference_total);
  }
  if (profile.max_reference_video_seconds) {
    english.push("Each reference video ≤" + profile.max_reference_video_seconds + "s");
    chinese.push("单段参考视频≤" + profile.max_reference_video_seconds + "秒");
  }
  return { en: english.join("; ") + suffix, zh: chinese.join("；") + suffixZH };
}

const MODEL_DEFAULTS = Object.fromEntries(
  Object.entries(MODELS).map(([name, profile]) => {
    const defaults = { displayName: profile.name, description: modelDescription(profile), icon: profile.icon, tags: [...new Set(profile.tags)] };
    const expression = defaultExpression(profile);
    if (expression) defaults.billingExpr = expression;
    return [PUBLIC_MODEL_NAMES[name], defaults];
  })
);

export const meta = {
  apiVersion: 1,
  key: "lecapi",
  name: "LecAPI",
  icon: "text:LE",
  version: "1.0.0",
  author: { name: "QuantumNous" },
  website: "https://api.lecapi.net/docs",
  baseUrl: "https://api.lecapi.net",
  description: {
    en: "LecAPI image and video generation with model parameters, reference materials and published default pricing",
    zh: "LecAPI 图片与视频生成，包含模型参数、参考素材要求及公开默认定价",
  },
  requiredCapabilities: ["submit-no-retry@1", "media-duration@1"],
  upstreams: ["vendor", "new_api"],
  fetchMode: "per_task",
  models: Object.keys(MODEL_MAPPING),
  modelDefaults: MODEL_DEFAULTS,
  usageSchema: VIDEO_USAGE_SCHEMA,
  usageProfiles: [{ models: IMAGE_NAMES, schema: IMAGE_USAGE_SCHEMA }],
  protocols: [
    { name: "openai_video", models: VIDEO_NAMES },
    { name: "openai_image", models: IMAGE_NAMES },
  ],
  routes: [
    { method: "POST", path: "/lecapi/v1/videos", type: "submit", decode: "createVideo", render: "task" },
    { method: "GET", path: "/lecapi/v1/videos/:task_id", type: "query", render: "task" },
    {
      method: "POST",
      path: "/lecapi/v1/images/generations",
      type: "submit",
      decode: "createImage",
      render: "task",
      models: SYNC_IMAGE_NAMES,
      retainResult: false,
    },
    { method: "POST", path: "/lecapi/v1/images/tasks", type: "submit", decode: "createImage", render: "task", models: ASYNC_IMAGE_NAMES },
    { method: "GET", path: "/lecapi/v1/images/:task_id", type: "query", render: "task" },
  ],
};

function modelProfile(ctx) {
  const vendorModel = vendorModelName(ctx.upstreamModel || ctx.model);
  if (!vendorModel) throw new Error("unsupported LecAPI model");
  return { ...MODELS[vendorModel], vendorModel: vendorModel, publicModel: PUBLIC_MODEL_NAMES[vendorModel] };
}

function vendorModelName(name) {
  if (typeof name !== "string") return "";
  const lookup = name.toLowerCase();
  return Object.prototype.hasOwnProperty.call(MODEL_LOOKUP, lookup) ? MODEL_LOOKUP[lookup] : "";
}

function apiRoot(ctx) {
  return ctx.baseUrl.replace(/\/+$/, "") + (ctx.upstream?.kind === "new_api" ? "/lecapi" : "");
}

function parameterScopes(input) {
  if (!input || typeof input !== "object" || Array.isArray(input)) throw new Error("request body must be an object");
  const scopes = [input];
  for (let index = 0; index < scopes.length; index++) {
    for (const container of CONTAINERS) {
      if (scopes[index][container] === undefined) continue;
      const nested = scopes[index][container];
      if (!nested || typeof nested !== "object" || Array.isArray(nested)) throw new Error(container + " must be an object");
      scopes.push(nested);
      if (scopes.length > 64) throw new Error("too many parameter containers");
    }
  }
  return scopes;
}

function parameterValue(value, parameter, field) {
  if (
    parameter.type === "enum" &&
    parameter.values?.every((candidate) => typeof candidate === "number") &&
    typeof value === "string" &&
    /^\d+$/.test(value.trim())
  )
    value = Number(value);
  if (parameter.type === "integer" || parameter.type === "number") {
    if (typeof value !== "number" && (typeof value !== "string" || !/^-?\d+(?:\.\d+)?$/.test(value.trim()))) throw new Error(field + " must be a number");
    value = Number(value);
    if (!Number.isFinite(value) || Math.abs(value) > Number.MAX_SAFE_INTEGER || (parameter.type === "integer" && !Number.isSafeInteger(value)))
      throw new Error(field + " must be a finite " + parameter.type);
    if ((parameter.min !== undefined && value < parameter.min) || (parameter.max !== undefined && value > parameter.max))
      throw new Error(field + " is outside the supported range");
  } else if (parameter.type === "boolean") {
    if (value === "true") value = true;
    if (value === "false") value = false;
    if (typeof value !== "boolean") throw new Error(field + " must be a boolean");
  } else if (parameter.type === "array") {
    if (!Array.isArray(value)) value = [value];
    value = value.slice();
    const constraints = parameter.constraints || {};
    if (value.length < (constraints.min_items || 0) || value.length > (constraints.max_items ?? 128))
      throw new Error(field + " has an unsupported number of items");
    return value;
  } else {
    if (typeof value !== "string") {
      if (parameter.type === "enum" && parameter.values?.some((candidate) => typeof candidate === "number")) value = Number(value);
      else throw new Error(field + " must be a string");
    } else value = value.trim();
    if (
      value === "" ||
      (typeof value === "string" &&
        Array.from(value).some((character) => {
          const code = character.charCodeAt(0);
          return code < 9 || code === 11 || code === 12 || (code >= 14 && code < 32) || code === 127;
        }))
    )
      throw new Error(field + " must not be empty or contain control characters");
    if (parameter.maxLength && Array.from(value).length > parameter.maxLength) throw new Error(field + " is too long");
  }
  if (parameter.values && !parameter.values.includes(value)) {
    const equivalent =
      typeof value === "string"
        ? parameter.values.find((candidate) => typeof candidate === "string" && candidate.toLowerCase() === value.toLowerCase())
        : undefined;
    if (equivalent === undefined) throw new Error(field + " must be one of " + parameter.values.join(", "));
    value = equivalent;
  }
  return value;
}

function materialValue(value, parameter, field) {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    value = value.url || value.image_url || value.video_url || value.audio_url;
    if (value && typeof value === "object") value = value.url;
  }
  if (typeof value !== "string" || !value.trim()) throw new Error(field + " must contain media URLs");
  value = value.trim();
  const constraints = parameter.constraints || {};
  if (value.startsWith("data:")) {
    if (!constraints.allow_data_url) throw new Error(field + " does not support data URLs");
    const match = /^data:([^;,]+);base64,([A-Za-z0-9+/]*={0,2})$/.exec(value);
    if (!match || !match[2] || (constraints.mime_types && !constraints.mime_types.includes(match[1]))) throw new Error(field + " has an unsupported data URL");
    const bytes = Math.floor((match[2].length * 3) / 4) - (match[2].endsWith("==") ? 2 : match[2].endsWith("=") ? 1 : 0);
    if (bytes > (constraints.max_bytes ?? 67108864)) throw new Error(field + " exceeds the media size limit");
    return value;
  }
  const url = /^(https?):\/\/([^/?#\s]+)(?:[^\s]*)$/i.exec(value);
  if (!url || !(constraints.url_schemes || ["https"]).includes(url[1].toLowerCase()) || url[2].includes("@"))
    throw new Error(field + " requires public " + (constraints.url_schemes || ["https"]).join("/").toUpperCase() + " URLs");
  return value;
}

// Building and reservation use this same final body. Nested quantities cannot
// bypass validation, and channel mappings select the actual upstream rules.
function submission(ctx) {
  const profile = modelProfile(ctx),
    body = { model: ctx.upstream?.kind === "new_api" ? profile.publicModel : profile.vendorModel };
  const media = {},
    files = [];
  let requestedCount;
  const aliases = {
    seconds: "duration",
    resolution_name: "resolution",
    ratio: "aspect_ratio",
    image: "images",
    image_url: "images",
    image_urls: "images",
    input_reference: "images",
    video: "videos",
    video_url: "videos",
    video_urls: "videos",
    audio_url: "audios",
    audio_urls: "audios",
    first_frame_url: "first_frame",
    last_frame_url: "last_frame",
  };
  for (const scope of parameterScopes(ctx.requestBody)) {
    for (const field of Object.keys(scope)) {
      const value = scope[field];
      if (CONTAINERS.includes(field)) continue;
      if (field === "model") {
        if (
          typeof value !== "string" ||
          ![ctx.model, ctx.upstreamModel, profile.vendorModel, profile.publicModel].some(
            (name) => typeof name === "string" && name.toLowerCase() === value.toLowerCase()
          )
        )
          throw new Error("model must match the selected channel model");
        continue;
      }
      if (QUANTITIES.includes(field)) {
        const quantity = parameterValue(value, { type: "integer", min: 1, max: profile.kind === "image" ? profile.parameters.n?.max || 1 : 1 }, field);
        if (profile.kind === "video") continue;
        if (requestedCount !== undefined && requestedCount !== quantity) throw new Error("conflicting image count parameters");
        requestedCount = quantity;
        if (profile.parameters.n) body.n = quantity;
        continue;
      }
      let target = field === "audio" && !profile.parameters.audio ? "audios" : aliases[field] || field;
      if (["first_frame", "last_frame"].includes(target) && !profile.parameters[target]) target = "images";
      if (target === "size" && !profile.parameters.size) {
        if (typeof value === "string" && /^\d+:\d+$/.test(value)) target = "aspect_ratio";
        else if (typeof value === "string" && /^(?:\d+p|[124]k)$/i.test(value)) target = "resolution";
      }
      const parameter = profile.parameters[target];
      if (!parameter) throw new Error(field + " is not supported by " + profile.publicModel);
      if (MEDIA_FIELDS.includes(target)) {
        const items = parameter.type === "array" ? (Array.isArray(value) ? value : [value]) : [value];
        for (const item of items) (media[target] || (media[target] = [])).push(materialValue(item, parameter, target));
      } else {
        const normalized = parameterValue(value, parameter, target);
        if (body[target] !== undefined && JSON.stringify(body[target]) !== JSON.stringify(normalized)) throw new Error("conflicting " + target + " parameters");
        body[target] = normalized;
      }
    }
  }
  for (const file of ctx.files || []) {
    const original = file.field.replace(/\[\]$/, "");
    const target = aliases[original] || original;
    const parameter = profile.parameters[target],
      constraints = parameter?.constraints || {};
    if (!MEDIA_FIELDS.includes(target) || !parameter || !constraints.allow_files)
      throw new Error("multipart files are not supported for " + file.field + " on " + profile.publicModel);
    if (!Number.isSafeInteger(file.size) || file.size <= 0 || file.size > (constraints.max_bytes ?? 67108864))
      throw new Error(file.field + " exceeds the media size limit");
    if (constraints.mime_types && !constraints.mime_types.includes(file.mimeType)) throw new Error(file.field + " has an unsupported media type");
    const reference = { __fileRef: file.ref, encoding: "dataUrl", maxBytes: constraints.max_bytes || 67108864 };
    (media[target] || (media[target] = [])).push(reference);
    files.push({ field: target, ref: file.ref });
  }
  for (const [field, parameter] of Object.entries(profile.parameters)) {
    if (MEDIA_FIELDS.includes(field)) {
      if (media[field]?.length)
        body[field] =
          parameter.type === "array"
            ? parameterValue(media[field], parameter, field)
            : media[field].length === 1
              ? media[field][0]
              : parameterValue(media[field], { type: "array", constraints: { max_items: 1 } }, field)[0];
      else if (parameter.required || parameter.constraints?.min_items) throw new Error(field + " is required");
    } else if (body[field] === undefined) {
      if (parameter.default !== undefined) body[field] = parameterValue(parameter.default, parameter, field);
      else if (parameter.required) throw new Error(field + " is required");
    }
  }
  const imageCount = media.images?.length || 0,
    videoCount = media.videos?.length || 0,
    audioCount = media.audios?.length || 0;
  if (profile.max_reference_total && imageCount + videoCount + audioCount + (media.documents?.length || 0) > profile.max_reference_total)
    throw new Error("too many combined reference materials");
  if (profile.vendorModel === "lec-mg-seedance-2-5-6" && audioCount && !imageCount) throw new Error("audio references require at least one image");
  if (body.size && !profile.parameters.size.values && !/^[1-9]\d{1,4}x[1-9]\d{1,4}$/i.test(body.size))
    throw new Error("size must contain positive pixel dimensions");
  if (
    body.subjects &&
    (body.subjects.length !== imageCount || body.subjects.some((name) => typeof name !== "string" || !/^[A-Za-z][A-Za-z ,.'-]*$/.test(name)))
  )
    throw new Error("subjects must contain one English name per reference image");
  if (["lec-sl-midjourney", "lec-sl-midjourney-niji"].includes(profile.vendorModel)) {
    if (body.action !== "hd_upscale" && !body.prompt) throw new Error("prompt is required");
    if (["edits", "blend"].includes(body.action) && !imageCount) throw new Error("images are required for " + body.action);
    if (body.draft && body.version !== "8.1") throw new Error("draft requires Midjourney version 8.1");
    if (body.action === "hd_upscale" && (body.original_task_id === undefined || body.index === undefined))
      throw new Error("hd_upscale requires original_task_id and index");
  }
  if (profile.vendorModel === "lec-ms-minimax-h3") {
    if (body.resolution === "2K" && body.duration > 15) throw new Error("2K supports at most 15 seconds");
    if (body.resolution === "480p" && videoCount) throw new Error("480p does not support reference videos");
  }
  let seconds = body.duration || 0,
    referenceSeconds = 0;
  if (profile.source_duration || (profile.vendorModel === "lec-ms-minimax-h3" && videoCount) || profile.max_reference_video_seconds) {
    for (const reference of media.videos || []) {
      const duration = typeof reference === "object" ? utils.mediaDuration({ fileRef: reference.__fileRef }) : utils.mediaDuration({ url: reference });
      if (
        !Number.isFinite(duration) ||
        duration <= 0 ||
        duration > 3600 ||
        (profile.max_reference_video_seconds && duration > profile.max_reference_video_seconds)
      )
        throw new Error("reference video duration is outside the supported range");
      referenceSeconds += duration;
    }
    referenceSeconds = Math.ceil(referenceSeconds);
    if (profile.source_duration) seconds = referenceSeconds;
    if (profile.vendorModel === "lec-ms-minimax-h3" && videoCount) {
      if (referenceSeconds < 1 || referenceSeconds > 15) throw new Error("reference videos must total at most 15 seconds");
      if (ctx.requestBody.reference_video_seconds !== undefined && body.reference_video_seconds !== referenceSeconds)
        throw new Error("reference_video_seconds must match the measured videos");
      body.reference_video_seconds = referenceSeconds;
    }
  }
  if (profile.vendorModel === "lec-ms-minimax-h3" && !videoCount) delete body.reference_video_seconds;
  for (const field of ["source_task_id", "original_task_id"]) {
    if (body[field] === undefined) continue;
    const origin = (ctx.originTasks || []).find((task) => task.taskId === body[field]);
    if (!origin || origin.status !== "SUCCESS") throw new Error(field + " must reference a completed task owned by this account");
    const source = responseObject(origin.data);
    if (field === "source_task_id") {
      if (vendorModelName(source.model) !== "lec-ms-minimax-h3" || String(source.resolution || source.size).toUpperCase() !== "768P")
        throw new Error("source_task_id must reference a completed 768P MiniMax-H3 MS generation");
      seconds = Number(source.seconds ?? source.duration);
      if (!Number.isInteger(seconds) || seconds < 4 || seconds > 15) throw new Error("source_task_id must have a duration between 4 and 15 seconds");
    } else if (vendorModelName(source.model) !== profile.vendorModel) throw new Error("original_task_id must reference the same upstream model");
    body[field] = origin.upstreamTaskId;
  }
  if (!Number.isInteger(seconds) || seconds < 0 || seconds > 3600) throw new Error("video duration must be between 0 and 3600 seconds");
  return {
    profile: profile,
    body: body,
    seconds: seconds,
    referenceSeconds: referenceSeconds,
    imageCount: imageCount,
    videoCount: videoCount,
    outputCount: requestedCount || body.n || 1,
    files: files,
  };
}

export function buildSubmitRequest(ctx) {
  const request = submission(ctx);
  const path =
    request.profile.kind === "image"
      ? ctx.upstream?.kind === "new_api" && ASYNC_IMAGE_MODELS.includes(request.profile.vendorModel)
        ? "/v1/images/tasks"
        : "/v1/images/generations"
      : "/v1/videos";
  return {
    url: apiRoot(ctx) + path,
    method: "POST",
    headers: { Authorization: "Bearer " + ctx.apiKey, "Content-Type": "application/json" },
    body: request.body,
    action: request.profile.kind === "image" ? request.body.action || "imagine" : "generate",
  };
}

export function extractUsage(ctx) {
  const request = submission(ctx);
  if (ctx.usagePurpose === "billing_ratios")
    return request.profile.kind === "image" && request.profile.billing.price_per_image !== undefined ? { image_count: request.outputCount } : { requests: 1 };
  if (request.profile.kind === "image")
    return { requests: 1, image_count: request.outputCount, resolution: request.body.resolution || "unspecified", action: request.body.action || "imagine" };
  return {
    requests: 1,
    seconds: request.seconds,
    reference_seconds: request.profile.vendorModel === "lec-ms-minimax-h3" ? request.referenceSeconds : 0,
    extra_images: request.profile.vendorModel === "lec-ms-minimax-h3" ? Math.max(request.imageCount - 5, 0) : 0,
    resolution: request.body.resolution || "unspecified",
    duration_tier: request.seconds > 15 ? "long" : "short",
    video_input: request.videoCount > 0 ? "video" : "none",
  };
}

function responseObject(value) {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("invalid LecAPI response");
  if (value.object === "lecapi.task") return responseObject(value.output || {});
  if (
    !value.status &&
    !value.id &&
    !value.task_id &&
    value.data &&
    typeof value.data === "object" &&
    !Array.isArray(value.data) &&
    !value.data.url &&
    !value.data.b64_json &&
    (value.data.status || value.data.id || value.data.task_id)
  )
    return value.data;
  return value;
}

function imageEntries(value, allowRoot) {
  const body = responseObject(value),
    entries = [];
  const data = body.data === undefined ? (body.images === undefined ? body.result?.data || body.result?.images : body.images) : body.data;
  for (const entry of Array.isArray(data) ? data : data && typeof data === "object" ? [data] : []) {
    if (typeof entry === "string" && /^https?:\/\/\S+$/.test(entry)) entries.push({ url: entry });
    else if (entry && typeof entry === "object") {
      const image = {};
      if (typeof entry.url === "string" && entry.url.trim()) image.url = entry.url;
      if (typeof entry.b64_json === "string" && entry.b64_json.trim()) image.b64_json = entry.b64_json;
      if (typeof entry.revised_prompt === "string") image.revised_prompt = entry.revised_prompt;
      if (image.url || image.b64_json) entries.push(image);
    }
  }
  if (!entries.length && (allowRoot || body.object === "image")) {
    const url = resultURL(value);
    if (url) entries.push({ url: url });
    else if (typeof body.b64_json === "string" && body.b64_json.trim()) entries.push({ b64_json: body.b64_json });
  }
  return entries;
}

function resultURL(value) {
  const body = responseObject(value);
  for (const candidate of [
    body.url,
    body.result_url,
    body.video_url,
    body.download_url,
    body.metadata?.url,
    body.result?.url,
    body.output?.url,
    body.video?.url,
  ]) {
    if (typeof candidate === "string" && /^https?:\/\/[^\s]+$/.test(candidate)) return candidate;
  }
  return "";
}

export function parseTaskResult(ctx, response) {
  const body = responseObject(response),
    id = response.object === "lecapi.task" ? response.id : body.id || body.task_id || body.metadata?.task_id;
  if (id !== undefined && id !== ctx.taskId) return { status: "UNKNOWN", reason: "LecAPI returned a different task id" };
  const rawStatus = response.object === "lecapi.task" ? response.status : body.status;
  const status = {
    queued: "QUEUED",
    submitted: "SUBMITTED",
    pending: "QUEUED",
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
  }[String(rawStatus || "").toLowerCase()];
  if (!status) return { status: "UNKNOWN", reason: "Unrecognized LecAPI task status" };
  if (status === "FAILURE" || body.success === false)
    return { status: "FAILURE", reason: response.error?.message || body.error?.message || body.fail_reason || body.message || "LecAPI generation failed" };
  if (status === "SUCCESS") {
    const images = imageEntries(response, modelProfile(ctx).kind === "image");
    if (!images.length && !resultURL(response)) return { status: "FAILURE", reason: "LecAPI completed without a deliverable" };
    return { status: "SUCCESS", progress: "100%" };
  }
  const progress = Number(String(body.progress ?? "").replace(/%$/, ""));
  return Number.isFinite(progress) && progress >= 0 ? { status: status, progress: Math.min(progress, 99) + "%" } : { status: status };
}

export function parseSubmitResponse(ctx, response) {
  const body = responseObject(response.body);
  if (body.success === false || body.error)
    return {
      taskId: ctx.publicTaskId || utils.uuid(),
      taskData: response.body,
      immediate: { status: "FAILURE", reason: body.error?.message || body.message || "LecAPI generation failed" },
    };
  const id = response.body.object === "lecapi.task" ? response.body.id : body.id || body.task_id || body.metadata?.task_id;
  const profile = modelProfile(ctx);
  const images = imageEntries(response.body, profile.kind === "image");
  const state = { kind: profile.kind, queryable: profile.kind === "video" || ASYNC_IMAGE_MODELS.includes(profile.vendorModel) };
  const rawStatus = response.body.object === "lecapi.task" ? response.body.status : body.status;
  if (["failed", "failure", "cancelled", "canceled"].includes(String(rawStatus || "").toLowerCase())) {
    const taskId = typeof id === "string" && id.trim() ? id : ctx.publicTaskId || utils.uuid();
    return { taskId: taskId, taskData: response.body, immediate: parseTaskResult({ ...ctx, taskId: taskId }, response.body), state: state };
  }
  const terminalSuccess = ["completed", "succeeded", "success"].includes(String(rawStatus || "").toLowerCase());
  if (terminalSuccess || (images.length && !rawStatus)) {
    const taskId = typeof id === "string" && id.trim() ? id : ctx.publicTaskId || utils.uuid();
    const immediate = images.length ? { status: "SUCCESS", progress: "100%" } : parseTaskResult({ ...ctx, taskId: taskId }, response.body);
    return { taskId: taskId, taskData: response.body, immediate: immediate, state: state };
  }
  if (typeof id !== "string" || !id.trim()) throw new Error("LecAPI did not return a task id or image payload");
  return { taskId: id, taskData: response.body, state: state };
}

export function buildQueryRequest(ctx) {
  // LecAPI exposes its asynchronous media task IDs through the video task API.
  // A chained gateway uses this plugin's image/video native query wrapper.
  const image = ctx.upstream?.kind === "new_api" && modelProfile(ctx).kind === "image";
  return {
    url: apiRoot(ctx) + (image ? "/v1/images/" : "/v1/videos/") + encodeURIComponent(ctx.taskId),
    method: "GET",
    headers: { Authorization: "Bearer " + ctx.apiKey },
  };
}

export function extractUsageOnComplete(ctx, result, response) {
  if (result.status !== "SUCCESS") return {};
  const body = responseObject(response),
    profile = modelProfile(ctx);
  if (profile.kind === "image") {
    const entries = imageEntries(response, true);
    const count = Math.max(entries.filter((entry) => entry.url).length, entries.filter((entry) => entry.b64_json).length);
    return count > 0 && count <= 128 ? { image_count: count } : {};
  }
  const seconds = Number(body.seconds ?? body.duration);
  // Output seconds may replace the reservation, while reference measurements
  // and extra-image charges stay frozen from the submitted body.
  if (profile.source_duration || profile.vendorModel === "lec-ms-minimax-h3-upscale") return {};
  if (!Number.isSafeInteger(seconds) || seconds <= 0 || seconds > 3600) return {};
  const parameter = profile.parameters.duration;
  if (
    !parameter ||
    seconds < (parameter.min ?? 1) ||
    seconds > (parameter.max ?? 3600) ||
    (parameter.values && !parameter.values.includes(seconds)) ||
    (profile.vendorModel === "lec-ms-minimax-h3" && body.resolution === "2K" && seconds > 15)
  )
    return {};
  return { seconds: seconds, duration_tier: seconds > 15 ? "long" : "short" };
}

export function extractUsageOnSubmit(ctx, response) {
  return extractUsageOnComplete(ctx, { status: "SUCCESS" }, response);
}

export function listArtifacts(task) {
  if (task.status !== "SUCCESS") return [];
  const body = responseObject(task.data);
  const imageTask = ["imagine", "edits", "blend", "hd_upscale"].includes(task.action) || MODELS[vendorModelName(body.model)]?.kind === "image";
  const entries = imageEntries(task.data, imageTask);
  if (entries.length)
    return entries.filter((entry) => entry.url).map((entry, index) => ({ key: "image-" + (index + 1), type: "image", mimeType: "image/png" }));
  return resultURL(task.data) ? [{ key: "video", type: "video", mimeType: "video/mp4" }] : [];
}

export function buildContentRequest(ctx) {
  const image = /^image-([1-9]\d*)$/.exec(ctx.artifactKey);
  if (!image && ctx.artifactKey !== "video") throw new Error("artifact_not_found");
  let data = ctx.data;
  if (ctx.upstreamTaskId && ctx.state?.queryable !== false) {
    const query = buildQueryRequest({ ...ctx, taskId: ctx.upstreamTaskId });
    const refreshed = utils.fetch(query);
    if (refreshed.status < 200 || refreshed.status >= 300) throw new Error("LecAPI result could not be refreshed");
    const result = parseTaskResult({ ...ctx, taskId: ctx.upstreamTaskId }, refreshed.body);
    if (result.status !== "SUCCESS") throw new Error("LecAPI result is not available");
    data = refreshed.body;
  }
  const url = image ? imageEntries(data, true).filter((entry) => entry.url)[Number(image[1]) - 1]?.url : resultURL(data);
  if (!url) throw new Error("artifact_not_found");
  return { url: url, method: "GET", credentialless: true };
}

function decodeRequest(ctx, kind, nativeRoute) {
  if (!ctx.body || !["json", "multipart"].includes(ctx.body.kind)) throw new Error("JSON or multipart body required");
  let body;
  if (ctx.body.kind === "json") {
    if (!ctx.body.value || typeof ctx.body.value !== "object" || Array.isArray(ctx.body.value)) throw new Error("JSON object required");
    body = utils.json.clone(ctx.body.value);
  } else {
    body = {};
    for (const [field, values] of Object.entries(ctx.body.fields || {})) {
      let value = values.length === 1 ? values[0] : values.slice();
      if (typeof value === "string" && /^[[{]/.test(value.trim())) {
        try {
          value = JSON.parse(value);
        } catch {
          throw new Error(field + " must contain valid JSON");
        }
      }
      body[field.replace(/\[\]$/, "")] = value;
    }
  }
  const requestedModel = ctx.model || body.model;
  const vendorModel = vendorModelName(requestedModel);
  const model = ctx.model || (vendorModel ? PUBLIC_MODEL_NAMES[vendorModel] : requestedModel);
  if (typeof model !== "string" || !model.trim()) throw new Error("model is required");
  if (vendorModel && MODELS[vendorModel].kind !== kind) throw new Error("model does not support this media endpoint");
  // Image response_format belongs to the host presenter. The vendor still
  // receives its documented default; the host converts URLs when requested.
  if (!nativeRoute && kind === "image") delete body.response_format;
  const origins = [];
  for (const scope of parameterScopes(body))
    for (const field of ["original_task_id", "source_task_id"]) {
      if (scope[field] !== undefined) {
        if (typeof scope[field] !== "string" || !/^task_[A-Za-z0-9_-]+$/.test(scope[field])) throw new Error(field + " must be a gateway task id");
        if (!origins.includes(scope[field])) origins.push(scope[field]);
      }
    }
  return { kind: "submit", model: model, requestBody: body, ...(origins.length ? { originTaskIds: origins } : {}) };
}

function nativeTask(ctx, task) {
  const status =
    { NOT_START: "queued", SUBMITTED: "queued", QUEUED: "queued", IN_PROGRESS: "in_progress", SUCCESS: "completed", FAILURE: "failed" }[task.status] ||
    "unknown";
  const output = publicResponse(task.data || {});
  const model = ctx.model || task.properties?.origin_model_name;
  if (typeof model === "string") output.model = PUBLIC_MODEL_NAMES[vendorModelName(model)] || model;
  return {
    object: "lecapi.task",
    id: task.task_id,
    status: status,
    progress: task.progress,
    output: output,
    ...(task.fail_reason ? { error: { message: task.fail_reason } } : {}),
  };
}

// Presenters translate only model identity fields. IDs, signed media URLs and
// other provider data stay intact, and the persisted upstream snapshot is read-only.
function publicResponse(value) {
  if (Array.isArray(value)) return value.map(publicResponse);
  if (!value || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value).map(([key, item]) => [
      key,
      ["model", "model_name", "origin_model_name", "upstream_model_name"].includes(key) && typeof item === "string"
        ? PUBLIC_MODEL_NAMES[vendorModelName(item)] || item
        : publicResponse(item),
    ])
  );
}

export const native = {
  createVideo: (ctx) => decodeRequest(ctx, "video", true),
  createImage: (ctx) => decodeRequest(ctx, "image", true),
  task: nativeTask,
};

export const protocols = {
  openai_video: {
    decodeRequest: (ctx) => decodeRequest(ctx, "video", false),
    render: (ctx, task) => {
      const body = publicResponse(responseObject(task.data || {}));
      if (typeof ctx.model === "string") body.model = PUBLIC_MODEL_NAMES[vendorModelName(ctx.model)] || ctx.model;
      return body;
    },
  },
  openai_image: {
    decodeRequest: (ctx) => decodeRequest(ctx, "image", false),
    render: (ctx, task) => ({ data: imageEntries(task.data, true), ...(task.data?.created ? { created: task.data.created } : {}) }),
  },
};
