import { authedFetch, authedRequest } from "@/shared/api/authed-client";
import { ApiError, pathParam } from "@/shared/api/http-client";
import type { PagePayload } from "@/shared/api/common.types";
import type {
  ImageStudioListParams,
  BatchDeleteImageStudioTasksResult,
  ImageStudioStreamEvent,
  ImageStudioTaskDTO,
  StreamImageStudioTaskRequest,
} from "@/shared/api/image-studio.types";

function extractJSONDocuments(source: string): { documents: string[]; remainder: string } {
  const documents: string[] = [];
  let startIndex = -1;
  let depth = 0;
  let inString = false;
  let escaped = false;
  let lastConsumedIndex = 0;

  for (let index = 0; index < source.length; index += 1) {
    const char = source[index];
    if (startIndex < 0) {
      if (char === "{") {
        startIndex = index;
        depth = 1;
        lastConsumedIndex = index;
      } else if (!/\s/.test(char)) {
        break;
      } else {
        lastConsumedIndex = index + 1;
      }
      continue;
    }
    if (inString) {
      if (escaped) {
        escaped = false;
      } else if (char === "\\") {
        escaped = true;
      } else if (char === "\"") {
        inString = false;
      }
      continue;
    }
    if (char === "\"") {
      inString = true;
      continue;
    }
    if (char === "{") {
      depth += 1;
      continue;
    }
    if (char !== "}") {
      continue;
    }
    depth -= 1;
    if (depth !== 0) {
      continue;
    }
    documents.push(source.slice(startIndex, index + 1));
    startIndex = -1;
    lastConsumedIndex = index + 1;
  }

  if (startIndex >= 0) {
    return { documents, remainder: source.slice(startIndex) };
  }
  return { documents, remainder: source.slice(lastConsumedIndex) };
}

function normalizeStreamEvent(rawEvent: unknown): ImageStudioStreamEvent {
  if (!rawEvent || typeof rawEvent !== "object") {
    throw new ApiError("stream event is invalid", 500);
  }
  return rawEvent as ImageStudioStreamEvent;
}

export type ImageStudioStreamOptions = {
  signal?: AbortSignal;
  onEvent?: (event: ImageStudioStreamEvent) => void;
};

export async function listImageStudioTasks(
  accessToken: string,
  params: ImageStudioListParams = {},
): Promise<PagePayload<ImageStudioTaskDTO>> {
  const searchParams = new URLSearchParams();
  searchParams.set("page", String(params.page && params.page > 0 ? params.page : 1));
  searchParams.set("page_size", String(params.pageSize && params.pageSize > 0 ? params.pageSize : 30));
  if (params.status?.trim()) searchParams.set("status", params.status.trim());
  if (params.favorite?.trim()) searchParams.set("favorite", params.favorite.trim());
  if (params.query?.trim()) searchParams.set("q", params.query.trim());
  const data = await authedRequest<PagePayload<ImageStudioTaskDTO>>(
    `/api/v1/image-studio/tasks?${searchParams.toString()}`,
    { accessToken },
    true,
  );
  return {
    total: data.total ?? 0,
    results: data.results ?? [],
  };
}

export async function getImageStudioTask(accessToken: string, taskID: string): Promise<ImageStudioTaskDTO> {
  return authedRequest<ImageStudioTaskDTO>(
    `/api/v1/image-studio/tasks/${pathParam(taskID)}`,
    { accessToken },
    true,
  );
}

export async function setImageStudioTaskFavorite(
  accessToken: string,
  taskID: string,
  favorite: boolean,
): Promise<ImageStudioTaskDTO> {
  return authedRequest<ImageStudioTaskDTO>(
    `/api/v1/image-studio/tasks/${pathParam(taskID)}/favorite`,
    {
      method: "PATCH",
      accessToken,
      body: { favorite },
    },
    true,
  );
}

export async function deleteImageStudioTask(
  accessToken: string,
  taskID: string,
): Promise<{ deleted: boolean; cleanupFailedFileIDs: string[] }> {
  return authedRequest<{ deleted: boolean; cleanupFailedFileIDs: string[] }>(
    `/api/v1/image-studio/tasks/${pathParam(taskID)}`,
    {
      method: "DELETE",
      accessToken,
    },
    true,
  );
}

export async function batchDeleteImageStudioTasks(
  accessToken: string,
  taskIDs: string[],
): Promise<BatchDeleteImageStudioTasksResult> {
  return authedRequest<BatchDeleteImageStudioTasksResult>(
    "/api/v1/image-studio/tasks/batch-delete",
    {
      method: "POST",
      accessToken,
      body: { taskIDs },
    },
    true,
  );
}

export async function streamImageStudioTask(
  accessToken: string,
  payload: StreamImageStudioTaskRequest,
  options: ImageStudioStreamOptions = {},
): Promise<ImageStudioTaskDTO> {
  const response = await authedFetch(
    "/api/v1/image-studio/tasks/stream",
    {
      method: "POST",
      accessToken,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
      signal: options.signal,
    },
    true,
  );
  if (!response.body) {
    throw new ApiError("stream body is empty", response.status);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let completed: ImageStudioTaskDTO | null = null;

  while (true) {
    const { done, value } = await reader.read();
    buffer += decoder.decode(value ?? new Uint8Array(), { stream: !done });
    const { documents, remainder } = extractJSONDocuments(buffer);
    buffer = remainder;

    for (const document of documents) {
      const event = normalizeStreamEvent(JSON.parse(document));
      options.onEvent?.(event);
      if (event.type === "completed") {
        completed = event.data;
      }
      if (event.type === "error") {
        throw new ApiError(event.message || "image task failed", event.status || response.status, undefined, event.errorCode);
      }
    }

    if (done) break;
  }

  const tail = buffer.trim();
  if (tail) {
    const event = normalizeStreamEvent(JSON.parse(tail));
    options.onEvent?.(event);
    if (event.type === "completed") completed = event.data;
    if (event.type === "error") {
      throw new ApiError(event.message || "image task failed", event.status || response.status, undefined, event.errorCode);
    }
  }

  if (!completed) {
    throw new ApiError("stream completed without final payload", response.status);
  }
  return completed;
}
