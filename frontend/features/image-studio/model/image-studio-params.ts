import type { ImageStudioTaskType } from "@/shared/api/image-studio.types";
import type { PublicModelDTO } from "@/shared/api/model.types";

export type ImageStudioParamFamily = "openai" | "google" | "xai" | "generic";
export type ImageStudioQuality = "auto" | "low" | "medium" | "high";
export type ImageStudioOutputFormat = "png" | "jpeg" | "webp";
export type ImageStudioBackground = "auto" | "transparent" | "opaque";
export type ImageStudioModeration = "auto" | "low";
export type ImageStudioInputFidelity = "auto" | "low" | "high";
export type ImageStudioSizeTier = "1K" | "2K" | "4K";
export type ImageStudioPresetRatio = "1:1" | "3:2" | "2:3" | "16:9" | "9:16" | "4:3" | "3:4" | "21:9";

export type ImageStudioParams = {
  size: string;
  quality: ImageStudioQuality;
  outputFormat: ImageStudioOutputFormat;
  outputCompression: number | null;
  background: ImageStudioBackground;
  moderation: ImageStudioModeration;
  n: number;
  partialImages: number;
  inputFidelity: ImageStudioInputFidelity;
};

export type ImageStudioParamCapabilities = {
  family: ImageStudioParamFamily;
  supportsQuality: boolean;
  supportsOutputFormat: boolean;
  supportsCompression: boolean;
  supportsBackground: boolean;
  supportsModeration: boolean;
  supportsInputFidelity: boolean;
  supportsCount: boolean;
  supportsPartialImages: boolean;
};

export const IMAGE_STUDIO_SIZE_TIERS: ImageStudioSizeTier[] = ["1K", "2K", "4K"];
export const IMAGE_STUDIO_PRESET_RATIOS: ImageStudioPresetRatio[] = ["1:1", "3:2", "2:3", "16:9", "9:16", "4:3", "3:4", "21:9"];
export const IMAGE_STUDIO_QUALITY_VALUES: ImageStudioQuality[] = ["auto", "low", "medium", "high"];
export const IMAGE_STUDIO_OUTPUT_FORMAT_VALUES: ImageStudioOutputFormat[] = ["png", "jpeg", "webp"];
export const IMAGE_STUDIO_BACKGROUND_VALUES: ImageStudioBackground[] = ["auto", "transparent", "opaque"];
export const IMAGE_STUDIO_MODERATION_VALUES: ImageStudioModeration[] = ["auto", "low"];
export const IMAGE_STUDIO_INPUT_FIDELITY_VALUES: ImageStudioInputFidelity[] = ["auto", "low", "high"];

export const DEFAULT_IMAGE_STUDIO_PARAMS: ImageStudioParams = {
  size: "auto",
  quality: "auto",
  outputFormat: "png",
  outputCompression: null,
  background: "auto",
  moderation: "auto",
  n: 1,
  partialImages: 0,
  inputFidelity: "auto",
};

const SIZE_PATTERN = /^\s*(\d+)\s*[xX]\s*(\d+)\s*$/;
const RATIO_PATTERN = /^\s*(\d+(?:\.\d+)?)\s*[:xX]\s*(\d+(?:\.\d+)?)\s*$/;
const SIZE_MULTIPLE = 16;
const MAX_EDGE = 3840;
const MAX_ASPECT_RATIO = 3;
const MIN_PIXELS = 655_360;
const MAX_PIXELS = 8_294_400;
const MAX_OUTPUT_IMAGES = 10;
const MAX_PARTIAL_IMAGES = 3;

const TIER_PIXEL_BUDGET: Record<ImageStudioSizeTier, number> = {
  "1K": 1_572_864,
  "2K": 4_194_304,
  "4K": MAX_PIXELS,
};

const COMMON_SIZE_PRESETS: Record<ImageStudioSizeTier, Record<ImageStudioPresetRatio, string>> = {
  "1K": {
    "1:1": "1024x1024",
    "3:2": "1536x1024",
    "2:3": "1024x1536",
    "16:9": "1280x720",
    "9:16": "720x1280",
    "4:3": "1024x768",
    "3:4": "768x1024",
    "21:9": "1280x544",
  },
  "2K": {
    "1:1": "2048x2048",
    "3:2": "2160x1440",
    "2:3": "1440x2160",
    "16:9": "2560x1440",
    "9:16": "1440x2560",
    "4:3": "2048x1536",
    "3:4": "1536x2048",
    "21:9": "2560x1088",
  },
  "4K": {
    "1:1": "2880x2880",
    "3:2": "3456x2304",
    "2:3": "2304x3456",
    "16:9": "3840x2160",
    "9:16": "2160x3840",
    "4:3": "3200x2400",
    "3:4": "2400x3200",
    "21:9": "3840x1600",
  },
};

