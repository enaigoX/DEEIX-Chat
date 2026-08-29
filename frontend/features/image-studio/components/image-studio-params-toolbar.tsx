"use client";

import * as React from "react";
import { ImageDown, Layers, Shield, SlidersHorizontal, Sparkles } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Slider } from "@/components/ui/slider";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { ImageStudioTaskType } from "@/shared/api/image-studio.types";
import type { PublicModelDTO } from "@/shared/api/model.types";
import { ImageStudioSizePicker } from "@/features/image-studio/components/image-studio-size-picker";
import {
  IMAGE_STUDIO_BACKGROUND_VALUES,
  IMAGE_STUDIO_INPUT_FIDELITY_VALUES,
  IMAGE_STUDIO_MODERATION_VALUES,
  IMAGE_STUDIO_OUTPUT_FORMAT_VALUES,
  IMAGE_STUDIO_QUALITY_VALUES,
  defaultImageStudioParamsForModel,
  imageStudioSizeAspectRatio,
  imageStudioSizeTier,
  normalizeImageStudioParams,
  resolveImageStudioParamCapabilities,
  type ImageStudioBackground,
  type ImageStudioInputFidelity,
  type ImageStudioModeration,
  type ImageStudioOutputFormat,
  type ImageStudioParams,
  type ImageStudioQuality,
} from "@/features/image-studio/model/image-studio-params";

type ImageStudioParamsToolbarProps = {
  params: ImageStudioParams;
  model: PublicModelDTO | null;
  taskType: ImageStudioTaskType;
  disabled?: boolean;
  onChange: (params: ImageStudioParams) => void;
};

export function ImageStudioParamsToolbar({
  params,
  model,
  taskType,
  disabled,
  onChange,
}: ImageStudioParamsToolbarProps) {
  const t = useTranslations("imageStudio.params");
  const capabilities = resolveImageStudioParamCapabilities(model, taskType);
  const normalized = normalizeImageStudioParams(params, model, taskType);

  const updateParams = React.useCallback((patch: Partial<ImageStudioParams>) => {
    onChange(normalizeImageStudioParams({ ...normalized, ...patch }, model, taskType));
  }, [model, normalized, onChange, taskType]);

  const providerLabel = t(`provider.${capabilities.family}`);
  const sizeRatio = imageStudioSizeAspectRatio(normalized.size);
  const sizeTier = imageStudioSizeTier(normalized.size);
  const hasAdvanced = capabilities.supportsBackground
    || capabilities.supportsModeration
    || capabilities.supportsInputFidelity
    || capabilities.supportsPartialImages
    || capabilities.supportsCompression;

  return (
    <div className="mb-2 flex min-w-0 flex-wrap items-center gap-2">
      <ImageStudioSizePicker
        value={normalized.size}
        disabled={disabled}
        onChange={(size) => updateParams({ size })}
      />

      {capabilities.family === "google" || capabilities.family === "xai" ? (
        <span className="inline-flex h-8 max-w-full items-center gap-1.5 rounded-md border border-border/60 px-2 text-xs text-muted-foreground">
          <Sparkles className="size-3.5 shrink-0" />
          <span className="truncate">
            {providerLabel}{sizeRatio ? ` - ${sizeRatio} - ${sizeTier}` : ""}
          </span>
        </span>
      ) : null}

      {capabilities.supportsQuality ? (
        <ParamSelect<ImageStudioQuality>
          label={t("quality.label")}
          value={normalized.quality}
          values={IMAGE_STUDIO_QUALITY_VALUES}
          disabled={disabled}
          formatLabel={(value) => t(`quality.${value}`)}
          onChange={(quality) => updateParams({ quality })}
        />
      ) : null}

      {capabilities.supportsOutputFormat ? (
        <ParamSelect<ImageStudioOutputFormat>
          label={t("format.label")}
          value={normalized.outputFormat}
          values={IMAGE_STUDIO_OUTPUT_FORMAT_VALUES}
          disabled={disabled}
          formatLabel={(value) => t(`format.${value}`)}
          onChange={(outputFormat) => updateParams({ outputFormat })}
        />
      ) : null}

      {capabilities.supportsCount ? (
        <NumberField
          label={t("count")}
          value={normalized.n}
          min={1}
          max={10}
          disabled={disabled}
          className="w-[76px]"
          onChange={(n) => updateParams({ n })}
        />
      ) : null}

      {hasAdvanced ? (
        <Popover>
          <Tooltip>
            <TooltipTrigger asChild>
              <PopoverTrigger asChild>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  className="size-8"
                  disabled={disabled}
                  aria-label={t("advanced")}
                >
                  <SlidersHorizontal className="size-3.5" />
                </Button>
              </PopoverTrigger>
            </TooltipTrigger>
            <TooltipContent>{t("advanced")}</TooltipContent>
          </Tooltip>
          <PopoverContent align="end" className="w-[min(24rem,calc(100vw-2rem))] p-3">
            <div className="grid gap-3">
              <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <p className="truncate text-xs font-medium">{t("advanced")}</p>
                  <p className="truncate text-[11px] text-muted-foreground">{providerLabel}</p>
                </div>
                <Button type="button" variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={() => onChange(defaultImageStudioParamsForModel(model, taskType))}>
                  {t("reset")}
                </Button>
              </div>

              {capabilities.supportsBackground ? (
                <IconSelect<ImageStudioBackground>
                  icon={<ImageDown className="size-3.5" />}
                  label={t("background.label")}
                  value={normalized.background}
                  values={IMAGE_STUDIO_BACKGROUND_VALUES}
                  formatLabel={(value) => t(`background.${value}`)}
                  onChange={(background) => updateParams({ background })}
                />
              ) : null}

              {capabilities.supportsModeration ? (
                <IconSelect<ImageStudioModeration>
                  icon={<Shield className="size-3.5" />}
                  label={t("moderation.label")}
                  value={normalized.moderation}
                  values={IMAGE_STUDIO_MODERATION_VALUES}
                  formatLabel={(value) => t(`moderation.${value}`)}
                  onChange={(moderation) => updateParams({ moderation })}
                />
              ) : null}

              {capabilities.supportsInputFidelity ? (
                <IconSelect<ImageStudioInputFidelity>
                  icon={<Layers className="size-3.5" />}
                  label={t("inputFidelity.label")}
                  value={normalized.inputFidelity}
                  values={IMAGE_STUDIO_INPUT_FIDELITY_VALUES}
                  formatLabel={(value) => t(`inputFidelity.${value}`)}
                  onChange={(inputFidelity) => updateParams({ inputFidelity })}
                />
              ) : null}

              {capabilities.supportsCompression ? (
                <CompressionField
                  value={normalized.outputCompression}
                  disabled={normalized.outputFormat === "png"}
                  onChange={(outputCompression) => updateParams({ outputCompression })}
                />
              ) : null}

              {capabilities.supportsPartialImages ? (
                <div className="grid gap-1.5">
                  <Label htmlFor="image-studio-partials">{t("partialImages")}</Label>
                  <NumberField
                    id="image-studio-partials"
                    label={t("partialImages")}
                    value={normalized.partialImages}
                    min={0}
                    max={3}
                    onChange={(partialImages) => updateParams({ partialImages })}
                  />
                </div>
              ) : null}
            </div>
          </PopoverContent>
        </Popover>
      ) : null}
    </div>
  );
}

