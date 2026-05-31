export type ImageStudioTaskType = "image_generation" | "image_edit";
export type ImageStudioTaskStatus = "queued" | "running" | "success" | "error" | "deleted";
export type ImageStudioAssetRole = "input" | "mask" | "output";

export type ImageStudioAssetDTO = {
  fileID: string;
  role: ImageStudioAssetRole | string;
  sortOrder: number;
  fileName: string;
  mimeType: string;
  fileSize: number;
  sha256: string;
  fileSnapshotJSON: string;
  metadataJSON: string;
  createdAt: string;
  updatedAt: string;
};

export type ImageStudioTaskDTO = {
  taskID: string;
  userID: number;
  taskType: ImageStudioTaskType | string;
  prompt: string;
  status: ImageStudioTaskStatus | string;
  requestedModelName: string;
  platformModelName: string;
  providerProtocol: string;
  upstreamName: string;
  routedBindingCode: string;
  modelVendor: string;
  modelIcon: string;
  upstreamModelName: string;
  optionsJSON: string;
  routeSnapshotJSON: string;
  usageSnapshotJSON: string;
  billingSnapshotJSON: string;
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  reasoningTokens: number;
  latencyMS: number;
  billedCurrency: string;
  billedNanousd: number;
  isFavorite: boolean;
  favoritedAt: string | null;
  errorCode: string;
  errorMessage: string;
  startedAt: string | null;
  completedAt: string | null;
  createdAt: string;
  updatedAt: string;
  assets: ImageStudioAssetDTO[];
};

export type StreamImageStudioTaskRequest = {
  taskType: ImageStudioTaskType;
  prompt: string;
  model: string;
  options?: Record<string, unknown>;
  fileIDs?: string[];
  maskFileID?: string;
};

export type ImageStudioStreamEvent =
  | { type: "media_status"; status: string; message: string; task?: ImageStudioTaskDTO }
  | { type: "usage"; [key: string]: unknown }
  | { type: "media_image_delta"; index: number; b64_json: string; mime_type: string; revised_prompt?: string }
  | { type: "completed"; data: ImageStudioTaskDTO }
  | { type: "error"; status?: number; message?: string; errorCode?: string; [key: string]: unknown };

export type ImageStudioListParams = {
  page?: number;
  pageSize?: number;
  status?: string;
  favorite?: string;
  query?: string;
};