function clampNumber(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function roundToMultiple(value: number, multiple: number): number {
  return Math.max(multiple, Math.round(value / multiple) * multiple);
}

function floorToMultiple(value: number, multiple: number): number {
  return Math.max(multiple, Math.floor(value / multiple) * multiple);
}

function ceilToMultiple(value: number, multiple: number): number {
  return Math.max(multiple, Math.ceil(value / multiple) * multiple);
}

function normalizeDimensions(width: number, height: number): { width: number; height: number } {
  let normalizedWidth = roundToMultiple(width, SIZE_MULTIPLE);
  let normalizedHeight = roundToMultiple(height, SIZE_MULTIPLE);

  const scaleToFit = (scale: number) => {
    normalizedWidth = floorToMultiple(normalizedWidth * scale, SIZE_MULTIPLE);
    normalizedHeight = floorToMultiple(normalizedHeight * scale, SIZE_MULTIPLE);
  };

  const scaleToFill = (scale: number) => {
    normalizedWidth = ceilToMultiple(normalizedWidth * scale, SIZE_MULTIPLE);
    normalizedHeight = ceilToMultiple(normalizedHeight * scale, SIZE_MULTIPLE);
  };

  for (let index = 0; index < 4; index += 1) {
    const maxEdge = Math.max(normalizedWidth, normalizedHeight);
    if (maxEdge > MAX_EDGE) {
      scaleToFit(MAX_EDGE / maxEdge);
    }

    if (normalizedWidth / normalizedHeight > MAX_ASPECT_RATIO) {
      normalizedWidth = floorToMultiple(normalizedHeight * MAX_ASPECT_RATIO, SIZE_MULTIPLE);
    } else if (normalizedHeight / normalizedWidth > MAX_ASPECT_RATIO) {
      normalizedHeight = floorToMultiple(normalizedWidth * MAX_ASPECT_RATIO, SIZE_MULTIPLE);
    }

    const pixels = normalizedWidth * normalizedHeight;
    if (pixels > MAX_PIXELS) {
      scaleToFit(Math.sqrt(MAX_PIXELS / pixels));
    } else if (pixels < MIN_PIXELS) {
      scaleToFill(Math.sqrt(MIN_PIXELS / pixels));
    }
  }

  return { width: normalizedWidth, height: normalizedHeight };
}

function parseSizeValue(value: string): { width: number; height: number } | null {
  const match = value.match(SIZE_PATTERN);
  if (!match) return null;
  const width = Number(match[1]);
  const height = Number(match[2]);
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return null;
  return { width, height };
}

export function parseImageStudioRatio(value: string): { width: number; height: number } | null {
  const match = value.match(RATIO_PATTERN);
  if (!match) return null;
  const width = Number(match[1]);
  const height = Number(match[2]);
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return null;
  return { width, height };
}

function gcd(a: number, b: number): number {
  return b === 0 ? Math.abs(a) : gcd(b, a % b);
}

function formatRatio(width: number, height: number): string {
  const roundedWidth = Math.round(width);
  const roundedHeight = Math.round(height);
  if (!Number.isFinite(roundedWidth) || !Number.isFinite(roundedHeight) || roundedWidth <= 0 || roundedHeight <= 0) {
    return "";
  }
  const divisor = gcd(roundedWidth, roundedHeight);
  return `${roundedWidth / divisor}:${roundedHeight / divisor}`;
}

function presetRatioKey(width: number, height: number): ImageStudioPresetRatio | null {
  const ratio = formatRatio(width, height);
  return ratio in COMMON_SIZE_PRESETS["1K"] ? (ratio as ImageStudioPresetRatio) : null;
}

export function normalizeImageStudioSize(value: string): string {
  const trimmed = value.trim();
  if (!trimmed || trimmed.toLowerCase() === "auto") return "auto";
  const parsedSize = parseSizeValue(trimmed);
  if (!parsedSize) return trimmed;
  const normalized = normalizeDimensions(parsedSize.width, parsedSize.height);
  return `${normalized.width}x${normalized.height}`;
}

export function calculateImageStudioSize(tier: ImageStudioSizeTier, ratio: string): string | null {
  const parsed = parseImageStudioRatio(ratio);
  if (!parsed) return null;

  const preset = presetRatioKey(parsed.width, parsed.height);
  if (preset) return COMMON_SIZE_PRESETS[tier][preset];

  const targetRatio = parsed.width / parsed.height;
  const pixelBudget = TIER_PIXEL_BUDGET[tier];
  let bestWidth = 0;
  let bestHeight = 0;
  let bestPixels = 0;

  for (let width = SIZE_MULTIPLE; width <= MAX_EDGE; width += SIZE_MULTIPLE) {
    const idealHeight = width / targetRatio;
    const candidates = [
      Math.floor(idealHeight / SIZE_MULTIPLE) * SIZE_MULTIPLE,
      Math.ceil(idealHeight / SIZE_MULTIPLE) * SIZE_MULTIPLE,
    ];

    for (const height of candidates) {
      if (height < SIZE_MULTIPLE || height > MAX_EDGE) continue;
      const pixels = width * height;
      if (pixels > pixelBudget || pixels < MIN_PIXELS) continue;
      if (Math.max(width / height, height / width) > MAX_ASPECT_RATIO) continue;

      const actualRatio = width / height;
      const ratioError = Math.abs(actualRatio - targetRatio) / targetRatio;
      if (ratioError > 0.01) continue;

      if (pixels > bestPixels) {
        bestPixels = pixels;
        bestWidth = width;
        bestHeight = height;
      }
    }
  }

  return bestPixels === 0 ? null : `${bestWidth}x${bestHeight}`;
}

export function imageStudioSizeAspectRatio(size: string): string {
  const normalized = normalizeImageStudioSize(size);
  if (normalized === "auto") return "";
  const parsedSize = parseSizeValue(normalized);
  if (parsedSize) return formatRatio(parsedSize.width, parsedSize.height);
  const parsedRatio = parseImageStudioRatio(normalized);
  if (parsedRatio) return formatRatio(parsedRatio.width, parsedRatio.height);
  return "";
}

export function imageStudioSizeTier(size: string): ImageStudioSizeTier {
  const normalized = normalizeImageStudioSize(size);
  for (const tier of IMAGE_STUDIO_SIZE_TIERS) {
    for (const preset of Object.values(COMMON_SIZE_PRESETS[tier])) {
      if (preset === normalized) return tier;
    }
  }
  const parsed = parseSizeValue(normalized);
  if (!parsed) return "1K";
  const pixels = parsed.width * parsed.height;
  if (pixels <= TIER_PIXEL_BUDGET["1K"]) return "1K";
  if (pixels <= TIER_PIXEL_BUDGET["2K"]) return "2K";
  return "4K";
}

export function imageStudioSizeLabelValue(size: string): string {
  const normalized = normalizeImageStudioSize(size);
  if (normalized === "auto") return "auto";
  const ratio = imageStudioSizeAspectRatio(normalized);
  return ratio ? `${normalized} - ${ratio}` : normalized;
}

function parseStringArrayJSON(raw: string | null | undefined): string[] {
  if (!raw?.trim()) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed.map((item) => (typeof item === "string" ? item.trim() : "")).filter(Boolean);
  } catch {
    return [];
  }
}

