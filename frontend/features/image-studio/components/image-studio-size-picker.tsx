"use client";

import * as React from "react";
import { Check, Maximize2 } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import {
  calculateImageStudioSize,
  imageStudioSizeAspectRatio,
  imageStudioSizeLabelValue,
  IMAGE_STUDIO_PRESET_RATIOS,
  IMAGE_STUDIO_SIZE_TIERS,
  normalizeImageStudioSize,
  parseImageStudioRatio,
  type ImageStudioPresetRatio,
  type ImageStudioSizeTier,
} from "@/features/image-studio/model/image-studio-params";

type ImageStudioSizePickerProps = {
  value: string;
  disabled?: boolean;
  onChange: (value: string) => void;
};

export function ImageStudioSizePicker({ value, disabled, onChange }: ImageStudioSizePickerProps) {
  const t = useTranslations("imageStudio.params");
  const [open, setOpen] = React.useState(false);
  const label = value.trim().toLowerCase() === "auto" ? t("size.auto") : imageStudioSizeLabelValue(value);

  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="h-8 min-w-0 justify-start gap-1.5 px-2"
        disabled={disabled}
        onClick={() => setOpen(true)}
        title={t("size.label")}
      >
        <Maximize2 className="size-3.5 shrink-0" />
        <span className="min-w-0 truncate text-xs">{label}</span>
      </Button>
      <ImageStudioSizeDialog
        open={open}
        value={value}
        onOpenChange={setOpen}
        onSelect={(nextValue) => {
          onChange(nextValue);
          setOpen(false);
        }}
      />
    </>
  );
}

