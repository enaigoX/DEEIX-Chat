"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import {
  deleteImageStudioTask,
  listImageStudioTasks,
  setImageStudioTaskFavorite,
  streamImageStudioTask,
} from "@/shared/api/image-studio";
import { uploadFile } from "@/shared/api/file";
import { listPublicModels } from "@/shared/api/model";
import type { ImageStudioStreamEvent, ImageStudioTaskDTO, ImageStudioTaskType } from "@/shared/api/image-studio.types";
import type { PublicModelDTO } from "@/shared/api/model.types";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";
import { imageModelOptions, outputAssets } from "@/features/image-studio/utils/image-studio-display";
import {
  buildImageStudioOptions,
  defaultImageStudioParamsForModel,
  normalizeImageStudioParams,
  type ImageStudioParams,
} from "@/features/image-studio/model/image-studio-params";

export type ImageStudioDraftFile = {
  fileID: string;
  fileName: string;
  previewURL: string;
};

export type ImageStudioFilter = "all" | "favorites" | "success" | "error";

export function useImageStudioPage() {
  const t = useTranslations("imageStudio.toast");
  const [tasks, setTasks] = React.useState<ImageStudioTaskDTO[]>([]);
  const [total, setTotal] = React.useState(0);
  const [models, setModels] = React.useState<PublicModelDTO[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [syncing, setSyncing] = React.useState(false);
  const [running, setRunning] = React.useState(false);
  const [uploading, setUploading] = React.useState(false);
  const [taskType, setTaskType] = React.useState<ImageStudioTaskType>("image_generation");
  const [model, setModel] = React.useState("");
  const [params, setParams] = React.useState<ImageStudioParams>(() => defaultImageStudioParamsForModel(null, "image_generation"));
  const [prompt, setPrompt] = React.useState("");
  const [filter, setFilter] = React.useState<ImageStudioFilter>("all");
  const [query, setQuery] = React.useState("");
  const [draftFiles, setDraftFiles] = React.useState<ImageStudioDraftFile[]>([]);
  const [selectedTask, setSelectedTask] = React.useState<ImageStudioTaskDTO | null>(null);
  const [streamStatus, setStreamStatus] = React.useState("");
  const [partialImages, setPartialImages] = React.useState<Record<number, string>>({});
  const fileInputRef = React.useRef<HTMLInputElement | null>(null);
  const draftFilesRef = React.useRef<ImageStudioDraftFile[]>([]);

  const modelOptions = React.useMemo(() => imageModelOptions(models, taskType), [models, taskType]);
  const selectedModel = model || modelOptions[0]?.value || "";
  const selectedModelInfo = React.useMemo(
    () => models.find((item) => item.platformModelName === selectedModel) ?? null,
    [models, selectedModel],
  );

  React.useEffect(() => {
    if (!modelOptions.length) {
      if (model) setModel("");
      return;
    }
    if (!model || !modelOptions.some((item) => item.value === model)) {
      setModel(modelOptions[0].value);
    }
  }, [model, modelOptions]);

  React.useEffect(() => {
    if (!selectedModelInfo) return;
    setParams((current) => normalizeImageStudioParams(current, selectedModelInfo, taskType));
  }, [selectedModelInfo, taskType]);

  const loadModels = React.useCallback(async () => {
    const accessToken = await resolveAccessToken();
    if (!accessToken) return;
    const nextModels = await listPublicModels(accessToken);
    setModels(nextModels);
  }, []);

  const loadTasks = React.useCallback(async (quiet = false) => {
    const accessToken = await resolveAccessToken();
    if (!accessToken) return;
    if (quiet) {
      setSyncing(true);
    } else {
      setLoading(true);
    }
    try {
      const status = filter === "success" || filter === "error" ? filter : undefined;
      const favorite = filter === "favorites" ? "true" : undefined;
      const page = await listImageStudioTasks(accessToken, {
        page: 1,
        pageSize: 60,
        status,
        favorite,
        query,
      });
      setTasks(page.results);
      setTotal(page.total);
    } finally {
      setLoading(false);
      setSyncing(false);
    }
  }, [filter, query]);

  React.useEffect(() => {
    void loadModels();
  }, [loadModels]);

  React.useEffect(() => {
    void loadTasks(false);
  }, [loadTasks]);

  React.useEffect(() => {
    draftFilesRef.current = draftFiles;
  }, [draftFiles]);

  React.useEffect(() => {
    return () => {
      draftFilesRef.current.forEach((file) => URL.revokeObjectURL(file.previewURL));
    };
  }, []);

  const onOpenFilePicker = React.useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const onFilesPicked = React.useCallback(async (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []).filter((file) => file.type.startsWith("image/"));
    event.target.value = "";
    if (!files.length) return;
    const accessToken = await resolveAccessToken();
    if (!accessToken) return;
    setUploading(true);
    try {
      const uploaded: ImageStudioDraftFile[] = [];
      for (const file of files.slice(0, 16 - draftFiles.length)) {
        const result = await uploadFile(accessToken, file, { purpose: "image_studio_input" });
        uploaded.push({
          fileID: result.file.fileID,
          fileName: result.file.fileName,
          previewURL: URL.createObjectURL(file),
        });
      }
      setDraftFiles((current) => [...current, ...uploaded]);
      if (uploaded.length > 0) setTaskType("image_edit");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("uploadFailed"));
    } finally {
      setUploading(false);
    }
  }, [draftFiles.length, t]);

  const onRemoveDraftFile = React.useCallback((fileID: string) => {
    setDraftFiles((current) => {
      const target = current.find((item) => item.fileID === fileID);
      if (target) URL.revokeObjectURL(target.previewURL);
      return current.filter((item) => item.fileID !== fileID);
    });
  }, []);

  const onSubmit = React.useCallback(async () => {
    const trimmedPrompt = prompt.trim();
    if (!trimmedPrompt || !selectedModel || running) return;
    const accessToken = await resolveAccessToken();
    if (!accessToken) return;
    setRunning(true);
    setStreamStatus("queued");
    setPartialImages({});
    try {
      const completed = await streamImageStudioTask(
        accessToken,
        {
          taskType,
          prompt: trimmedPrompt,
          model: selectedModel,
          options: buildImageStudioOptions(params, selectedModelInfo, taskType),
          fileIDs: taskType === "image_edit" ? draftFiles.map((file) => file.fileID) : [],
        },
        {
          onEvent: (event: ImageStudioStreamEvent) => {
            if (event.type === "media_status") {
              setStreamStatus(normalizeStreamStatus(event.status || ""));
              if (event.task) {
                setTasks((current) => upsertTask(current, event.task));
              }
            }
            if (event.type === "media_image_delta") {
              setPartialImages((current) => ({
                ...current,
                [event.index]: `data:${event.mime_type || "image/png"};base64,${event.b64_json}`,
              }));
            }
          },
        },
      );
      setTasks((current) => upsertTask(current, completed));
      setSelectedTask(completed);
      setPrompt("");
      setPartialImages({});
      setStreamStatus("");
      if (outputAssets(completed).length > 0) {
        toast.success(t("completed"));
      }
      setDraftFiles((current) => {
        current.forEach((file) => URL.revokeObjectURL(file.previewURL));
        return [];
      });
      void loadTasks(true);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("failed"));
    } finally {
      setRunning(false);
    }
  }, [draftFiles, loadTasks, params, prompt, running, selectedModel, selectedModelInfo, t, taskType]);

  const onToggleFavorite = React.useCallback(async (task: ImageStudioTaskDTO) => {
    const accessToken = await resolveAccessToken();
    if (!accessToken) return;
    const nextTask = await setImageStudioTaskFavorite(accessToken, task.taskID, !task.isFavorite);
    setTasks((current) => upsertTask(current, nextTask));
    setSelectedTask((current) => current?.taskID === nextTask.taskID ? nextTask : current);
  }, []);

  const onDeleteTask = React.useCallback(async (task: ImageStudioTaskDTO) => {
    const accessToken = await resolveAccessToken();
    if (!accessToken) return;
    await deleteImageStudioTask(accessToken, task.taskID);
    setTasks((current) => current.filter((item) => item.taskID !== task.taskID));
    setSelectedTask((current) => current?.taskID === task.taskID ? null : current);
    setTotal((current) => Math.max(0, current - 1));
  }, []);

  return {
    fileInputRef,
    tasks,
    total,
    models,
    modelOptions,
    selectedModel,
    selectedModelInfo,
    loading,
    syncing,
    running,
    uploading,
    taskType,
    model,
    params,
    prompt,
    filter,
    query,
    draftFiles,
    selectedTask,
    streamStatus,
    partialImages,
    setTaskType,
    setModel,
    setParams,
    setPrompt,
    setFilter,
    setQuery,
    setSelectedTask,
    onOpenFilePicker,
    onFilesPicked,
    onRemoveDraftFile,
    onSubmit,
    onToggleFavorite,
    onDeleteTask,
    refresh: () => loadTasks(true),
  };
}

function upsertTask(items: ImageStudioTaskDTO[], task: ImageStudioTaskDTO): ImageStudioTaskDTO[] {
  const existingIndex = items.findIndex((item) => item.taskID === task.taskID);
  if (existingIndex < 0) {
    return [task, ...items];
  }
  const next = [...items];
  next[existingIndex] = task;
  return next;
}

function normalizeStreamStatus(status: string): string {
  switch (status) {
    case "queued":
    case "running":
    case "success":
    case "error":
      return status;
    default:
      return status ? "running" : "";
  }
}