export function imageStudioModelProtocols(model: PublicModelDTO | null | undefined): string[] {
  return parseStringArrayJSON(model?.protocolsJSON);
}

export function resolveImageStudioParamFamily(model: PublicModelDTO | null | undefined, taskType: ImageStudioTaskType): ImageStudioParamFamily {
  const protocols = imageStudioModelProtocols(model);
  const has = (protocol: string) => protocols.includes(protocol);

  if (taskType === "image_edit") {
    if (has("openai_image_edits")) return "openai";
    if (has("xai_image_edits")) return "xai";
    if (has("google_image_generation")) return "google";
  } else {
    if (has("openai_image_generations")) return "openai";
    if (has("xai_image")) return "xai";
    if (has("google_image_generation")) return "google";
  }

  if (has("openai_image_generations") || has("openai_image_edits")) return "openai";
  if (has("xai_image") || has("xai_image_edits")) return "xai";
  if (has("google_image_generation")) return "google";
  return "generic";
}

export function resolveImageStudioParamCapabilities(
  model: PublicModelDTO | null | undefined,
  taskType: ImageStudioTaskType,
): ImageStudioParamCapabilities {
  const family = resolveImageStudioParamFamily(model, taskType);
  return {
    family,
    supportsQuality: family === "openai" || family === "generic",
    supportsOutputFormat: family === "openai" || family === "generic",
    supportsCompression: family === "openai" || family === "generic",
    supportsBackground: family === "openai",
    supportsModeration: family === "openai" && taskType !== "image_edit",
    supportsInputFidelity: family === "openai" && taskType === "image_edit",
    supportsCount: family === "openai" || family === "xai" || family === "generic",
    supportsPartialImages: family === "openai",
  };
}