function ImageStudioSizeDialog({
  open,
  value,
  onOpenChange,
  onSelect,
}: {
  open: boolean;
  value: string;
  onOpenChange: (open: boolean) => void;
  onSelect: (value: string) => void;
}) {
  const t = useTranslations("imageStudio.params");
  const current = normalizeImageStudioSize(value);
  const [tier, setTier] = React.useState<ImageStudioSizeTier>("1K");
  const [customRatio, setCustomRatio] = React.useState("1:1");
  const [customWidth, setCustomWidth] = React.useState("1024");
  const [customHeight, setCustomHeight] = React.useState("1024");

  React.useEffect(() => {
    if (!open) return;
    const parsed = parseResolution(current);
    if (parsed) {
      setCustomWidth(String(parsed.width));
      setCustomHeight(String(parsed.height));
      setCustomRatio(imageStudioSizeAspectRatio(current) || "1:1");
      return;
    }
    setCustomRatio(imageStudioSizeAspectRatio(current) || "1:1");
  }, [current, open]);

  const customRatioSize = calculateImageStudioSize(tier, customRatio);
  const customResolution = normalizeImageStudioSize(`${customWidth}x${customHeight}`);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[calc(100vw-2rem)] gap-4 p-4 sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("size.title")}</DialogTitle>
          <DialogDescription className="sr-only">{t("size.description")}</DialogDescription>
        </DialogHeader>

        <div className="grid gap-4">
          <div className="flex flex-wrap gap-2">
            <SizeChoice
              selected={current === "auto"}
              label={t("size.auto")}
              detail={t("size.autoDetail")}
              onClick={() => onSelect("auto")}
            />
          </div>

          <div className="grid gap-3">
            <div className="flex min-w-0 items-center justify-between gap-3">
              <Label className="mb-0">{t("size.tier")}</Label>
              <Tabs value={tier} onValueChange={(next) => setTier(next as ImageStudioSizeTier)}>
                <TabsList className="rounded-md">
                  {IMAGE_STUDIO_SIZE_TIERS.map((item) => (
                    <TabsTrigger key={item} value={item} className="h-7 rounded-md px-3 text-xs">
                      {item}
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            </div>

            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              {IMAGE_STUDIO_PRESET_RATIOS.map((ratio) => {
                const size = calculateImageStudioSize(tier, ratio) ?? "";
                return (
                  <SizeChoice
                    key={`${tier}-${ratio}`}
                    selected={current === size}
                    label={t(ratioLabelKey(ratio))}
                    detail={size}
                    previewRatio={ratio}
                    onClick={() => {
                      if (size) onSelect(size);
                    }}
                  />
                );
              })}
            </div>
          </div>

          <div className="grid gap-3 border-t border-border/60 pt-4">
            <div className="grid gap-2 sm:grid-cols-[1fr_auto] sm:items-end">
              <div className="grid gap-1.5">
                <Label htmlFor="image-studio-custom-ratio">{t("size.customRatio")}</Label>
                <Input
                  id="image-studio-custom-ratio"
                  value={customRatio}
                  className="h-8"
                  onChange={(event) => setCustomRatio(event.target.value)}
                />
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-8"
                disabled={!customRatioSize}
                onClick={() => {
                  if (customRatioSize) onSelect(customRatioSize);
                }}
              >
                <span>{customRatioSize || t("size.invalid")}</span>
              </Button>
            </div>

            <div className="grid gap-2 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
              <div className="grid gap-1.5">
                <Label htmlFor="image-studio-custom-width">{t("size.width")}</Label>
                <Input
                  id="image-studio-custom-width"
                  value={customWidth}
                  inputMode="numeric"
                  className="h-8"
                  onChange={(event) => setCustomWidth(event.target.value.replace(/[^\d]/g, ""))}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="image-studio-custom-height">{t("size.height")}</Label>
                <Input
                  id="image-studio-custom-height"
                  value={customHeight}
                  inputMode="numeric"
                  className="h-8"
                  onChange={(event) => setCustomHeight(event.target.value.replace(/[^\d]/g, ""))}
                />
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-8"
                disabled={!parseResolution(customResolution)}
                onClick={() => onSelect(customResolution)}
              >
                <span>{parseResolution(customResolution) ? customResolution : t("size.invalid")}</span>
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function SizeChoice({
  selected,
  label,
  detail,
  previewRatio,
  onClick,
}: {
  selected: boolean;
  label: string;
  detail: string;
  previewRatio?: ImageStudioPresetRatio;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={cn(
        "flex min-h-16 min-w-0 items-center gap-3 rounded-md border border-border/60 bg-background px-3 py-2 text-left transition-colors hover:bg-muted/40",
        selected && "border-primary/70 bg-primary/5",
      )}
      onClick={onClick}
    >
      {previewRatio ? <RatioSwatch ratio={previewRatio} /> : null}
      <span className="min-w-0 flex-1">
        <span className="block truncate text-xs font-medium">{label}</span>
        <span className="block truncate text-[11px] text-muted-foreground">{detail}</span>
      </span>
      {selected ? <Check className="size-3.5 shrink-0 text-primary" /> : null}
    </button>
  );
}

function RatioSwatch({ ratio }: { ratio: string }) {
  const parsed = parseImageStudioRatio(ratio) ?? { width: 1, height: 1 };
  const wide = parsed.width >= parsed.height;
  const width = wide ? 30 : Math.max(16, Math.round((parsed.width / parsed.height) * 30));
  const height = wide ? Math.max(16, Math.round((parsed.height / parsed.width) * 30)) : 30;

  return (
    <span className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted/55">
      <span
        className="rounded-[2px] border border-foreground/55 bg-background"
        style={{ width: `${width}px`, height: `${height}px` }}
      />
    </span>
  );
}

function parseResolution(value: string): { width: number; height: number } | null {
  const match = value.match(/^\s*(\d+)\s*x\s*(\d+)\s*$/i);
  if (!match) return null;
  const width = Number(match[1]);
  const height = Number(match[2]);
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return null;
  return { width, height };
}

function ratioLabelKey(ratio: ImageStudioPresetRatio): "ratio.square" | "ratio.landscape" | "ratio.portrait" | "ratio.wide" | "ratio.vertical" | "ratio.classic" | "ratio.poster" | "ratio.ultrawide" {
  switch (ratio) {
    case "1:1":
      return "ratio.square";
    case "3:2":
      return "ratio.landscape";
    case "2:3":
      return "ratio.portrait";
    case "16:9":
      return "ratio.wide";
    case "9:16":
      return "ratio.vertical";
    case "4:3":
      return "ratio.classic";
    case "3:4":
      return "ratio.poster";
    case "21:9":
      return "ratio.ultrawide";
    default:
      return "ratio.square";
  }
}
