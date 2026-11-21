// LecAPI public media catalog and documentation snapshot: 2026-11-20.
// Sources: https://api.lecapi.net/api/media-models/catalog, /api/pricing, and /docs.
// Catalog version: sha256:997b48bbd66fbf0869e486a04d35fd84bd6ec2f6604f15db13171cbf23bccfff.
// Pricing version: a42d372ccf0b5dd13ecf71203521f9d2.
// The 10:55 CST refresh returned 73 catalog and 84 pricing entries without changing either version ID.
// Static media kinds take precedence over inconsistent /v1/models labels.
// User pricing policy: published CNY price +20%, platform CNY/USD units 1:1.
// Each expression contains the final marked-up numeric coefficient, with no FX
// conversion or hidden multiplier. Rounding only removes binary float noise.
// Retired profiles remain readable for historical tasks but cannot accept new submissions.
// TinySnow Image 2 follows its Base64-only docs rather than the url capability.
// Missing or unverifiable prices remain unconfigured.
const MODELS = {
  "lec-ap-seedance-2-0-ot": {
    kind: "video",
    name: "Seedance 2.0 官转（AP）",
    icon: "Doubao.Color",
    tags: ["per_token"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "21:9", "4:3", "3:4"] },
      resolution: { type: "enum", required: true, default: "720p", values: ["720p", "1080p"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      first_frame: { type: "string", constraints: { url_schemes: ["https"] } },
      last_frame: { type: "string", constraints: { url_schemes: ["https"] } },
    },
    billing: {
      billing_unit: "million_tokens",
      tokens_per_second: { "720p": 21600, "1080p": 48600 },
      token_prices_per_million: { "720p": { without_video: 24.288, with_video: 14.784 }, "1080p": { without_video: 26.928, with_video: 16.368 } },
    },
  },
  "lec-ap-seedance-2-0-ot-fast": {
    kind: "video",
    name: "Seedance 2.0 Fast 官转（AP）",
    icon: "Doubao.Color",
    tags: ["per_token"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "21:9", "4:3", "3:4"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      first_frame: { type: "string", constraints: { url_schemes: ["https"] } },
      last_frame: { type: "string", constraints: { url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: {
      billing_unit: "million_tokens",
      tokens_per_second: { "720p": 21600 },
      token_prices_per_million: { "720p": { without_video: 19.536, with_video: 11.616 } },
    },
  },
  "lec-ap-seedance-2-0-ot-mini": {
    kind: "video",
    name: "Seedance 2.0 Mini 官转（AP）",
    icon: "Doubao.Color",
    tags: ["per_token"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "21:9", "4:3", "3:4"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      first_frame: { type: "string", constraints: { url_schemes: ["https"] } },
      last_frame: { type: "string", constraints: { url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: {
      billing_unit: "million_tokens",
      tokens_per_second: { "720p": 21600 },
      token_prices_per_million: { "720p": { without_video: 12.144, with_video: 7.392 } },
    },
  },
  "lec-ap-seedance-2-5-ot": {
    kind: "video",
    name: "Seedance 2.5 官转（AP）",
    icon: "Doubao.Color",
    tags: ["per_token"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "21:9", "4:3", "3:4", "adaptive"] },
      resolution: { type: "enum", required: true, default: "720p", values: ["720p", "1080p"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      first_frame: { type: "string", constraints: { url_schemes: ["https"] } },
      last_frame: { type: "string", constraints: { url_schemes: ["https"] } },
      omni_reference_task_type: { type: "enum", default: "auto", values: ["auto", "reference", "edit", "extend"] },
      draft: { type: "boolean" },
      draft_task_id: { type: "string" },
    },
    billing: {
      billing_unit: "million_tokens",
      tokens_per_second: { "720p": 21600, "1080p": 48600 },
      token_prices_per_million: { "720p": { without_video: 36.96, with_video: 22.176 }, "1080p": { without_video: 40.656, with_video: 24.288 } },
    },
  },
  "lec-vp-seedance-2-5-fp": {
    kind: "video",
    name: "Seedance 2.5 FP（VP）",
    icon: "Doubao.Color",
    tags: ["full", "multi", "image", "video", "audio", "supported", "per_second"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "21:9", "3:4", "4:3"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p", "1080p"] },
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
    },
    billing: { billing_unit: "second", equivalent_second_price: 0.3, resolution_prices: { "480p": 0.3, "720p": 0.35, "1080p": 1.05 } },
  },
  "lec-md-seedance-2-0-fast-900-720p": {
    // Still catalogued, but absent from the pricing and authenticated model lists.
    retired: true,
    kind: "video",
    name: "Seedance 2.0 Fast 900（MD）",
    icon: "Doubao.Color",
    tags: ["fast", "900", "720p", "image", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 15, values: [5, 10, 15], min: 5, max: 15 },
      aspect_ratio: { type: "enum", required: true, default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"], allow_data_url: true } },
      process_face: { type: "boolean" },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", price_per_request: 2 },
  },
  "lec-rs-seedance-2-0-fast": {
    kind: "video",
    name: "Seedance 2.0 Fast（RS）",
    icon: "Doubao.Color",
    tags: ["fast", "720p", "image", "face", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 10, values: [5, 10, 15], min: 5, max: 15 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["http", "https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", price_per_request: 1.3, resolution_prices: { "720p": 1.3 } },
  },
  "lec-mj-seedance-2-5-fd-720p": {
    retired: true,
    kind: "video",
    name: "Seedance 2.5（MJ）",
    icon: "Doubao.Color",
    tags: ["full", "720p", "image", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 15, values: [5, 8, 10, 12, 15, 18, 20, 25, 30], min: 5, max: 30 },
      aspect_ratio: { type: "enum", required: true, default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 0, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", price_per_request: 5.5, resolution_prices: { "720p": 5.5 } },
  },
  "lec-vg-seedance-2-5-vg-720p": {
    retired: true,
    kind: "video",
    name: "Seedance 2.5 VG（VG）",
    icon: "Doubao.Color",
    tags: ["full", "3010", "720p", "image", "audio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "adaptive"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", price_per_request: 5, resolution_prices: { "720p": 5 } },
  },
  "lec-zy-seedance-2-5-discount2": {
    kind: "video",
    name: "Seedance 2.5 二号（ZY）",
    icon: "Doubao.Color",
    tags: ["720p", "per_request", "image"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 30, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 0, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", price_per_request: 4.5, resolution_prices: { "720p": 4.5 } },
  },
  "lec-zy-seedance-2-5-discount": {
    kind: "video",
    name: "Seedance 2.5 一号（ZY）",
    icon: "Doubao.Color",
    tags: ["720p", "per_request", "image"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 30, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 0, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", price_per_request: 5.5, resolution_prices: { "720p": 5.5 } },
  },
  "lec-mf-seedance-2-5-x": {
    kind: "video",
    name: "Seedance 2.5 X（MF）",
    icon: "Doubao.Color",
    tags: ["x", "per_request", "480p", "720p", "native", "image", "video", "audio"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", price_per_request: 5, resolution_prices: { "480p": 5, "720p": 5 } },
  },
  "lec-haya-seedance-2-5-m": {
    kind: "video",
    name: "Seedance 2.5 M（HAYA）",
    icon: "Doubao.Color",
    tags: ["M", "multi", "image", "video", "audio", "supported", "per_second"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 5000 },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p"] },
      images: {
        type: "array",
        constraints: {
          max_items: 30,
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
          max_items: 10,
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
          max_items: 10,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["audio/aac", "audio/flac", "audio/mp4", "audio/mpeg", "audio/ogg", "audio/wav", "audio/x-wav"],
          max_bytes: 20971520,
          allow_files: true,
        },
      },
    },
    billing: { billing_unit: "second", equivalent_second_price: 0.25, resolution_prices: { "480p": 0.25, "720p": 0.35 } },
  },
  "lec-haya-seedance-2-5-flash": {
    retired: true,
    kind: "video",
    name: "Seedance 2.5 Flash（HAYA）",
    icon: "Doubao.Color",
    tags: ["Flash", "multi", "image", "video", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p"] },
      images: {
        type: "array",
        constraints: {
          max_items: 10,
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
          max_items: 5,
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
          max_items: 5,
          url_schemes: ["https"],
          allow_data_url: true,
          mime_types: ["audio/aac", "audio/flac", "audio/mp4", "audio/mpeg", "audio/ogg", "audio/wav", "audio/x-wav"],
          max_bytes: 20971520,
          allow_files: true,
        },
      },
    },
    billing: { billing_unit: "request", price_per_request: 3.5, resolution_prices: { "480p": 3.5, "720p": 4.5 } },
  },
  "lec-59-seedance-2-5-flash": {
    retired: true,
    kind: "video",
    name: "Seedance 2.5 Flash 1055（59）",
    icon: "Doubao.Color",
    tags: ["flash", "1055", "multi", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 5000 },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 10, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", price_per_request: 3, resolution_prices: { "480p": 3, "720p": 4 } },
  },
  "lec-59-seedance-2-0-mini": {
    retired: true,
    kind: "video",
    name: "Seedance 2.0 Mini 933（59）",
    icon: "Doubao.Color",
    tags: ["mini", "933", "multi", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 8000 },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 15, step: 1, values: [4, 5, 6, 7, 8, 9, 10, 11, 13, 14, 15] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", price_per_request: 0.9, resolution_prices: { "480p": 0.9, "720p": 1.2 } },
  },
  "lec-59-seedance-2-0-full": {
    retired: true,
    kind: "video",
    name: "Seedance 2.0 满血 933（59）",
    icon: "Doubao.Color",
    tags: ["full", "933", "720p", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 8000 },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 15, step: 1, values: [4, 5, 6, 7, 8, 9, 10, 11, 13, 14, 15] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", price_per_request: 3.3, resolution_prices: { "720p": 3.3 } },
  },
  "lec-59-seedance-2-0-fast": {
    retired: true,
    kind: "video",
    name: "Seedance 2.0 Fast 933（59）",
    icon: "Doubao.Color",
    tags: ["fast", "933", "multi", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 8000 },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 15, step: 1, values: [4, 5, 6, 7, 8, 9, 10, 11, 13, 14, 15] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", price_per_request: 1.3, resolution_prices: { "480p": 1.3, "720p": 1.8 } },
  },
  "lec-hc-seedance-2-5-3010": {
    kind: "video",
    name: "Seedance 2.5 3010（HC）",
    icon: "Doubao.Color",
    tags: ["3010", "multi", "per_request", "native", "image", "audio"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 5000 },
      duration: { type: "integer", required: true, default: 5, min: 4, max: 30, step: 1 },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", price_per_request: 6, resolution_prices: { "480p": 4, "720p": 6 } },
  },
  "lec-jl-seedance-2-0-fast-auto": {
    retired: true,
    kind: "video",
    name: "Seedance 2.0 Fast 933（JL）",
    icon: "Doubao.Color",
    tags: ["933", "per_request", "native", "image", "video", "audio"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 4, max: 15, step: 1 },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 9 } },
      videos: { type: "array", constraints: { max_items: 3 } },
      audios: { type: "array", constraints: { max_items: 3 } },
    },
    billing: { billing_unit: "request", price_per_request: 2, resolution_prices: { "480p": 1.5, "720p": 2 } },
  },
  "lec-jl-seedance-2-0-full": {
    retired: true,
    kind: "video",
    name: "Seedance 2.0 满血 933（JL）",
    icon: "Doubao.Color",
    tags: ["933", "per_request", "native", "image", "video", "audio"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 8000 },
      duration: { type: "integer", required: true, default: 5, min: 4, max: 15, step: 1 },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 9 } },
      videos: { type: "array", constraints: { max_items: 3 } },
      audios: { type: "array", constraints: { max_items: 3 } },
    },
    billing: { billing_unit: "request", price_per_request: 3, resolution_prices: { "480p": 2, "720p": 3 } },
  },
  "lec-mf-seedance-2-5-t1": {
    retired: true,
    kind: "video",
    name: "Seedance 2.5 T1（MF）",
    icon: "Doubao.Color",
    tags: ["t1", "per_request", "480p", "720p", "native", "image", "video", "audio"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 5000 },
      duration: { type: "integer", default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", price_per_request: 7.5, resolution_prices: { "480p": 4.5, "720p": 7.5 } },
  },
  "lec-mg-seedance-2-5": {
    kind: "video",
    name: "Seedance 2.5（MG）",
    icon: "Doubao.Color",
    tags: ["720p", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["21:9", "16:9", "4:3", "1:1", "3:4", "9:16"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
    },
    billing: { billing_unit: "request", price_per_request: 5.5, resolution_prices: { "720p": 5.5 } },
  },
  "lec-ty-seedance-2-5-1050-flxd": {
    kind: "video",
    name: "Seedance 2.5 1050 FLXD（TY）",
    icon: "Doubao.Color",
    tags: ["full", "1050", "multi", "image", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 10, url_schemes: ["http", "https"] } },
      videos: { type: "array", constraints: { max_items: 0, url_schemes: ["http", "https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["http", "https"] } },
    },
    billing: { billing_unit: "request", price_per_request: 4, resolution_prices: { "480p": 4, "720p": 6 } },
  },
  "lec-ty-seedream-5-5": {
    kind: "image",
    name: "Seedream 5.5（TY）",
    icon: "Doubao.Color",
    tags: ["image", "aspect_ratio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      resolution: { type: "enum", default: "1K", values: ["1K", "2K"] },
      aspect_ratio: { type: "enum", default: "1:1", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      images: { type: "array", constraints: { max_items: 8, url_schemes: ["http", "https"] } },
    },
    billing: { billing_unit: "request", price: 0.13 },
  },
  "lec-vw-seedance-2-0-fast": {
    retired: true,
    kind: "video",
    name: "Seedance 2.0 Fast（VW）",
    icon: "Doubao.Color",
    tags: ["fast", "image", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      aspect_ratio: { type: "enum", default: "auto", values: ["auto", "16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 9, url_schemes: ["https"], max_bytes: 15728640 } },
      resolution: { type: "enum", default: "720p", values: ["720p"] },
      duration: { type: "integer", default: 15, values: [15], min: 15, max: 15 },
    },
    billing: { billing_unit: "request", price_per_request: 1.3 },
  },
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
    retired: true,
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
    retired: true,
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
    bill_reference_videos: true,
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
    bill_reference_videos: true,
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
    retired: true,
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
      duration: { type: "integer", required: true, default: 10, values: [10, 20, 30, 40, 50, 60], min: 10, max: 60 },
      aspect_ratio: { type: "enum", required: true, default: "16:9", values: ["1:1", "16:9", "9:16"] },
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
    retired: true,
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
    retired: true,
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
    retired: true,
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
    retired: true,
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
    retired: true,
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
  "lec-jm-seedance-2-5-flash": {
    retired: true,
    kind: "video",
    name: "Seedance 2.5 Flash（JM）",
    icon: "Doubao.Color",
    tags: ["flash", "per_second", "supported", "image", "audio"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 4, max: 30, step: 1 },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["21:9", "16:9", "4:3", "1:1", "3:4", "9:16"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "second", resolution_prices: { "480p": 0.15, "720p": 0.2 }, equivalent_second_price: 0.2 },
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
    billing: { billing_unit: "request", resolution_prices: { "720p": 6 }, price_per_request: 6 },
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
    billing: { billing_unit: "request", resolution_prices: { "720p": 5.5 }, price_per_request: 5.5 },
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
    billing: { billing_unit: "request", resolution_prices: { "720p": 5.5 }, price_per_request: 5.5 },
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
      aspect_ratio: { type: "enum", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
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
    retired: true,
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
    retired: true,
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
    billing: { billing_unit: "second", resolution_prices: { "768p": 0.06 }, equivalent_second_price: 0.06 },
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
    retired: true,
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
    retired: true,
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
      duration: {
        type: "integer",
        required: true,
        default: 15,
        min: 4,
        max: 30,
        step: 1,
        values: [4, 5, 6, 7, 8, 9, 10, 11, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30],
      },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16"] },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
    },
    billing: { billing_unit: "second", resolution_prices: { "480p": 0.24, "720p": 0.34 }, equivalent_second_price: 0.34 },
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
    retired: true,
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
    retired: true,
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
    retired: true,
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
      duration: { type: "integer", default: 5, min: -1, max: 15, step: 1, values: [-1, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["adaptive", "16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p", "1080p"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
      mode: { type: "enum", values: ["text2video", "singleImage2video", "image2video", "frames2video", "mixed2video"] },
    },
    billing: { billing_unit: "second", resolution_prices: { "1080p": 1.55, "480p": 0.45, "720p": 0.75 }, equivalent_second_price: 0.45 },
  },
  "lec-gt-seedance-2-5-720p": {
    retired: true,
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
    retired: true,
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
    tags: ["full", "480p", "image", "audio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 4, min: 2, max: 20, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "480p": 2.5 }, price_per_request: 3 },
  },
  "lec-haya-seedance-2-5-xg-480p": {
    kind: "video",
    name: "Seedance 2.5 XG 3（HAYA）",
    icon: "Doubao.Color",
    tags: ["XG", "480p", "image", "video", "audio", "supported", "per_request"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 5000 },
      duration: { type: "integer", required: true, default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4"] },
      images: {
        type: "array",
        constraints: {
          max_items: 30,
          url_schemes: ["https"],
          allow_data_url: true,
          allow_files: true,
          mime_types: ["image/png", "image/jpeg", "image/webp", "image/gif"],
          max_bytes: 20971520,
        },
      },
      videos: {
        type: "array",
        constraints: {
          max_items: 10,
          url_schemes: ["https"],
          allow_data_url: true,
          allow_files: true,
          mime_types: ["video/mp4", "video/webm", "video/quicktime"],
          max_bytes: 20971520,
        },
      },
      audios: {
        type: "array",
        constraints: {
          max_items: 10,
          url_schemes: ["https"],
          allow_data_url: true,
          allow_files: true,
          mime_types: ["audio/aac", "audio/flac", "audio/mp4", "audio/mpeg", "audio/ogg", "audio/wav", "audio/x-wav"],
          max_bytes: 20971520,
        },
      },
    },
    billing: { billing_unit: "request", price_per_request: 4.5 },
  },
  "lec-hc-seedance-2-0-903": {
    retired: true,
    kind: "video",
    name: "Seedance 2.0 满血 903（HC）",
    icon: "Doubao.Color",
    tags: ["full", "903", "720p", "per_request", "native", "image", "audio"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 15, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1"] },
      images: { type: "array", constraints: { max_items: 9, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", price_per_request: 3 },
  },
  "lec-mf-seedance-2-5-std": {
    kind: "video",
    name: "Seedance 2.5（MF）",
    icon: "Doubao.Color",
    tags: ["multi", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 15000 },
      duration: { type: "integer", default: 4, min: 4, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      resolution: { type: "enum", default: "480p", values: ["480p", "720p", "1080p", "4k"] },
      images: { type: "array", constraints: { max_items: 30, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "480p": 4, "720p": 5, "1080p": 8, "4k": 9 } },
  },
  "lec-mw-vidu-q4-nopeak": {
    kind: "video",
    name: "Vidu Q4（MW）",
    icon: "Vidu.Color",
    tags: ["multi", "image", "audio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 2, max: 16, step: 1 },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p", "1080p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", required: true, constraints: { min_items: 1, max_items: 10, url_schemes: ["http", "https"] } },
      audios: { type: "array", constraints: { max_items: 3, url_schemes: ["http", "https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "480p": 0.7, "720p": 0.9, "1080p": 1.1 } },
  },
  "lec-mw-wan-3-0-s": {
    kind: "video",
    name: "Wan 3.0 S（MW）",
    icon: "AlibabaCloud.Color",
    tags: ["multi", "image", "video", "audio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 5, min: 5, max: 30, step: 1 },
      resolution: { type: "enum", default: "720p", values: ["720p", "1080p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"] } },
      videos: { type: "array", constraints: { max_items: 5, url_schemes: ["http", "https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["http", "https"] } },
    },
    billing: { billing_unit: "request", price_per_request: 3.5 },
  },
  "lec-seedance-2-5-301010": {
    kind: "video",
    name: "Seedance 2.5 301010（渠道1）",
    icon: "Doubao.Color",
    tags: ["720p", "image", "video", "audio", "native", "per_request"],
    parameters: {
      prompt: { type: "string", required: true, maxLength: 5000 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3"] },
      images: {
        type: "array",
        required: true,
        constraints: { min_items: 1, max_items: 30, url_schemes: ["https"], mime_types: ["image/png", "image/jpeg", "image/webp"], max_bytes: 15728640 },
      },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"], mime_types: ["video/mp4"], max_bytes: 62914560 } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"], mime_types: ["audio/mpeg", "audio/wav"], max_bytes: 52428800 } },
    },
    billing: { billing_unit: "request", price_per_request: 4.7 },
  },
  "lec-mj-seedance-2-5-qn": {
    kind: "video",
    name: "Seedance 2.5 QN（MJ）",
    icon: "Doubao.Color",
    tags: ["full", "multi", "image", "video", "audio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", required: true, default: 15, min: 4, max: 30, step: 1 },
      resolution: { type: "enum", default: "720p", values: ["480p", "720p"] },
      aspect_ratio: { type: "enum", default: "16:9", values: ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"] },
      images: { type: "array", constraints: { max_items: 20, url_schemes: ["https"] } },
      videos: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
      audios: { type: "array", constraints: { max_items: 10, url_schemes: ["https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "480p": 3.5, "720p": 5.5 } },
  },
  "lec-vg-wan-3-0": {
    retired: true,
    kind: "video",
    name: "Wan 3.0 P（VG）",
    icon: "AlibabaCloud.Color",
    tags: ["multi", "image", "video", "audio", "per_request"],
    parameters: {
      prompt: { type: "string", required: true },
      duration: { type: "integer", default: 5, min: 2, max: 30, step: 1 },
      aspect_ratio: { type: "enum", default: "16:9", values: ["auto", "16:9", "4:3", "1:1", "3:4", "9:16"] },
      resolution: { type: "enum", default: "720p", values: ["720p", "1080p"] },
      images: { type: "array", constraints: { max_items: 10, url_schemes: ["http", "https"], allow_data_url: true, allow_files: true } },
      videos: { type: "array", constraints: { max_items: 5, url_schemes: ["http", "https"] } },
      audios: { type: "array", constraints: { max_items: 5, url_schemes: ["http", "https"] } },
    },
    billing: { billing_unit: "request", resolution_prices: { "720p": 1.5, "1080p": 1.8 } },
  },
  "lec-image-2": {
    retired: true,
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
    retired: true,
    kind: "image",
    name: "Image generation",
    icon: "OpenAI.Color",
    tags: ["image"],
    parameters: {
      prompt: { type: "string", required: true },
      images: { type: "array", constraints: { url_schemes: ["https"] } },
      aspect_ratio: { type: "enum", default: "1:1", values: ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"] },
      resolution: { type: "enum", values: ["1K", "2K", "4K"] },
      quality: { type: "enum", values: ["standard", "thinking"], default: "standard" },
      response_format: { type: "enum", values: ["url"], default: "url" },
    },
    billing: {},
  },
  "nano-banana-2": {
    retired: true,
    kind: "image",
    name: "Image generation Pro",
    icon: "Gemini.Color",
    tags: ["image"],
    parameters: {
      prompt: { type: "string", required: true },
      images: { type: "array", constraints: { url_schemes: ["https"] } },
      aspect_ratio: { type: "enum", values: ["16:9", "9:16", "1:1", "2:3", "3:2"] },
      resolution: { type: "enum", values: ["1K", "2K", "4K"] },
      response_format: { type: "enum", values: ["url"], default: "url" },
    },
    billing: {},
  },
  "lec-vp-seedance-2-9-301010": {
    retired: true,
    kind: "video",
    name: "Seedance 2.9 301010 (VP)",
    icon: "Doubao.Color",
    tags: ["video"],
    parameters: { prompt: { type: "string", required: true } },
    billing: {},
    unverified: true,
  },
  "lec-omni-video-redraw": {
    retired: true,
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

// These fixed output settings are omitted from the upstream JSON contract.
const FIXED_VIDEO_OUTPUTS = {
  "lec-haya-seedance-2-5-xg-480p": { resolution: "480p" },
  "lec-hc-seedance-2-0-903": { resolution: "720p" },
  "lec-md-seedance-2-0-fast-900-720p": { resolution: "720p" },
  "lec-mf-seedance-2-5-30ref": { resolution: "720p" },
  "lec-mg-seedance-2-5": { resolution: "720p" },
  "lec-rs-seedance-2-0-fast": { resolution: "720p" },
  "lec-seedance-2-0-fast-933-720p": { resolution: "720p" },
  "lec-seedance-2-0-full-933-720p": { resolution: "720p" },
  "lec-seedance-2-0-mini-b1-480p": { resolution: "480p" },
  "lec-seedance-2-0-mini-c9-933-720p": { resolution: "720p" },
  "lec-seedance-2-5-301010": { resolution: "720p", duration: 30 },
  "lec-ty-seedance-2-0-full-933-w-720p": { resolution: "720p" },
  "lec-vp-seedance-2-5-301010-49": { resolution: "720p" },
  "lec-vv-seedance-2-5-r2": { resolution: "480p" },
  "lec-vv-seedance-2-5-r2-plus": { resolution: "720p" },
  "lec-vw-seedance-2-0-fast": { resolution: "720p", duration: 15 },
};

const INTERNAL_VIDEO_MODELS = new Set();
// Published IDs and established route affinities survive catalog refreshes.
// Known siblings may replace a removed anchor without renaming the pool.
const VIDEO_POOLS = {
  "seedance-2-5-01": ["seedance-2-5-request-660", "lec-mm-seedance-2-5", "lec-mm-seedance-2-5-full", "lec-mg-seedance-2-5", "lec-vp-seedance-2-5-301010-49", "lec-mj-seedance-2-5-bd-720p", "lec-mj-seedance-2-5-qn", "lec-zy-seedance-2-5-discount"],
  "seedance-2-5-02": ["seedance-2-5-request-720", "lec-mf-seedance-2-5-30ref", "lec-hc-seedance-2-5-3010", "lec-ty-seedance-2-5-1050-flxd"],
  "seedance-2-5-03": ["seedance-2-5-request-600-480p-480-1080p-960-4k-1080", "lec-mf-seedance-2-5-std", "lec-seedance-2-5-301010", "lec-zy-seedance-2-5-discount2"],
  "seedance-2-5-04": ["seedance-2-5-request-600-480p-600", "lec-mf-seedance-2-5-x", "lec-haya-seedance-2-5-xg-480p"],
  "seedance-2-0-01": ["seedance-2-0-request-420", "lec-vp-seedance-2-0-903", "lec-ty-seedance-2-0-full-933-w-720p"],
  "seedance-2-0-02": ["seedance-2-0-request-540", "lec-vp-seedance-2-0-933-s2", "lec-seedance-2-0-fast-933-720p"],
};

const PRICE_MARKUP = 1.2;

function roundedSalePrice(markedUpPrice, unit) {
  const scale = unit === "second" ? 100 : 10;
  return Math.ceil(Number((markedUpPrice * scale).toFixed(10))) / scale;
}

function compatibleVideoRates(pool, candidate) {
  if (pool.unit !== candidate.unit || pool.durationKey !== candidate.durationKey) return false;
  const shared = candidate.rates.filter(([res]) => pool.anchors[res] !== undefined);
  return shared.length > 0 && shared.every(([res, rate]) => {
    const anchor = pool.anchors[res];
    if (pool.unit === "second") return rate === anchor;
    // Round once after averaging. Compare the marked-up, unrounded rates.
    const delta = Number(((rate - anchor) * PRICE_MARKUP).toFixed(10));
    return delta >= -0.5 && delta <= 0.2;
  });
}

function addVideoRoute(pool, candidate) {
  for (const [resolution, price] of candidate.rates) {
    pool.anchors[resolution] ??= price;
    (pool.prices[resolution] ??= []).push(price);
  }
  pool.routes.push(candidate.vendorModel);
}

function compareVideoCapabilities(left, right) {
  const a = Object.values(MODELS[left].parameters).filter((parameter) => parameter.constraints?.max_items !== 0);
  const b = Object.values(MODELS[right].parameters).filter((parameter) => parameter.constraints?.max_items !== 0);
  const breadth = (parameters) => parameters.reduce((sum, parameter) => sum + (parameter.type === "enum" ? parameter.values.length : parameter.constraints?.max_items ?? 0), 0);
  const leftDurations = MODELS[left].parameters.duration?.values;
  const rightDurations = MODELS[right].parameters.duration?.values;
  return b.length - a.length || (leftDurations && rightDurations ? rightDurations.length - leftDurations.length : 0) || breadth(b) - breadth(a);
}

// A pool may add new resolutions. At every shared resolution, compare against
// the original anchor, never the running mean. Only per-request rates can vary.
for (const [family, pattern] of [
  ["seedance-2-5", /-seedance-2-5(?:-|$)/],
  ["seedance-2-0", /-seedance-2-0(?:-|$)/],
]) {
  const candidates = new Map();
  for (const [vendorModel, profile] of Object.entries(MODELS)) {
    if (!pattern.test(vendorModel) || profile.retired || profile.kind !== "video") continue;
    const billing = profile.billing;
    const unit = billing.billing_unit;
    if (!["request", "second"].includes(unit)) continue;
    if (profile.source_duration || profile.bill_reference_videos || billing.duration_prices || billing.video_reference_prices) continue;
    const fixed = FIXED_VIDEO_OUTPUTS[vendorModel]?.resolution;
    const resolutions = fixed ? [fixed] : profile.parameters.resolution?.values ?? Object.keys(billing.resolution_prices || {});
    const baseRate = unit === "request" ? billing.price_per_request : billing.equivalent_second_price;
    const rates = resolutions.map((resolution) => [resolution, billing.resolution_prices ? billing.resolution_prices[resolution] : baseRate]);
    if (!rates.length || rates.some(([, rate]) => !Number.isFinite(rate) || rate <= 0)) continue;
    const duration = profile.parameters.duration;
    // Pooled per-second settlement validates reported duration using the
    // representative's contract. Exclude extra charges and mismatched limits.
    if (unit === "second" && !duration) continue;
    const durationKey = unit === "second" ? JSON.stringify([duration.min, duration.max, duration.values]) : "";
    candidates.set(vendorModel, { vendorModel, unit, durationKey, rates });
  }
  const pools = [];
  for (const [model, [legacy, ...siblings]] of Object.entries(VIDEO_POOLS)) {
    if (!model.startsWith(family + "-")) continue;
    const knownRoutes = siblings.map((route) => candidates.get(route)).filter(Boolean);
    const anchor = knownRoutes[0];
    const pool = { model, legacy, unit: anchor?.unit ?? "request", durationKey: anchor?.durationKey, anchors: {}, prices: {}, routes: [] };
    for (const candidate of knownRoutes) {
      if (!pool.routes.length || compatibleVideoRates(pool, candidate)) addVideoRoute(pool, candidate);
      // A repriced sibling remains a standalone model, not a member of a different pool.
      candidates.delete(candidate.vendorModel);
    }
    pools.push(pool);
  }
  for (const candidate of candidates.values()) {
    const compatible = pools.filter((pool) => compatibleVideoRates(pool, candidate));
    const exact = compatible.filter((pool) => candidate.rates.some(([res, rate]) => pool.anchors[res] === rate));
    const matches = exact.length ? exact : compatible;
    if (matches.length === 1) addVideoRoute(matches[0], candidate);
  }
  for (const { model, legacy, unit, anchors, prices, routes } of pools) {
    if (!routes.length) {
      MODELS[model] = { kind: "video", poolUnavailable: true, historicalPool: true, parameters: { duration: { min: 4, max: 30 } }, billing: { billing_unit: unit }, tags: [] };
      MODELS[legacy] = { ...MODELS[model], legacyPool: true };
      continue;
    }
    const resolutions = Object.keys(prices).sort((a, b) => parseInt(a, 10) * (a.endsWith("k") ? 1000 : 1) - parseInt(b, 10) * (b.endsWith("k") ? 1000 : 1));
    const resolution = anchors["720p"] !== undefined ? "720p" : resolutions[0];
    const resolutionPrices = Object.fromEntries(resolutions.map((res) => {
      const rates = prices[res];
      return [res, rates.reduce((sum, rate) => sum + rate, 0) / rates.length];
    }));
    routes.sort(compareVideoCapabilities);
    const preferred = MODELS[routes[0]];
    const parameters = { ...preferred.parameters, resolution: { type: "enum", default: resolution, values: resolutions } };
    for (const field of ["images", "videos", "audios"]) {
      const supported = routes.map((route) => MODELS[route].parameters[field]).filter((parameter) => parameter && parameter.constraints?.max_items !== 0);
      if (!supported.length) continue;
      const best = supported.reduce((a, b) => (b.constraints?.max_items ?? 0) > (a.constraints?.max_items ?? 0) ? b : a);
      const required = routes.every((route) => MODELS[route].parameters[field]?.required);
      parameters[field] = { ...best, required, constraints: { ...best.constraints, min_items: required ? best.constraints?.min_items : undefined } };
    }
    MODELS[model] = {
      ...preferred,
      parameters,
      billing: { billing_unit: unit, resolution_prices: resolutionPrices },
      tags: routes.flatMap((route) => MODELS[route].tags).filter((tag) => ["image", "video", "audio", "per_request", "per_second"].includes(tag)),
      routes,
    };
    // Legacy names are only advertised as channel mappings, not plugin models.
    MODELS[legacy] = { ...MODELS[model], legacyPool: true };
    for (const route of routes) INTERNAL_VIDEO_MODELS.add(route);
  }
}

// Previous pooled names remain resolvable for polling, not new work.
for (const model of [
  "seedance-2-5", "seedance-2-5-request-420", "seedance-2-5-request-480", "seedance-2-5-request-540",
  "seedance-2-5-request-600", "seedance-2-5-request-756", "seedance-2-5-request-960", "seedance-2-5-request-1080",
  "seedance-2-0-request-156", "seedance-2-0-request-180", "seedance-2-0-request-240", "seedance-2-0-request-360",
]) {
  if (!MODELS[model]) MODELS[model] = { kind: "video", retired: true, historicalPool: true, parameters: { duration: { min: 4, max: 30 } }, billing: { billing_unit: "request" } };
}

// Public identities stay independent from LecAPI's exact wire model names.
const MODEL_MAPPING = {};
const PUBLIC_MODEL_NAMES = {};
const MODEL_LOOKUP = {};
for (const vendorModel of Object.keys(MODELS)) {
  // Reuse the exact spelling already shared by Hailuo's host binding.
  const publicModel = vendorModel === "lec-minimax-h3" ? "MiniMax-H3" : vendorModel.startsWith("lec-") ? vendorModel.slice(4) : vendorModel;
  if (Object.prototype.hasOwnProperty.call(MODEL_MAPPING, publicModel)) throw new Error("LecAPI public model id collision: " + publicModel);
  if (!MODELS[vendorModel].retired && !MODELS[vendorModel].legacyPool && !INTERNAL_VIDEO_MODELS.has(vendorModel)) MODEL_MAPPING[publicModel] = vendorModel;
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
const ASYNC_IMAGE_MODELS = ["lec-sl-midjourney", "lec-sl-midjourney-niji", "lec-ty-seedream-5-pro", "lec-ty-seedream-5-5"];
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
const TOKEN_VIDEO_USAGE_SCHEMA = {
  ...VIDEO_USAGE_SCHEMA,
  tokens: { type: "number", unit: "token", description: { en: "Video generation token unit price", zh: "视频生成 Token 单价" } },
};

function defaultExpression(profile) {
  const billing = profile.billing;
  const quantity =
    billing.price_per_image !== undefined
      ? "image_count"
      : billing.billing_unit === "million_tokens"
        ? "tokens"
        : billing.billing_unit === "second"
          ? "seconds"
          : "requests";
  const priceWithFee = (price) => {
    const markedUp = price * PRICE_MARKUP;
    return profile.kind === "video" && ["request", "second"].includes(billing.billing_unit)
      ? roundedSalePrice(markedUp, billing.billing_unit)
      : Number(markedUp.toFixed(12));
  };
  const leaf = (name, price) =>
    "tier(" +
    JSON.stringify(name) +
    ", u(" +
    JSON.stringify(quantity) +
    ") * " +
    priceWithFee(price) +
    (billing.billing_unit === "million_tokens" ? " / 1000000" : "") +
    (profile.bill_reference_videos ? ' + u("reference_seconds") * ' + priceWithFee(price) : "") +
    ")";
  if (billing.token_prices_per_million) {
    const resolutions = Object.keys(billing.token_prices_per_million),
      branches = [];
    for (let index = 0; index < resolutions.length; index++) {
      const resolution = resolutions[index],
        last = index === resolutions.length - 1,
        rates = billing.token_prices_per_million[resolution],
        condition = 'u("resolution") == ' + JSON.stringify(resolution);
      branches.push((last ? "" : condition + " && ") + 'u("video_input") == "video" ? ' + leaf(resolution + "-reference", rates.with_video));
      branches.push((last ? "" : condition + " ? ") + leaf(resolution, rates.without_video));
    }
    return branches.join(" : ");
  }
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
            Number((billing.extra_image_price * PRICE_MARKUP).toFixed(12)) +
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

const PARAMETER_LABELS = {
  prompt: ["Prompt", "提示词"],
  subjects: ["Subject names", "主体列表"],
  duration: ["Output duration", "生成时长"],
  aspect_ratio: ["Aspect ratio", "画面比例"],
  resolution: ["Output resolution", "输出分辨率"],
  images: ["Reference images", "参考图片"],
  videos: ["Reference videos", "参考视频"],
  audios: ["Reference audio", "参考音频"],
  documents: ["Reference documents", "参考文档"],
  first_frame: ["First frame image", "首帧图片"],
  last_frame: ["Last frame image", "尾帧图片"],
  output_format: ["Output image format", "输出图片格式"],
  response_format: ["Result format", "结果返回格式"],
  n: ["Output image count", "生成图片数量"],
  size: ["Output dimensions", "输出尺寸"],
  quality: ["Image quality", "图片质量"],
  audio: ["Generate audio", "是否生成音频"],
  fps: ["Frame rate", "输出帧率"],
  action: ["Image operation", "图片操作"],
  version: ["Model version", "模型版本"],
  image_size: ["Image size tier", "图片尺寸档位"],
  style: ["Enable raw style", "是否启用原始风格"],
  hd: ["Enable HD generation", "是否启用高清生成"],
  draft: ["Enable draft mode", "是否启用草稿模式"],
  tile: ["Generate a tileable pattern", "是否生成可平铺图案"],
  seed: ["Random seed", "随机种子"],
  stylize: ["Stylization strength", "风格化强度"],
  chaos: ["Composition variation", "构图变化程度"],
  weird: ["Creative variation", "创意偏离程度"],
  original_task_id: ["Original task ID", "原始任务编号"],
  source_task_id: ["Source task ID", "来源任务编号"],
  index: ["Original image index", "原图序号"],
  reference_video_seconds: ["Reference video duration", "参考视频合计时长"],
  process_face: ["Enable face processing", "是否启用人脸处理"],
  mode: ["Processing mode", "处理模式"],
  omni_reference_task_type: ["Reference video task type", "参考视频用途"],
  draft_task_id: ["Draft task ID", "草稿任务编号"],
};
const PARAMETER_VALUE_LABELS = {
  auto: "自动",
  adaptive: "自适应",
  "-1": "智能时长",
  text2video: "文生视频",
  singleImage2video: "单图生视频",
  image2video: "多图生视频",
  frames2video: "首尾帧生视频",
  mixed2video: "混合素材生视频",
  true: "开启",
  false: "关闭",
  url: "图片链接",
  b64_json: "Base64 图片",
  imagine: "生成图片",
  edits: "编辑图片",
  blend: "融合图片",
  hd_upscale: "高清放大",
  low: "低",
  medium: "中",
  high: "高",
  reference: "素材参考",
  edit: "编辑视频",
  extend: "延长视频",
};

function describeVideoPoolMaterials(profile) {
  const fields = ["images", "videos", "audios"];
  const labels = { en: ["images", "videos", "audio clips"], zh: [" 张图", " 段视频", " 段音频"] };
  const english = [], chinese = [];
  for (const resolution of profile.parameters.resolution.values) {
    const shapes = new Map();
    for (const route of profile.routes) {
      const candidate = MODELS[route];
      const fixed = FIXED_VIDEO_OUTPUTS[route]?.resolution;
      if (fixed ? fixed !== resolution : !(candidate.parameters.resolution?.values ?? Object.keys(candidate.billing.resolution_prices)).includes(resolution)) continue;
      const maxima = fields.map((field) => candidate.parameters[field]?.constraints?.max_items ?? 0);
      const minima = fields.map((field) => candidate.parameters[field]?.constraints?.min_items || (candidate.parameters[field]?.required ? 1 : 0));
      const sum = maxima.reduce((a, b) => a + b, 0);
      const total = Math.min(candidate.max_reference_total ?? sum, sum);
      const videoSeconds = maxima[1] ? candidate.max_reference_video_seconds ?? 0 : 0;
      shapes.set(JSON.stringify([maxima, minima, total, videoSeconds]), { maxima, minima, total, sum, videoSeconds });
    }
    const all = [...shapes.values()];
    // Keep alternatives unless a single other combination covers all of their counts.
    const options = all.filter((a) => !all.some((b) => a !== b && b.total >= a.total && (!a.maxima[1] || !b.videoSeconds || (a.videoSeconds && b.videoSeconds >= a.videoSeconds)) && b.maxima.every((max, i) => max >= a.maxima[i] && b.minima[i] <= a.minima[i])));
    const en = [], zh = [];
    for (const { maxima, minima, total, sum, videoSeconds } of options) {
      const enCounts = [], zhCounts = [];
      for (let i = 0; i < fields.length; i++) {
        if (!maxima[i]) continue;
        const range = minima[i] ? minima[i] + "–" + maxima[i] : maxima[i];
        enCounts.push((minima[i] ? "" : "up to ") + range + " " + labels.en[i]);
        zhCounts.push((minima[i] ? "" : "最多 ") + range + labels.zh[i]);
      }
      const enLimits = [], zhLimits = [];
      if (total < sum) {
        enLimits.push("up to " + total + " materials combined");
        zhLimits.push("合计最多 " + total + " 项");
      }
      if (videoSeconds) {
        enLimits.push("up to " + videoSeconds + " seconds per video");
        zhLimits.push("单段视频最长 " + videoSeconds + " 秒");
      }
      en.push((enCounts.join(" + ") || "text only") + (enLimits.length ? " (" + enLimits.join(", ") + ")" : ""));
      zh.push((zhCounts.join(" + ") || "纯文生") + (zhLimits.length ? "（" + zhLimits.join("，") + "）" : ""));
    }
    english.push(resolution + ": " + en.join(" or "));
    chinese.push(resolution + "：" + zh.join("，或 "));
  }
  return ["Reference combinations by resolution: " + english.join(" | "), "各分辨率的参考素材组合：" + chinese.join(" | ")];
}

function describeVideoPoolRatios(profile) {
  return profile.parameters.resolution.values.map((resolution) => {
    const ratios = new Set();
    for (const route of profile.routes) {
      const candidate = MODELS[route];
      const fixed = FIXED_VIDEO_OUTPUTS[route]?.resolution;
      if (fixed ? fixed !== resolution : !(candidate.parameters.resolution?.values ?? Object.keys(candidate.billing.resolution_prices)).includes(resolution)) continue;
      for (const ratio of candidate.parameters.aspect_ratio?.values ?? []) ratios.add(ratio);
    }
    return ratios.size ? resolution + " (" + [...ratios].join(" / ") + ")" : "";
  }).filter(Boolean).join(" | ");
}

function modelDescription(profile) {
  if (profile.poolUnavailable) return { en: "No upstream route currently available", zh: "当前没有可用的上游线路" };
  const english = [],
    chinese = [];
  const types = { string: "字符串类型", integer: "整数类型", number: "数字类型", boolean: "布尔类型", array: "数组类型" };
  for (const [field, parameter] of Object.entries(profile.parameters)) {
    const labels = PARAMETER_LABELS[field] || [field, field];
    const en = [labels[0]],
      zh = [labels[1]];
    if (parameter.type !== "enum") {
      en.push(parameter.type);
      zh.push(types[parameter.type] || parameter.type);
    }
    if (parameter.required) {
      en.push("required");
      zh.push("必填");
    }
    if (field === "prompt") {
      en.push("describe the content to generate");
      zh.push("用来描述生成内容");
    }
    const seconds = ["duration", "reference_video_seconds"].includes(field);
    if (parameter.values) {
      en.push("options: " + parameter.values.join(" / ") + (seconds ? " seconds" : ""));
      zh.push(
        "可选 " +
          parameter.values.map((value) => String(value) + (PARAMETER_VALUE_LABELS[value] ? "（" + PARAMETER_VALUE_LABELS[value] + "）" : "")).join(" / ") +
          (seconds ? " 秒" : "")
      );
    } else if (parameter.min !== undefined || parameter.max !== undefined) {
      const range =
        parameter.min !== undefined && parameter.max !== undefined
          ? parameter.min + "–" + parameter.max
          : parameter.min !== undefined
            ? "≥" + parameter.min
            : "≤" + parameter.max;
      en.push(range + (seconds ? " seconds" : ""));
      zh.push("支持 " + range + (seconds ? " 秒" : ""));
    } else if (parameter.type === "boolean") {
      en.push("true to enable, false to disable");
      zh.push("true 为开启，false 为关闭");
    }
    if (parameter.default !== undefined) {
      en.push("default " + parameter.default + (seconds ? " seconds" : ""));
      zh.push(
        "默认 " +
          parameter.default +
          (PARAMETER_VALUE_LABELS[parameter.default] ? "（" + PARAMETER_VALUE_LABELS[parameter.default] + "）" : "") +
          (seconds ? " 秒" : "")
      );
    }
    const constraints = parameter.constraints || {};
    if (parameter.type === "array" && constraints.max_items !== undefined) {
      const units = { images: "张", videos: "段", audios: "段", documents: "份", subjects: "个主体" };
      const unit = units[field] || "项";
      const minimum = constraints.min_items || (parameter.required ? 1 : 0);
      const maximum = constraints.max_items;
      if (maximum === 0) {
        en.push("not supported by this model");
        zh.push("当前模型不支持此素材");
      } else if (minimum === maximum) {
        en.push("exactly " + maximum + " item(s)");
        zh.push("需传入 " + maximum + " " + unit);
      } else if (minimum > 0) {
        en.push(minimum + "–" + maximum + " items");
        zh.push("支持 " + minimum + "–" + maximum + " " + unit);
      } else {
        en.push("up to " + maximum + " items");
        zh.push("最多 " + maximum + " " + unit);
      }
    }
    if (MEDIA_FIELDS.includes(field) && constraints.max_items !== 0) {
      const schemes = (constraints.url_schemes || ["https"]).join("/").toUpperCase();
      en.push("public " + schemes + " URLs");
      zh.push("链接须使用公网 " + schemes + " 地址");
      if (constraints.allow_data_url) {
        en.push("also accepts Base64 data URLs");
        zh.push("也支持 Base64 data URL");
      }
      if (constraints.allow_files) {
        en.push("also accepts multipart file uploads");
        zh.push("也支持 multipart 文件上传");
      }
      if (constraints.mime_types) {
        en.push("formats: " + constraints.mime_types.join(" / "));
        zh.push("文件格式支持 " + constraints.mime_types.join(" / "));
      }
      if (constraints.max_bytes) {
        const bytes = constraints.max_bytes;
        if ((bytes + 1) % 1048576 === 0) {
          const size = (bytes + 1) / 1048576 + " MiB";
          en.push("each file < " + size);
          zh.push("每个文件小于 " + size);
        } else {
          const size = bytes % 1048576 === 0 ? bytes / 1048576 + " MiB" : bytes + " bytes";
          en.push("each file ≤ " + size);
          zh.push("每个文件不超过 " + size.replace("bytes", "字节"));
        }
      }
    }
    if (parameter.maxLength) {
      en.push("up to " + parameter.maxLength + " characters");
      zh.push("最多 " + parameter.maxLength + " 个字符");
    }
    if (field === "subjects") {
      en.push("one English name per reference image, in the same order");
      zh.push("每张参考图对应一个英文主体名，顺序须一致");
    }
    if (field === "resolution" && profile.routes) {
      const [enMaterials, zhMaterials] = describeVideoPoolMaterials(profile);
      en.push(enMaterials);
      zh.push(zhMaterials);
    }
    if (field === "aspect_ratio" && profile.routes) {
      const ratios = describeVideoPoolRatios(profile);
      english.push("aspect_ratio: Aspect ratios by resolution: " + ratios);
      chinese.push("aspect_ratio: 各分辨率的可用画幅：" + ratios);
      continue;
    }
    english.push(field + ": " + en.join(", "));
    chinese.push(field + "：" + zh.join("，"));
  }
  if (!profile.routes && profile.max_reference_total) {
    english.push("Up to " + profile.max_reference_total + " reference materials in total");
    chinese.push("参考素材合计最多 " + profile.max_reference_total + " 项");
  }
  if (!profile.routes && profile.max_reference_video_seconds) {
    english.push("Each reference video may be up to " + profile.max_reference_video_seconds + " seconds");
    chinese.push("每段参考视频最长 " + profile.max_reference_video_seconds + " 秒");
  }
  if (profile.routes) {
    english.push("Resolution, duration, aspect ratio and references must be supported together");
    chinese.push("分辨率、时长、画幅和参考素材须有可用的组合");
  }
  if (profile.billing.billing_unit === "million_tokens") {
    english.push("First and last frames allow at most two images including reference images, and cannot be combined with reference videos");
    chinese.push("启用首尾帧时，首尾帧与参考图合计最多两张，且不能同时使用参考视频");
    if (profile.parameters.omni_reference_task_type) {
      english.push("edit and extend require reference videos; draft_task_id must be a completed gateway task owned by this account");
      chinese.push("edit 编辑与 extend 延长均须传入参考视频；draft_task_id 须为当前账户已完成的网关任务编号");
    }
  }
  english.push("Reference materials are used in the order supplied");
  chinese.push("参考素材按传入顺序使用，请保持顺序");
  return { en: english.join("; "), zh: chinese.join("；") };
}

// Catalog titles belong to public API IDs, not the vendor's route labels.
const PUBLIC_DISPLAY_NAMES = {
  "seedance-2-5-01": "Seedance 2.5 · 多参考",
  "seedance-2-5-02": "Seedance 2.5 · 人像参考",
  "seedance-2-5-03": "Seedance 2.5 · 4K",
  "seedance-2-5-04": "Seedance 2.5 · X 版",
  "seedance-2-0-01": "Seedance 2.0 满血 · 720p",
  "seedance-2-0-02": "Seedance 2.0 · 720p 多参考",
  "ty-seedance-2-0-full-933-w-720p": "Seedance 2.0 满血 · 933 W",
  "seedance-2-0-fast-933-720p": "Seedance 2.0 Fast · 933",
  "zy-seedance-2-5-discount2": "Seedance 2.5 · 720p 图生",
  "haya-seedance-2-5-xg-480p": "Seedance 2.5 XG · 480p",
  "ap-seedance-2-0-ot": "Seedance 2.0 · Token",
  "ap-seedance-2-0-ot-fast": "Seedance 2.0 Fast · Token",
  "ap-seedance-2-0-ot-mini": "Seedance 2.0 Mini · Token",
  "ap-seedance-2-5-ot": "Seedance 2.5 · Token",
  "ap-seedance-2-5-yz": "Seedance 2.5 · 1080p 按秒",
  "vp-seedance-2-5-fp": "Seedance 2.5 满血 · 1080p",
  "haya-seedance-2-5-m": "Seedance 2.5 · 全参考",
  "mg-seedance-2-5-6": "Seedance 2.5 · 图音联动",
  "vg-seedance-2-5-pa": "Seedance 2.5 满血 · 图音参考",
  "vg-seedance-2-5-wd": "Seedance 2.5 满血 · 720p",
  "vv-seedance-2-5-r2": "Seedance 2.5 · 480p",
  "vv-seedance-2-5-r2-plus": "Seedance 2.5 · 720p",
  "rs-seedance-2-0-fast": "Seedance 2.0 Fast · 720p",
  "seedance-2-0-full-933-720p": "Seedance 2.0 满血 933 · 720p",
  "seedance-2-0-mini-b1-480p": "Seedance 2.0 Mini · 480p",
  "seedance-2-0-mini-c9-933-720p": "Seedance 2.0 Mini · 720p",
  "gt-seedance-2-0-mini": "Seedance 2.0 Mini · 按秒",
  "gt-seedance-2-0-fast": "Seedance 2.0 Fast · 按秒",
  "gt-seedance-2-0-full": "Seedance 2.0 满血 · 智能时长",
  "ty-seedream-5-5": "Seedream 5.5",
  "ac-seedream-5-0-lite": "Seedream 5.0 Lite",
  "ac-seedream-5-0-pro": "Seedream 5.0 Pro · 4K",
  "ty-seedream-5-pro": "Seedream 5.0 Pro · 2K",
  "ac-image-2-5-flare": "Image 2.5 Flare · 格式可选",
  "ac-image-2-5-sunburst": "Image 2.5 Sunburst · 格式可选",
  "ac-image-2": "Image 2 · 格式可选",
  "nw-image-2": "Image 2 · 画质可调",
  "nw-image-2-5": "Image 2.5",
  "nw-image-2-6": "Image 2.6",
  "tinysnow-image-2": "Image 2 · 批量出图",
  "tinysnow-image-2-5-flare": "Image 2.5 Flare · 批量出图",
  "tinysnow-image-2-5-sunburst": "Image 2.5 Sunburst · 批量出图",
  "aj-wan-3-0": "Wan 3.0 · 多参考按秒",
  "wan-3-0-h2": "Wan 3.0 · 480p",
  "aj-wan-3-0-prime": "Wan 3.0 Prime",
  "mw-wan-3-0-s": "Wan 3.0 · 1080p 按次",
  "ap-muse-video-720-10s": "Muse Video · 720p",
  "muse-zr-700": "Muse Video · Character",
  "mf-gemini-omni-1-2-flash": "Gemini Omni 1.2 Flash",
  "mf-video-upscale": "Video Upscaler",
  "nw-nano-banana-2-1": "Nano Banana 2.1",
  "tinysnow-banana-3-1": "Nano Banana 2",
  "tinysnow-banana-3": "Nano Banana Pro",
  "sl-midjourney": "Midjourney Studio",
  "sl-midjourney-niji": "Midjourney Niji Studio",
  "ms-minimax-h3": "MiniMax H3 · Generate",
  "ms-minimax-h3-upscale": "MiniMax H3 · Regenerate",
  "minimax-h3-768p": "MiniMax H3 · 768p",
  "ad-minimax-h3-replace": "MiniMax H3 · Character Replace",
  "ad-minimax-h3-replace-multi": "MiniMax H3 · Multi-Character Replace",
  "ty-face-processing-1-0": "Face Processing 1.0",
  "mw-vidu-q4-nopeak": "Vidu Q4",
};

const MODEL_DEFAULTS = Object.fromEntries(
  Object.entries(MODEL_MAPPING).map(([publicModel, name]) => {
      const profile = MODELS[name];
      const displayName = PUBLIC_DISPLAY_NAMES[publicModel] ?? publicModel;
      const defaults = { displayName, description: modelDescription(profile), icon: profile.icon, tags: [...new Set(profile.tags)] };
      if (/seedance|seedream|face-processing/.test(name)) defaults.vendor = "ByteDance";
      else if (/minimax|h3video/.test(name)) defaults.vendor = "Minimax";
      else if (/wan-/.test(name)) defaults.vendor = "Wan";
      else if (/vidu/.test(name)) defaults.vendor = "Vidu";
      else if (/banana|gemini|omni/.test(name)) defaults.vendor = "Google";
      else if (/midjourney/.test(name)) defaults.vendor = "Midjourney";
      else if (/image-2|^image2$/.test(name)) defaults.vendor = "OpenAI";
      const expression = defaultExpression(profile);
      if (expression) defaults.billingExpr = expression;
      return [publicModel, defaults];
  })
);

export const meta = {
  apiVersion: 1,
  key: "lecapi",
  name: "LecAPI",
  icon: "text:LE",
  version: "1.9.0",
  author: { name: "QuantumNous" },
  website: "https://api.lecapi.net/docs",
  baseUrl: "https://api.lecapi.net",
  description: {
    en: "LecAPI image and video generation with model parameters, reference materials and published default pricing",
    zh: "LecAPI 图片与视频生成，包含模型参数、参考素材要求及公开默认定价",
  },
  requiredCapabilities: ["submit-no-retry@1", "media-duration@1", "lecapi-video-pool@1"],
  upstreams: ["vendor", "new_api"],
  fetchMode: "per_task",
  models: Object.keys(MODEL_MAPPING),
  modelDefaults: MODEL_DEFAULTS,
  usageSchema: VIDEO_USAGE_SCHEMA,
  usageProfiles: Object.entries(MODEL_MAPPING).map(([publicModel, name]) => {
      const profile = MODELS[name];
      const tokenPriced = profile.billing.billing_unit === "million_tokens";
      const schema = { ...(profile.kind === "image" ? IMAGE_USAGE_SCHEMA : tokenPriced ? TOKEN_VIDEO_USAGE_SCHEMA : VIDEO_USAGE_SCHEMA) };
      const resolution = profile.parameters.resolution;
      const resolutions = resolution?.values || [];
      schema.resolution = { ...schema.resolution, enum: resolution?.default ? resolutions : ["unspecified", ...resolutions] };
      const usageProfile = { models: [publicModel], schema };
      if (tokenPriced) {
        usageProfile.examples = [
          {
            label: resolution.default + " · 4s",
            facts: {
              requests: 1,
              seconds: 4,
              reference_seconds: 0,
              extra_images: 0,
              resolution: resolution.default,
              duration_tier: "short",
              video_input: "none",
              tokens: profile.billing.tokens_per_second[resolution.default] * 4,
            },
          },
        ];
      }
      return usageProfile;
  }),
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

function closestImageAspectRatio(width, height, profile) {
  const ratios = ["1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "21:9"];
  if (["lec-sl-midjourney", "lec-sl-midjourney-niji"].includes(profile.vendorModel)) ratios.push("4:5", "5:4");
  let closest,
    closestDistance = Infinity;
  for (const ratio of ratios) {
    if (!profile.parameters.aspect_ratio.values.includes(ratio)) continue;
    const [numerator, denominator] = ratio.split(":").map(Number);
    // Relative distance treats landscape and portrait canvases symmetrically.
    const distance = Math.abs(Math.log(width / height / (numerator / denominator)));
    if (distance < closestDistance) {
      closest = ratio;
      closestDistance = distance;
    }
  }
  if (!closest) throw new Error("model does not support a canvas aspect ratio");
  return closest;
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
function submission(ctx, profile = modelProfile(ctx)) {
  const body = { model: ctx.upstream?.kind === "new_api" ? profile.publicModel : profile.vendorModel };
  if (profile.retired) throw new Error(profile.publicModel + " is no longer available on LecAPI");
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
      let value = scope[field];
      if (CONTAINERS.includes(field)) continue;
      if (field === "model") {
        const names = [ctx.model, ctx.upstreamModel, profile.publicModel];
        if (!profile.poolRoute) names.push(profile.vendorModel);
        if (
          typeof value !== "string" ||
          !names.some((name) => typeof name === "string" && name.toLowerCase() === value.toLowerCase())
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
      if (profile.vendorModel === "lec-vw-seedance-2-0-fast" && target === "last_frame") throw new Error("last_frame is not supported by this model");
      if (!profile.poolRoute && ["first_frame", "last_frame"].includes(target) && !profile.parameters[target]) target = "images";
      if (target === "size" && !profile.parameters.size && typeof value === "string") {
        value = value.trim();
        if (/^\d+:\d+$/.test(value)) target = "aspect_ratio";
        else if (/^(?:\d+p|[124]k)$/i.test(value)) target = "resolution";
        else if (profile.kind === "image" && profile.parameters.aspect_ratio) {
          const dimensions = /^(\d+)\s*[x×]\s*(\d+)$/i.exec(value);
          if (!dimensions) throw new Error("size must contain pixel dimensions, a supported aspect ratio or a resolution");
          const width = Number(dimensions[1]),
            height = Number(dimensions[2]);
          if (!Number.isSafeInteger(width) || !Number.isSafeInteger(height) || width <= 0 || height <= 0)
            throw new Error("size must contain positive safe integer pixel dimensions");
          // Snap the canvas shape without changing resolution or its billing tier.
          value = closestImageAspectRatio(width, height, profile);
          target = "aspect_ratio";
        }
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
  if (["lec-sl-midjourney", "lec-sl-midjourney-niji"].includes(profile.vendorModel) && body.action === undefined) {
    // Infer the canvas action from every reference alias and nested parameter scope.
    const referenceCount = media.images?.length || 0;
    body.action = referenceCount === 0 ? "imagine" : referenceCount === 1 ? "edits" : "blend";
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
  if (profile.billing.billing_unit === "million_tokens") {
    const frameCount = (body.first_frame ? 1 : 0) + (body.last_frame ? 1 : 0);
    if (frameCount && (videoCount || imageCount + frameCount > 2))
      throw new Error("first and last frames support at most two combined images and cannot be combined with reference videos");
    if (["edit", "extend"].includes(body.omni_reference_task_type) && !videoCount)
      throw new Error(body.omni_reference_task_type + " requires reference videos");
  }
  if (profile.max_reference_total && imageCount + videoCount + audioCount + (media.documents?.length || 0) > profile.max_reference_total)
    throw new Error("too many combined reference materials");
  if (profile.vendorModel === "lec-mg-seedance-2-5-6" && audioCount && !imageCount) throw new Error("audio references require at least one image");
  if (profile.vendorModel === "lec-gt-seedance-2-0-full") {
    if ((videoCount || audioCount) && !imageCount) throw new Error("video or audio references require at least one image");
    if (body.mode === "frames2video" && (imageCount !== 2 || videoCount || audioCount))
      throw new Error("frames2video requires exactly two images and no video or audio references");
  }
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
  // Smart duration reserves the longest supported output while preserving -1 upstream.
  let seconds = body.duration === -1 ? profile.parameters.duration.max : body.duration || 0,
    referenceSeconds = 0;
  if (
    profile.source_duration ||
    profile.bill_reference_videos ||
    (profile.vendorModel === "lec-ms-minimax-h3" && videoCount) ||
    profile.max_reference_video_seconds
  ) {
    for (const reference of media.videos || []) {
      const duration = typeof reference === "object" ? utils.mediaDuration({ fileRef: reference.__fileRef }) : utils.mediaDuration({ url: reference });
      if (
        !Number.isFinite(duration) ||
        duration <= 0 ||
        duration > 3600 ||
        (profile.max_reference_video_seconds && duration > profile.max_reference_video_seconds)
      )
        throw new Error("reference video duration is outside the supported range");
      // AJ rounds each reference clip separately before adding it to output time.
      referenceSeconds += profile.bill_reference_videos ? Math.ceil(duration) : duration;
    }
    referenceSeconds = Math.ceil(referenceSeconds);
    if (referenceSeconds > 3600) throw new Error("combined reference video duration exceeds 3600 seconds");
    if (profile.source_duration) seconds = referenceSeconds;
    if (profile.vendorModel === "lec-ms-minimax-h3" && videoCount) {
      if (referenceSeconds < 1 || referenceSeconds > 15) throw new Error("reference videos must total at most 15 seconds");
      if (ctx.requestBody.reference_video_seconds !== undefined && body.reference_video_seconds !== referenceSeconds)
        throw new Error("reference_video_seconds must match the measured videos");
      body.reference_video_seconds = referenceSeconds;
    }
  }
  if (profile.vendorModel === "lec-ms-minimax-h3" && !videoCount) delete body.reference_video_seconds;
  for (const field of ["source_task_id", "original_task_id", "draft_task_id"]) {
    if (body[field] === undefined) continue;
    const origin = (ctx.originTasks || []).find((task) => task.taskId === body[field]);
    if (!origin || origin.status !== "SUCCESS") throw new Error(field + " must reference a completed task owned by this account");
    const source = responseObject(origin.data);
    if (field === "source_task_id") {
      if (vendorModelName(source.model) !== "lec-ms-minimax-h3" || String(source.resolution || source.size).toUpperCase() !== "768P")
        throw new Error("source_task_id must reference a completed 768P MiniMax-H3 MS generation");
      seconds = Number(source.seconds ?? source.duration);
      if (!Number.isInteger(seconds) || seconds < 4 || seconds > 15) throw new Error("source_task_id must have a duration between 4 and 15 seconds");
    } else if (vendorModelName(source.model) !== profile.vendorModel) throw new Error(field + " must reference the same upstream model");
    body[field] = origin.upstreamTaskId;
  }
  if (!Number.isInteger(seconds) || seconds < 0 || seconds > 3600) throw new Error("video duration must be between 0 and 3600 seconds");
  let tokens = 0;
  if (profile.billing.billing_unit === "million_tokens") {
    const rate = profile.billing.tokens_per_second[body.resolution];
    // LecAPI freezes this estimate and later settles from total_tokens. Reference
    // video requests add four seconds or 80%, whichever reserves more tokens.
    tokens = videoCount ? Math.max((seconds + 4) * rate, (seconds * rate * 18) / 10) : seconds * rate;
    if (!Number.isSafeInteger(tokens) || tokens <= 0 || tokens > 2147483647)
      throw new Error("estimated video tokens exceed the supported range");
  }
  return {
    profile: profile,
    body: body,
    seconds: seconds,
    referenceSeconds: referenceSeconds,
    imageCount: imageCount,
    videoCount: videoCount,
    outputCount: requestedCount || body.n || 1,
    files: files,
    tokens: tokens,
  };
}

function videoPoolSubmission(ctx, profile, includeAlternatives) {
  const resolutionParameter = profile.parameters.resolution;
  let selected;
  const routeAlternatives = [];
  for (const vendorModel of profile.routes) {
    const candidate = MODELS[vendorModel];
    const fixed = FIXED_VIDEO_OUTPUTS[vendorModel] || {};
    const duration = candidate.parameters.duration || (fixed.duration ? { type: "integer", values: [fixed.duration] } : null);
    let defaultDuration = duration?.default ?? fixed.duration;
    if (duration) {
      try {
        defaultDuration = parameterValue(profile.parameters.duration.default, duration, "duration");
      } catch {
        // This candidate can still use its own default if duration is omitted.
      }
    }
    try {
      const request = submission(ctx, {
        ...candidate,
        vendorModel,
        publicModel: profile.publicModel,
        poolRoute: true,
        parameters: {
          ...candidate.parameters,
          aspect_ratio: { ...candidate.parameters.aspect_ratio, default: profile.parameters.aspect_ratio.default },
          resolution: { ...resolutionParameter, values: fixed.resolution ? [fixed.resolution] : candidate.parameters.resolution?.values ?? Object.keys(candidate.billing.resolution_prices) },
          ...(duration ? { duration: { ...duration, default: defaultDuration } } : {}),
        },
      });
      const resolution = request.body.resolution;
      if (ctx.upstream?.kind !== "new_api") {
        if (fixed.resolution) delete request.body.resolution;
        if (fixed.duration) delete request.body.duration;
      }
      if (!selected) {
        selected = { ...request, profile, resolution };
        if (!includeAlternatives || ctx.upstream?.kind === "new_api") return selected;
      } else if (request.seconds === selected.seconds && request.videoCount === selected.videoCount && resolution === selected.resolution) {
        // Alternatives must share every reservation fact and effective output
        // setting; otherwise a retry could change the request or its price.
        routeAlternatives.push(request.body);
      }
    } catch {
      // A candidate can be excluded before submit, never by downgrading its input.
    }
  }
  if (selected) return { ...selected, routeAlternatives };
  throw new Error("No LecAPI route supports this combination of resolution, duration and reference materials");
}

export function listLecAPIVideoRoutes() {
  return Object.keys(VIDEO_POOLS).flatMap((model) => (MODELS[model].routes ?? []).map((route) => ({
    model,
    route,
    resolutions: FIXED_VIDEO_OUTPUTS[route]?.resolution ? [FIXED_VIDEO_OUTPUTS[route].resolution] : MODELS[route].parameters.resolution?.values ?? Object.keys(MODELS[route].billing.resolution_prices),
  })));
}

function resolveSubmission(ctx, includeAlternatives = false) {
  const profile = modelProfile(ctx);
  if (profile.poolUnavailable) throw new Error("No LecAPI route is currently available for this model");
  if (!profile.retired && (INTERNAL_VIDEO_MODELS.has(profile.vendorModel) || INTERNAL_VIDEO_MODELS.has(vendorModelName(ctx.model))))
    throw new Error("Use a public LecAPI video entry instead of a retired route model name");
  return profile.routes ? videoPoolSubmission(ctx, profile, includeAlternatives) : submission(ctx, profile);
}

export function buildSubmitRequest(ctx) {
  const request = resolveSubmission(ctx, true);
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
    ...(request.routeAlternatives ? { lecapiRouteAlternatives: request.routeAlternatives } : {}),
  };
}

export function extractUsage(ctx) {
  const request = resolveSubmission(ctx);
  if (ctx.usagePurpose === "billing_ratios")
    return request.profile.kind === "image" && request.profile.billing.price_per_image !== undefined ? { image_count: request.outputCount } : { requests: 1 };
  if (request.profile.kind === "image")
    return { requests: 1, image_count: request.outputCount, resolution: request.body.resolution || "unspecified", action: request.body.action || "imagine" };
  const facts = {
    requests: 1,
    seconds: request.seconds,
    reference_seconds: request.profile.vendorModel === "lec-ms-minimax-h3" || request.profile.bill_reference_videos ? request.referenceSeconds : 0,
    extra_images: request.profile.vendorModel === "lec-ms-minimax-h3" ? Math.max(request.imageCount - 5, 0) : 0,
    resolution: request.resolution || request.body.resolution || "unspecified",
    duration_tier: request.seconds > 15 ? "long" : "short",
    video_input: request.videoCount > 0 ? "video" : "none",
  };
  if (request.profile.billing.billing_unit === "million_tokens") facts.tokens = request.tokens;
  return facts;
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

function signedURLTimestamp(value) {
  const numeric = Number(value);
  if (Number.isFinite(numeric)) return numeric >= 1000000000 ? Math.floor(numeric > 1000000000000 ? numeric / 1000 : numeric) : 0;
  const compact = /^(\d{4})(\d{2})(\d{2})T(\d{2})(\d{2})(\d{2})Z$/.exec(String(value));
  if (compact) return Math.floor(Date.UTC(Number(compact[1]), Number(compact[2]) - 1, Number(compact[3]), Number(compact[4]), Number(compact[5]), Number(compact[6])) / 1000);
  const parsed = Date.parse(String(value));
  return Number.isFinite(parsed) ? Math.floor(parsed / 1000) : 0;
}

function signedURLExpiration(value) {
  if (typeof value !== "string") return 0;
  const queryStart = value.indexOf("?");
  if (queryStart < 0) return 0;
  const query = value.slice(queryStart + 1).split("#", 1)[0];
  const parameters = {};
  for (const pair of query.split("&")) {
    const separator = pair.indexOf("=");
    if (separator < 0) continue;
    try {
      const key = decodeURIComponent(pair.slice(0, separator).replace(/\+/g, " ")).toLowerCase();
      if (parameters[key] === undefined) parameters[key] = decodeURIComponent(pair.slice(separator + 1).replace(/\+/g, " "));
    } catch {
      continue;
    }
  }

  for (const key of ["expires", "expire", "expiration", "expires_at", "expiresat", "expiry", "valid_until"]) {
    if (parameters[key] !== undefined) {
      const expiration = signedURLTimestamp(parameters[key]);
      if (expiration > 0) return expiration;
    }
  }

  for (const prefix of ["x-amz", "x-goog", "x-oss"]) {
    const signedAt = signedURLTimestamp(parameters[prefix + "-date"] || "");
    const duration = Number(parameters[prefix + "-expires"]);
    if (signedAt > 0 && Number.isFinite(duration) && duration > 0) return signedAt + duration;
  }

  for (const key of ["q-sign-time", "q-key-time"]) {
    const interval = parameters[key]?.split(";");
    if (interval?.length === 2) {
      const expiration = signedURLTimestamp(interval[1]);
      if (expiration > 0) return expiration;
    }
  }
  return 0;
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
  if (profile.routes?.includes(ctx.lecapiSubmitRoute)) {
    state.lecapiSubmitRoute = ctx.lecapiSubmitRoute;
    if (typeof ctx.lecapiRetryURL === "string" && Array.isArray(ctx.lecapiRetryCandidates)) {
      state.lecapiRetryURL = ctx.lecapiRetryURL;
      state.lecapiRetryCandidates = ctx.lecapiRetryCandidates;
      if (ctx.lecapiRetryInFlight) state.lecapiRetryInFlight = ctx.lecapiRetryInFlight;
    }
  }
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
  if (profile.billing.billing_unit === "million_tokens") {
    const value = body.usage?.total_tokens ?? body.total_tokens;
    if (typeof value !== "number" && (typeof value !== "string" || !/^\d+$/.test(value))) return {};
    const tokens = Number(value);
    // Missing or malformed usage keeps the frozen estimate. Explicit zero is
    // real usage, and completion cannot change the frozen input or resolution.
    return Number.isSafeInteger(tokens) && tokens >= 0 && tokens <= 2147483647 ? { tokens: tokens } : {};
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
    return entries
      .filter((entry) => entry.url)
      .map((entry, index) => ({ key: "image-" + (index + 1), type: "image", mimeType: "image/png" }))
      .concat(
        entries
          .filter((entry) => !entry.url && entry.b64_json)
          .map((entry, index) => ({ key: "image-b64-" + (index + 1), type: "image", mimeType: "image/png" }))
      );
  return resultURL(task.data) ? [{ key: "video", type: "video", mimeType: "video/mp4" }] : [];
}

export function buildContentRequest(ctx) {
  const image = /^image-(b64-)?([1-9]\d*)$/.exec(ctx.artifactKey);
  if (!image && ctx.artifactKey !== "video") throw new Error("artifact_not_found");
  let data = ctx.data;
  // Before state was persisted, synchronous images used a locally generated
  // id. Only models with an asynchronous task API can refresh those old rows.
  const profile = modelProfile(ctx);
  const queryable = ctx.state?.queryable ?? (profile.kind === "video" || ASYNC_IMAGE_MODELS.includes(profile.vendorModel));
  // Prefer the signed URL's actual expiration; completion age alone can cause
  // unnecessary queries and rate limits while OSS archives a still-valid URL.
  const completedAt = Number(responseObject(data).completed_at);
  const now = utils.unixNow();
  const completedAge = now - completedAt;
  const sourceURL = resultURL(data);
  const sourceExpiresAt = signedURLExpiration(sourceURL);
  const freshSignedURL = sourceExpiresAt > 0 && sourceExpiresAt - now > 30;
  const recentUnsignedResult = sourceExpiresAt === 0 && completedAt > 0 && completedAge >= 0 && completedAge < 60;
  const freshResult = !!sourceURL && (freshSignedURL || recentUnsignedResult);
  if (ctx.upstreamTaskId && queryable && !freshResult) {
    const query = buildQueryRequest({ ...ctx, taskId: ctx.upstreamTaskId });
    const refreshed = utils.fetch(query);
    if (refreshed.status < 200 || refreshed.status >= 300) throw new Error("LecAPI result refresh failed (HTTP " + refreshed.status + ")");
    const body = responseObject(refreshed.body);
    const id = refreshed.body.object === "lecapi.task" ? refreshed.body.id : body.id || body.task_id || body.metadata?.task_id;
    if (id !== ctx.upstreamTaskId) throw new Error("LecAPI result refresh returned a different task id");
    const result = parseTaskResult({ ...ctx, taskId: ctx.upstreamTaskId }, refreshed.body);
    if (result.status !== "SUCCESS") throw new Error("LecAPI result is not available");
    data = refreshed.body;
  }
  const entry = image ? imageEntries(data, true).filter((item) => (image[1] ? !item.url && item.b64_json : item.url))[Number(image[2]) - 1] : undefined;
  const url = image ? entry?.url || (entry?.b64_json ? "data:image/png;base64," + entry.b64_json : "") : resultURL(data);
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
  if (vendorModel && MODELS[vendorModel].retired) throw new Error(PUBLIC_MODEL_NAMES[vendorModel] + " is no longer available on LecAPI");
  if (INTERNAL_VIDEO_MODELS.has(vendorModel) || INTERNAL_VIDEO_MODELS.has(vendorModelName(body.model)))
    throw new Error("Use a public LecAPI video entry instead of a retired route model name");
  if (vendorModel && MODELS[vendorModel].kind !== kind) throw new Error("model does not support this media endpoint");
  // Image response_format belongs to the host presenter. The vendor still
  // receives its documented default; the host converts URLs when requested.
  if (!nativeRoute && kind === "image") delete body.response_format;
  const origins = [];
  for (const scope of parameterScopes(body))
    for (const field of ["original_task_id", "source_task_id", "draft_task_id"]) {
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
  const model = ctx.model || task.properties?.origin_model_name;
  const mapped = vendorModelName(task.properties?.upstream_model_name || model);
  const profile = MODELS[mapped];
  const output = publicResponse(task.data || {}, profile?.routes || profile?.historicalPool ? model : "");
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
function publicResponse(value, poolModel = "") {
  if (Array.isArray(value)) return value.map((item) => publicResponse(item, poolModel));
  if (!value || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value).map(([key, item]) => [
      key,
      ["model", "model_name", "origin_model_name", "upstream_model_name"].includes(key) && typeof item === "string"
        ? (poolModel && MODELS[vendorModelName(item)]?.kind === "video" ? poolModel : PUBLIC_MODEL_NAMES[vendorModelName(item)]) || item
        : publicResponse(item, poolModel),
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
      const mapped = vendorModelName(task.properties?.upstream_model_name || ctx.model);
      const profile = MODELS[mapped];
      const body = publicResponse(responseObject(task.data || {}), profile?.routes || profile?.historicalPool ? ctx.model : "");
      if (typeof ctx.model === "string") body.model = PUBLIC_MODEL_NAMES[vendorModelName(ctx.model)] || ctx.model;
      return body;
    },
  },
  openai_image: {
    decodeRequest: (ctx) => decodeRequest(ctx, "image", false),
    render: (ctx, task) => ({ data: imageEntries(task.data, true), ...(task.data?.created ? { created: task.data.created } : {}) }),
  },
};