function parseRecord(raw: string | null | undefined): Record<string, unknown> {
  if (!raw?.trim()) return {};
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {};
    return parsed as Record<string, unknown>;
  } catch {
    return {};
  }
}

function clonePlainValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(clonePlainValue);
  }
  if (value && typeof value === "object") {
    const next: Record<string, unknown> = {};
    for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
      next[key] = clonePlainValue(item);
    }
    return next;
  }
  return value;
}

export function parseImageStudioModelDefaultOptions(model: PublicModelDTO | null | undefined): Record<string, unknown> {
  const capabilities = parseRecord(model?.capabilitiesJSON);
  const defaults = capabilities.defaultOptions;
  if (!defaults || typeof defaults !== "object" || Array.isArray(defaults)) return {};
  return clonePlainValue(defaults) as Record<string, unknown>;
}

function stringOption(options: Record<string, unknown>, ...keys: string[]): string {
  for (const key of keys) {
    const value = options[key];
    if (typeof value === "string" && value.trim()) return value.trim();
  }
  return "";
}

function numberOption(options: Record<string, unknown>, ...keys: string[]): number | null {
  for (const key of keys) {
    const value = options[key];
    const numeric = typeof value === "number" ? value : typeof value === "string" && value.trim() ? Number(value) : NaN;
    if (Number.isFinite(numeric)) return numeric;
  }
  return null;
}

function mapOption(options: Record<string, unknown>, key: string): Record<string, unknown> {
  const value = options[key];
  if (!value || typeof value !== "object" || Array.isArray(value)) return {};
  return value as Record<string, unknown>;
}

function nestedStringOption(options: Record<string, unknown>, path: string[]): string {
  let current: Record<string, unknown> = options;
  for (let index = 0; index < path.length; index += 1) {
    const segment = path[index];
    const value = current[segment];
    if (index === path.length - 1) {
      return typeof value === "string" && value.trim() ? value.trim() : "";
    }
    if (!value || typeof value !== "object" || Array.isArray(value)) return "";
    current = value as Record<string, unknown>;
  }
  return "";
}

function enumOption<T extends string>(value: string, values: readonly T[], fallback: T): T {
  return values.includes(value as T) ? (value as T) : fallback;
}

function sizeFromProviderOptions(options: Record<string, unknown>): string {
  const directSize = stringOption(options, "size");
  if (directSize) return directSize;

  const aspectRatio = stringOption(options, "aspect_ratio", "aspectRatio")
    || nestedStringOption(options, ["imageConfig", "aspectRatio"])
    || nestedStringOption(options, ["imageConfig", "aspect_ratio"])
    || nestedStringOption(options, ["image_config", "aspectRatio"])
    || nestedStringOption(options, ["image_config", "aspect_ratio"])
    || nestedStringOption(options, ["responseFormat", "image", "aspectRatio"])
    || nestedStringOption(options, ["responseFormat", "image", "aspect_ratio"])
    || nestedStringOption(options, ["generationConfig", "imageConfig", "aspectRatio"])
    || nestedStringOption(options, ["generationConfig", "imageConfig", "aspect_ratio"])
    || nestedStringOption(options, ["generationConfig", "responseFormat", "image", "aspectRatio"])
    || nestedStringOption(options, ["generationConfig", "responseFormat", "image", "aspect_ratio"]);
  if (!aspectRatio) return "";

  const rawTier = stringOption(options, "image_size", "imageSize", "resolution")
    || nestedStringOption(options, ["imageConfig", "imageSize"])
    || nestedStringOption(options, ["imageConfig", "image_size"])
    || nestedStringOption(options, ["image_config", "imageSize"])
    || nestedStringOption(options, ["image_config", "image_size"])
    || nestedStringOption(options, ["responseFormat", "image", "imageSize"])
    || nestedStringOption(options, ["responseFormat", "image", "image_size"])
    || nestedStringOption(options, ["generationConfig", "imageConfig", "imageSize"])
    || nestedStringOption(options, ["generationConfig", "imageConfig", "image_size"])
    || nestedStringOption(options, ["generationConfig", "responseFormat", "image", "imageSize"])
    || nestedStringOption(options, ["generationConfig", "responseFormat", "image", "image_size"]);
  const normalizedTier = rawTier.trim().toUpperCase();
  const tier = normalizedTier === "2K" || normalizedTier === "4K" ? normalizedTier : "1K";
  return calculateImageStudioSize(tier as ImageStudioSizeTier, aspectRatio) ?? "";
}

