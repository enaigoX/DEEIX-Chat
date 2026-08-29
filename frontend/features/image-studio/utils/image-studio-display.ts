import type { ImageStudioAssetDTO, ImageStudioTaskDTO } from "@/shared/api/image-studio.types";
import type { PublicModelDTO } from "@/shared/api/model.types";
import type { ModelSelectOption } from "@/shared/components/model-select";
import { resolveModelOptionIconUrl, resolveModelOptionLabel } from "@/shared/lib/model-option-display";

export function parseModelKinds(model: PublicModelDTO): string[] {
  try {
    const parsed = JSON.parse(model.kindsJSON || "[]");
    return Array.isArray(parsed) ? parsed.map((item) => String(item)) : [];
  } catch {
    return [];
  }
}

export function imageModelOptions(models: PublicModelDTO[], taskType: "image_generation" | "image_edit"): ModelSelectOption[] {
  const expectedKind = taskType === "image_edit" ? "image_edit" : "image_gen";
  return models
    .filter((model) => parseModelKinds(model).includes(expectedKind))
    .sort((a, b) => (a.sortOrder - b.sortOrder) || a.platformModelName.localeCompare(b.platformModelName))
    .map((model) => ({
      label: resolveModelOptionLabel(model.platformModelName),
      value: model.platformModelName,
      iconUrl: resolveModelOptionIconUrl({
        platformModelName: model.platformModelName,
        vendor: model.vendor ?? "",
        icon: model.icon ?? "",
      }),
    }));
}

export function outputAssets(task: ImageStudioTaskDTO | null | undefined): ImageStudioAssetDTO[] {
  return (task?.assets ?? [])
    .filter((asset) => asset.role === "output")
    .sort((a, b) => a.sortOrder - b.sortOrder);
}

export function inputAssets(task: ImageStudioTaskDTO | null | undefined): ImageStudioAssetDTO[] {
  return (task?.assets ?? [])
    .filter((asset) => asset.role === "input" || asset.role === "mask")
    .sort((a, b) => a.sortOrder - b.sortOrder);
}

export function fileContentURL(fileID: string): string {
  return `/api/v1/files/${encodeURIComponent(fileID)}/content`;
}

export function formatCost(nanousd: number, currency: string): string {
  if (!nanousd || nanousd <= 0) return "";
  const usd = nanousd / 1_000_000_000;
  return `${currency || "USD"} ${usd.toFixed(6)}`;
}

export function safeJSONRecord(value: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(value || "{}");
    return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed as Record<string, unknown> : {};
  } catch {
    return {};
  }
}