function ParamSelect<T extends string>({
  label,
  value,
  values,
  disabled,
  formatLabel,
  onChange,
}: {
  label: string;
  value: T;
  values: readonly T[];
  disabled?: boolean;
  formatLabel: (value: T) => string;
  onChange: (value: T) => void;
}) {
  return (
    <Select value={value} disabled={disabled} onValueChange={(next) => onChange(next as T)}>
      <SelectTrigger className="h-8 w-[112px] min-w-0 px-2" aria-label={label} title={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent align="start">
        {values.map((item) => (
          <SelectItem key={item} value={item}>
            {formatLabel(item)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function IconSelect<T extends string>({
  icon,
  label,
  value,
  values,
  formatLabel,
  onChange,
}: {
  icon: React.ReactNode;
  label: string;
  value: T;
  values: readonly T[];
  formatLabel: (value: T) => string;
  onChange: (value: T) => void;
}) {
  return (
    <div className="grid gap-1.5">
      <Label>{label}</Label>
      <div className="grid grid-cols-[auto_1fr] items-center gap-2">
        <span className="flex size-8 items-center justify-center rounded-md border border-border/60 text-muted-foreground">
          {icon}
        </span>
        <Select value={value} onValueChange={(next) => onChange(next as T)}>
          <SelectTrigger className="h-8">
            <SelectValue />
          </SelectTrigger>
          <SelectContent align="end">
            {values.map((item) => (
              <SelectItem key={item} value={item}>
                {formatLabel(item)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}

function NumberField({
  id,
  label,
  value,
  min,
  max,
  disabled,
  className,
  onChange,
}: {
  id?: string;
  label: string;
  value: number;
  min: number;
  max: number;
  disabled?: boolean;
  className?: string;
  onChange: (value: number) => void;
}) {
  return (
    <Input
      id={id}
      type="number"
      min={min}
      max={max}
      step={1}
      value={String(value)}
      disabled={disabled}
      aria-label={label}
      title={label}
      className={cn("h-8 px-2 text-xs", className)}
      onChange={(event) => {
        const numeric = Number(event.target.value);
        if (!Number.isFinite(numeric)) return;
        onChange(Math.min(max, Math.max(min, Math.round(numeric))));
      }}
    />
  );
}

function CompressionField({
  value,
  disabled,
  onChange,
}: {
  value: number | null;
  disabled: boolean;
  onChange: (value: number | null) => void;
}) {
  const t = useTranslations("imageStudio.params");
  const sliderValue = value ?? 80;

  return (
    <div className="grid gap-2">
      <div className="flex items-center justify-between gap-3">
        <Label className="mb-0" htmlFor="image-studio-compression">{t("compression")}</Label>
        <Input
          id="image-studio-compression"
          type="number"
          min={0}
          max={100}
          value={disabled || value == null ? "" : String(value)}
          disabled={disabled}
          placeholder={disabled ? t("disabled") : "80"}
          className="h-8 w-20 px-2 text-xs"
          onChange={(event) => {
            const numeric = Number(event.target.value);
            onChange(Number.isFinite(numeric) ? Math.min(100, Math.max(0, Math.round(numeric))) : null);
          }}
        />
      </div>
      <Slider
        min={0}
        max={100}
        step={1}
        value={[sliderValue]}
        disabled={disabled}
        onValueChange={(next) => onChange(next[0] ?? null)}
      />
    </div>
  );
}