export function imageStudioParamsFromOptions(options: Record<string, unknown>): Partial<ImageStudioParams> {
  const params: Partial<ImageStudioParams> = {};
  const size = sizeFromProviderOptions(options);
  if (size) params.size = normalizeImageStudioSize(size);

  const quality = stringOption(options, "quality");
  if (quality) params.quality = enumOption(quality, IMAGE_STUDIO_QUALITY_VALUES, DEFAULT_IMAGE_STUDIO_PARAMS.quality);

  const outputFormat = stringOption(options, "output_format", "outputFormat");
  if (outputFormat) {
    params.outputFormat = enumOption(outputFormat, IMAGE_STUDIO_OUTPUT_FORMAT_VALUES, DEFAULT_IMAGE_STUDIO_PARAMS.outputFormat);
  }

  const outputCompression = numberOption(options, "output_compression", "outputCompression");
  if (outputCompression !== null) params.outputCompression = clampNumber(Math.round(outputCompression), 0, 100);

  const background = stringOption(options, "background");
  if (background) params.background = enumOption(background, IMAGE_STUDIO_BACKGROUND_VALUES, DEFAULT_IMAGE_STUDIO_PARAMS.background);

  const moderation = stringOption(options, "moderation");
  if (moderation) params.moderation = enumOption(moderation, IMAGE_STUDIO_MODERATION_VALUES, DEFAULT_IMAGE_STUDIO_PARAMS.moderation);

  const count = numberOption(options, "n");
  if (count !== null) params.n = Math.round(count);

  const partialImages = numberOption(options, "partial_images", "partialImages");
  if (partialImages !== null) params.partialImages = Math.round(partialImages);

  const inputFidelity = stringOption(options, "input_fidelity", "inputFidelity");
  if (inputFidelity) {
    params.inputFidelity = enumOption(inputFidelity, IMAGE_STUDIO_INPUT_FIDELITY_VALUES, DEFAULT_IMAGE_STUDIO_PARAMS.inputFidelity);
  }

  return params;
}

export function normalizeImageStudioParams(
  params: Partial<ImageStudioParams>,
  model: PublicModelDTO | null | undefined,
  taskType: ImageStudioTaskType,
): ImageStudioParams {
  const capabilities = resolveImageStudioParamCapabilities(model, taskType);
  const outputFormat = enumOption(params.outputFormat ?? DEFAULT_IMAGE_STUDIO_PARAMS.outputFormat, IMAGE_STUDIO_OUTPUT_FORMAT_VALUES, "png");
  const compression = params.outputCompression == null || outputFormat === "png"
    ? null
    : clampNumber(Math.round(Number(params.outputCompression)), 0, 100);

  return {
    size: normalizeImageStudioSize(params.size ?? DEFAULT_IMAGE_STUDIO_PARAMS.size),
    quality: enumOption(params.quality ?? DEFAULT_IMAGE_STUDIO_PARAMS.quality, IMAGE_STUDIO_QUALITY_VALUES, "auto"),
    outputFormat,
    outputCompression: compression,
    background: enumOption(params.background ?? DEFAULT_IMAGE_STUDIO_PARAMS.background, IMAGE_STUDIO_BACKGROUND_VALUES, "auto"),
    moderation: enumOption(params.moderation ?? DEFAULT_IMAGE_STUDIO_PARAMS.moderation, IMAGE_STUDIO_MODERATION_VALUES, "auto"),
    n: capabilities.supportsCount ? clampNumber(Math.round(Number(params.n ?? DEFAULT_IMAGE_STUDIO_PARAMS.n)), 1, MAX_OUTPUT_IMAGES) : 1,
    partialImages: capabilities.supportsPartialImages
      ? clampNumber(Math.round(Number(params.partialImages ?? DEFAULT_IMAGE_STUDIO_PARAMS.partialImages)), 0, MAX_PARTIAL_IMAGES)
      : 0,
    inputFidelity: enumOption(params.inputFidelity ?? DEFAULT_IMAGE_STUDIO_PARAMS.inputFidelity, IMAGE_STUDIO_INPUT_FIDELITY_VALUES, "auto"),
  };
}

export function defaultImageStudioParamsForModel(
  model: PublicModelDTO | null | undefined,
  taskType: ImageStudioTaskType,
): ImageStudioParams {
  const defaults = imageStudioParamsFromOptions(parseImageStudioModelDefaultOptions(model));
  return normalizeImageStudioParams({ ...DEFAULT_IMAGE_STUDIO_PARAMS, ...defaults }, model, taskType);
}

function openAIImageModelSupportsAuto(model: PublicModelDTO | null | undefined): boolean {
  const modelName = model?.platformModelName.trim().toLowerCase() ?? "";
  return modelName.startsWith("gpt-image-") || modelName === "chatgpt-image-latest";
}

function setStringOption(options: Record<string, unknown>, key: string, value: string) {
  if (value.trim()) options[key] = value.trim();
}

function setNumberOption(options: Record<string, unknown>, key: string, value: number | null | undefined) {
  if (typeof value === "number" && Number.isFinite(value)) options[key] = value;
}

function providerSizeConfig(size: string): { aspectRatio: string; tier: ImageStudioSizeTier } | null {
  const aspectRatio = imageStudioSizeAspectRatio(size);
  if (!aspectRatio) return null;
  return { aspectRatio, tier: imageStudioSizeTier(size) };
}

export function buildImageStudioOptions(
  params: ImageStudioParams,
  model: PublicModelDTO | null | undefined,
  taskType: ImageStudioTaskType,
): Record<string, unknown> {
  const normalized = normalizeImageStudioParams(params, model, taskType);
  const capabilities = resolveImageStudioParamCapabilities(model, taskType);
  const options = parseImageStudioModelDefaultOptions(model);

  if (capabilities.family === "google") {
    const sizeConfig = providerSizeConfig(normalized.size);
    if (sizeConfig) {
      options.aspect_ratio = sizeConfig.aspectRatio;
      options.image_size = sizeConfig.tier;
    }
    return options;
  }

  if (capabilities.family === "xai") {
    const sizeConfig = providerSizeConfig(normalized.size);
    if (sizeConfig) {
      options.aspect_ratio = sizeConfig.aspectRatio;
      options.resolution = sizeConfig.tier.toLowerCase();
    }
    setNumberOption(options, "n", normalized.n);
    return options;
  }

  const openAIAutoSize = capabilities.family === "openai" && openAIImageModelSupportsAuto(model);
  setStringOption(options, "size", normalized.size === "auto" && !openAIAutoSize ? "1024x1024" : normalized.size);
  if (normalized.quality !== "auto" || capabilities.family === "generic" || openAIAutoSize) {
    setStringOption(options, "quality", normalized.quality);
  }
  if (capabilities.supportsCount) {
    setNumberOption(options, "n", normalized.n);
  }
  if (capabilities.supportsOutputFormat) {
    setStringOption(options, "output_format", normalized.outputFormat);
  }
  if (capabilities.supportsCompression && normalized.outputFormat !== "png") {
    setNumberOption(options, "output_compression", normalized.outputCompression);
  } else {
    delete options.output_compression;
    delete options.outputCompression;
  }
  if (capabilities.supportsBackground && normalized.background !== "auto") {
    setStringOption(options, "background", normalized.background);
  }
  if (capabilities.supportsModeration && normalized.moderation !== "auto") {
    setStringOption(options, "moderation", normalized.moderation);
  }
  if (capabilities.supportsInputFidelity && normalized.inputFidelity !== "auto") {
    setStringOption(options, "input_fidelity", normalized.inputFidelity);
  }
  if (capabilities.supportsPartialImages && normalized.partialImages > 0) {
    setNumberOption(options, "partial_images", normalized.partialImages);
  } else {
    delete options.partial_images;
    delete options.partialImages;
  }

  return options;
}
