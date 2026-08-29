"use client";

import * as React from "react";
import {
  Download,
  Heart,
  ImageIcon,
  Loader2,
  Pencil,
  RefreshCw,
  Send,
  Sparkles,
  Trash2,
  Upload,
  X,
} from "lucide-react";
import { useTranslations } from "next-intl";

import { Badge } from "@/components/ui/badge";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { ModelSelect } from "@/shared/components/model-select";
import type { ImageStudioAssetDTO, ImageStudioTaskDTO } from "@/shared/api/image-studio.types";
import { useImageStudioPage, type ImageStudioFilter } from "@/features/image-studio/hooks/use-image-studio-page";
import {
  downloadImageStudioAsset,
  useImageStudioAssetURL,
} from "@/features/image-studio/hooks/use-image-studio-asset-url";
import {
  formatCost,
  inputAssets,
  outputAssets,
} from "@/features/image-studio/utils/image-studio-display";
import { ImageStudioParamsToolbar } from "@/features/image-studio/components/image-studio-params-toolbar";

const FILTERS: ImageStudioFilter[] = ["all", "favorites", "success", "error"];

type ImageStudioStatusLabels = {
  queued: string;
  running: string;
  success: string;
  error: string;
  unknown: string;
};

type ImageStudioAssetLabels = {
  loading: string;
  failed: string;
};

export function AppImageStudio() {
  const t = useTranslations("imageStudio");
  const statusLabels = useImageStudioStatusLabels();
  const assetLabels = useImageStudioAssetLabels();
  const [deleteTarget, setDeleteTarget] = React.useState<ImageStudioTaskDTO | null>(null);
  const [batchDeleteOpen, setBatchDeleteOpen] = React.useState(false);
  const {
    fileInputRef,
    tasks,
    total,
    modelOptions,
    selectedModel,
    selectedModelInfo,
    loading,
    syncing,
    running,
    uploading,
    deleting,
    taskType,
    model,
    params,
    prompt,
    filter,
    query,
    draftFiles,
    selectedTask,
    streamStatus,
    partialImages: partialImageMap,
    selectedTaskIDs,
    allVisibleSelected,
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
    onToggleTaskSelection,
    onToggleSelectAll,
    clearTaskSelection,
    onDeleteSelectedTasks,
    refresh,
  } = useImageStudioPage();
  const partialImages = Object.entries(partialImageMap)
    .sort(([a], [b]) => Number(a) - Number(b))
    .map(([, value]) => value);
  const runningLabel = streamStatus
    ? statusLabels[streamStatus as keyof ImageStudioStatusLabels] ?? t("composer.running")
    : t("composer.running");

  return (
    <>
      <div className="flex h-full min-h-0 w-full min-w-0 flex-1 flex-col overflow-hidden bg-background">
        <header className="flex shrink-0 flex-col gap-3 border-b border-border/50 px-3 py-3 md:px-4">
          <div className="flex min-w-0 flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="flex min-w-0 items-center gap-2">
              <span className="flex size-8 shrink-0 items-center justify-center rounded-md border border-border/60 bg-muted/30">
                <Sparkles className="size-4" strokeWidth={1.7} />
              </span>
              <div className="min-w-0">
                <h1 className="truncate text-sm font-semibold leading-5">{t("title")}</h1>
                <p className="truncate text-xs text-muted-foreground">{t("cloudTasks", { count: total })}</p>
              </div>
            </div>

            <div className="grid min-w-0 grid-cols-1 gap-2 md:w-[520px] md:grid-cols-[1fr_auto]">
              <ModelSelect
                value={model}
                fallbackValue={modelOptions[0]?.value ?? ""}
                options={modelOptions}
                disabled={running || modelOptions.length === 0}
                align="end"
                valueAlign="start"
                triggerClassName="h-8"
                onChange={setModel}
              />
              <Button type="button" variant="outline" size="sm" className="h-8" onClick={refresh} disabled={syncing}>
                <RefreshCw className={cn("size-3.5", syncing && "animate-spin")} />
                <span>{t("refresh")}</span>
              </Button>
            </div>
          </div>

          <div className="flex min-w-0 flex-col gap-2 md:flex-row md:items-center md:justify-between">
            <Tabs value={taskType} onValueChange={(value) => setTaskType(value as "image_generation" | "image_edit")}>
              <TabsList className="rounded-md">
                <TabsTrigger
                  value="image_generation"
                  className="rounded-md"
                  aria-label={t("taskTypes.generation")}
                  title={t("taskTypes.generation")}
                >
                  <Sparkles className="size-3.5" />
                </TabsTrigger>
                <TabsTrigger
                  value="image_edit"
                  className="rounded-md"
                  aria-label={t("taskTypes.edit")}
                  title={t("taskTypes.edit")}
                >
                  <Pencil className="size-3.5" />
                </TabsTrigger>
              </TabsList>
            </Tabs>
            <div className="flex min-w-0 flex-1 gap-2 md:max-w-xl">
              <Input
                value={query}
                placeholder={t("searchPlaceholder")}
                className="h-8"
                onChange={(event) => setQuery(event.target.value)}
              />
              <div className="flex shrink-0 gap-1">
                {FILTERS.map((item) => (
                  <Button
                    key={item}
                    type="button"
                    size="sm"
                    variant={filter === item ? "default" : "outline"}
                    className="h-8 px-2"
                    onClick={() => setFilter(item)}
                  >
                    {t(`filters.${item}`)}
                  </Button>
                ))}
              </div>
            </div>
          </div>
        </header>

        <main className="min-h-0 flex-1 overflow-y-auto px-3 py-3 md:px-4">
          {!loading && tasks.length > 0 ? (
            <div className="mb-3 flex min-h-8 items-center justify-between gap-3 border-b border-border/50 pb-3">
              <label className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
                <Checkbox
                  checked={allVisibleSelected}
                  onCheckedChange={onToggleSelectAll}
                  aria-label={t("selection.selectAll")}
                />
                <span>
                  {selectedTaskIDs.size > 0
                    ? t("selection.selected", { count: selectedTaskIDs.size })
                    : t("selection.selectAll")}
                </span>
              </label>
              {selectedTaskIDs.size > 0 ? (
                <div className="flex shrink-0 items-center gap-1">
                  <Button type="button" variant="ghost" size="sm" className="h-8" onClick={clearTaskSelection} disabled={deleting}>
                    {t("selection.clear")}
                  </Button>
                  <Button type="button" variant="destructive" size="sm" className="h-8" onClick={() => setBatchDeleteOpen(true)} disabled={deleting}>
                    {deleting ? <Loader2 className="size-3.5 animate-spin" /> : <Trash2 className="size-3.5" />}
                    <span>{t("selection.deleteSelected")}</span>
                  </Button>
                </div>
              ) : null}
            </div>
          ) : null}
          {loading ? (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
              {Array.from({ length: 10 }).map((_, index) => (
                <div key={index} className="aspect-[4/5] animate-pulse rounded-md bg-muted/40" />
              ))}
            </div>
          ) : tasks.length === 0 && partialImages.length === 0 ? (
            <div className="flex h-full min-h-[360px] items-center justify-center rounded-md border border-dashed border-border/60">
              <div className="flex max-w-xs flex-col items-center gap-2 text-center">
                <ImageIcon className="size-8 text-muted-foreground" strokeWidth={1.5} />
                <p className="text-sm font-medium">{t("empty.title")}</p>
                <p className="text-xs leading-5 text-muted-foreground">{t("empty.description")}</p>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
              {partialImages.map((src, index) => (
                <div key={`partial-${index}`} className="group relative aspect-[4/5] overflow-hidden rounded-md border border-border/60 bg-muted/30">
                  <img src={src} alt="" className="h-full w-full object-cover" />
                  <div className="absolute left-2 top-2">
                    <Badge variant="secondary">{t("streaming")}</Badge>
                  </div>
                </div>
              ))}
              {tasks.map((task) => (
                <TaskCard
                  key={task.taskID}
                  task={task}
                  statusLabels={statusLabels}
                  labels={{
                    asset: assetLabels,
                    favorite: t("card.favorite"),
                    delete: t("card.delete"),
                    select: t("card.select"),
                    deselect: t("card.deselect"),
                    imageCount: (count) => t("card.imageCount", { count }),
                  }}
                  onOpen={() => setSelectedTask(task)}
                  onToggleFavorite={() => void onToggleFavorite(task)}
                  onDelete={() => setDeleteTarget(task)}
                  selected={selectedTaskIDs.has(task.taskID)}
                  onToggleSelection={() => onToggleTaskSelection(task.taskID)}
                />
              ))}
            </div>
          )}
        </main>

        <footer className="shrink-0 border-t border-border/50 bg-background/95 px-3 py-3 md:px-4">
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            multiple
            className="hidden"
            onChange={onFilesPicked}
          />
          {draftFiles.length > 0 ? (
            <div className="mb-2 flex min-w-0 gap-2 overflow-x-auto pb-1">
              {draftFiles.map((file) => (
                <div key={file.fileID} className="relative size-16 shrink-0 overflow-hidden rounded-md border border-border/60">
                  <img src={file.previewURL} alt="" className="h-full w-full object-cover" />
                  <button
                    type="button"
                    aria-label={t("composer.removeFile")}
                    title={t("composer.removeFile")}
                    className="absolute right-1 top-1 flex size-5 items-center justify-center rounded-full bg-background/90 text-foreground shadow-sm"
                    onClick={() => onRemoveDraftFile(file.fileID)}
                  >
                    <X className="size-3" />
                  </button>
                </div>
              ))}
            </div>
          ) : null}
          <ImageStudioParamsToolbar
            params={params}
            model={selectedModelInfo}
            taskType={taskType}
            disabled={running || modelOptions.length === 0}
            onChange={setParams}
          />
          <div className="grid gap-2 md:grid-cols-[auto_1fr_auto] md:items-end">
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="size-9"
              onClick={onOpenFilePicker}
              disabled={uploading || running}
              aria-label={t("composer.upload")}
              title={t("composer.upload")}
            >
              {uploading ? <Loader2 className="size-4 animate-spin" /> : <Upload className="size-4" />}
            </Button>
            <Textarea
              value={prompt}
              placeholder={taskType === "image_edit" ? t("composer.editPlaceholder") : t("composer.generationPlaceholder")}
              className="max-h-36 min-h-20 resize-none text-sm"
              onChange={(event) => setPrompt(event.target.value)}
              onKeyDown={(event) => {
                if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
                  event.preventDefault();
                  void onSubmit();
                }
              }}
            />
            <Button
              type="button"
              className="h-9"
              disabled={!prompt.trim() || !selectedModel || running || (taskType === "image_edit" && draftFiles.length === 0)}
              onClick={() => void onSubmit()}
            >
              {running ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
              <span>{running ? runningLabel : t("composer.run")}</span>
            </Button>
          </div>
        </footer>
      </div>

      <TaskDetailDialog
        task={selectedTask}
        statusLabels={statusLabels}
        open={Boolean(selectedTask)}
        onOpenChange={(open) => {
          if (!open) setSelectedTask(null);
        }}
        onToggleFavorite={() => {
          if (selectedTask) void onToggleFavorite(selectedTask);
        }}
        onDelete={() => {
          if (selectedTask) setDeleteTarget(selectedTask);
        }}
      />

      <AlertDialog open={Boolean(deleteTarget)} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("deleteDialog.title")}</AlertDialogTitle>
            <AlertDialogDescription>{t("deleteDialog.description")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>{t("deleteDialog.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleting}
              onClick={() => {
                if (deleteTarget) void onDeleteTask(deleteTarget);
                setDeleteTarget(null);
              }}
            >
              {t("deleteDialog.confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={batchDeleteOpen} onOpenChange={setBatchDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("batchDeleteDialog.title")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("batchDeleteDialog.description", { count: selectedTaskIDs.size })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>{t("deleteDialog.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleting}
              onClick={() => {
                void onDeleteSelectedTasks();
                setBatchDeleteOpen(false);
              }}
            >
              {t("batchDeleteDialog.confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

function useImageStudioAssetLabels(): ImageStudioAssetLabels {
  const t = useTranslations("imageStudio.asset");
  return React.useMemo(
    () => ({
      loading: t("loading"),
      failed: t("failed"),
    }),
    [t],
  );
}

function useImageStudioStatusLabels(): ImageStudioStatusLabels {
  const t = useTranslations("imageStudio.status");
  return React.useMemo(
    () => ({
      queued: t("queued"),
      running: t("running"),
      success: t("success"),
      error: t("error"),
      unknown: t("unknown"),
    }),
    [t],
  );
}

function TaskCard({
  task,
  statusLabels,
  labels,
  onOpen,
  onToggleFavorite,
  onDelete,
  selected,
  onToggleSelection,
}: {
  task: ImageStudioTaskDTO;
  statusLabels: ImageStudioStatusLabels;
  labels: {
    asset: ImageStudioAssetLabels;
    favorite: string;
    delete: string;
    select: string;
    deselect: string;
    imageCount: (count: number) => string;
  };
  onOpen: () => void;
  onToggleFavorite: () => void;
  onDelete: () => void;
  selected: boolean;
  onToggleSelection: () => void;
}) {
  const outputs = outputAssets(task);
  const preview = outputs[0];
  return (
    <article className={cn("group relative overflow-hidden rounded-md border border-border/55 bg-card", selected && "border-primary ring-1 ring-primary")}>
      <button type="button" className="block aspect-[4/5] w-full bg-muted/30 text-left" onClick={onOpen}>
        {preview ? (
          <ImageStudioAssetImage
            asset={preview}
            labels={labels.asset}
            thumbnail
            lazy
            className="h-full w-full object-cover transition-transform duration-200 group-hover:scale-[1.02]"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center">
            {task.status === "running" || task.status === "queued" ? (
              <Loader2 className="size-6 animate-spin text-muted-foreground" />
            ) : (
              <ImageIcon className="size-7 text-muted-foreground" strokeWidth={1.5} />
            )}
          </div>
        )}
      </button>
      <div className="absolute left-2 top-2 z-10 flex size-7 items-center justify-center rounded-md bg-background/92 shadow-sm">
        <Checkbox
          checked={selected}
          onCheckedChange={onToggleSelection}
          onClick={(event) => event.stopPropagation()}
          aria-label={selected ? labels.deselect : labels.select}
        />
      </div>
      <div className="absolute left-10 top-2 flex gap-1">
        <Badge variant={task.status === "success" ? "secondary" : task.status === "error" ? "destructive" : "outline"}>
          {statusLabels[task.status as keyof ImageStudioStatusLabels] ?? (task.status || statusLabels.unknown)}
        </Badge>
      </div>
      <div className="absolute right-2 top-2 flex gap-1 opacity-0 transition-opacity group-hover:opacity-100">
        <IconButton label={labels.favorite} onClick={onToggleFavorite}>
          <Heart className={cn("size-3.5", task.isFavorite && "fill-current")} />
        </IconButton>
        <IconButton label={labels.delete} onClick={onDelete}>
          <Trash2 className="size-3.5" />
        </IconButton>
      </div>
      <div className="space-y-1 p-2">
        <p className="line-clamp-2 min-h-8 text-xs leading-4">{task.prompt}</p>
        <div className="flex min-w-0 items-center justify-between gap-2 text-[11px] text-muted-foreground">
          <span className="truncate">{task.platformModelName || task.requestedModelName}</span>
          <span className="shrink-0">{labels.imageCount(outputs.length || 0)}</span>
        </div>
      </div>
    </article>
  );
}

function TaskDetailDialog({
  task,
  statusLabels,
  open,
  onOpenChange,
  onToggleFavorite,
  onDelete,
}: {
  task: ImageStudioTaskDTO | null;
  statusLabels: ImageStudioStatusLabels;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onToggleFavorite: () => void;
  onDelete: () => void;
}) {
  const t = useTranslations("imageStudio.detail");
  const assetLabels = useImageStudioAssetLabels();
  const outputs = outputAssets(task);
  const inputs = inputAssets(task);
  const downloadFirstOutput = React.useCallback(async () => {
    if (!outputs[0]) return;
    await downloadImageStudioAsset(outputs[0].fileID, outputs[0].fileName);
  }, [outputs]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[calc(100vw-2rem)] gap-3 p-4 sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
          <DialogDescription className="sr-only">{t("description")}</DialogDescription>
        </DialogHeader>
        {task ? (
          <div className="grid min-h-0 gap-4 md:grid-cols-[minmax(0,1fr)_320px]">
            <div className="grid max-h-[70svh] min-h-0 gap-3 overflow-y-auto">
              {outputs.length > 0 ? outputs.map((asset) => (
                <ImageStudioAssetImage
                  key={asset.fileID}
                  asset={asset}
                  labels={assetLabels}
                  className="w-full rounded-md border border-border/60 object-contain"
                />
              )) : (
                <div className="flex aspect-square items-center justify-center rounded-md border border-border/60 bg-muted/30">
                  <ImageIcon className="size-8 text-muted-foreground" />
                </div>
              )}
            </div>
            <aside className="min-w-0 space-y-4">
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <Badge variant={task.status === "error" ? "destructive" : "secondary"}>
                    {statusLabels[task.status as keyof ImageStudioStatusLabels] ?? (task.status || statusLabels.unknown)}
                  </Badge>
                  {formatCost(task.billedNanousd, task.billedCurrency) ? (
                    <Badge variant="outline">{formatCost(task.billedNanousd, task.billedCurrency)}</Badge>
                  ) : null}
                </div>
                <p className="text-sm leading-6">{task.prompt}</p>
                {task.errorMessage ? <p className="text-xs leading-5 text-destructive">{task.errorMessage}</p> : null}
              </div>

              <div className="grid gap-2 text-xs text-muted-foreground">
                <MetaRow label={t("model")} value={task.platformModelName || task.requestedModelName} />
                <MetaRow label={t("provider")} value={task.upstreamName || task.providerProtocol} />
                <MetaRow label={t("latency")} value={task.latencyMS ? `${task.latencyMS} ms` : ""} />
                <MetaRow label={t("created")} value={new Date(task.createdAt).toLocaleString()} />
              </div>

              {inputs.length > 0 ? (
                <div className="space-y-2">
                  <p className="text-xs font-medium">{t("inputs")}</p>
                  <div className="grid grid-cols-4 gap-2">
                    {inputs.map((asset) => (
                      <ImageStudioAssetImage
                        key={`${asset.role}-${asset.fileID}`}
                        asset={asset}
                        labels={assetLabels}
                        className="aspect-square rounded-md border border-border/60 object-cover"
                      />
                    ))}
                  </div>
                </div>
              ) : null}

              <div className="flex gap-2">
                <Button type="button" variant="outline" size="sm" onClick={onToggleFavorite}>
                  <Heart className={cn("size-3.5", task.isFavorite && "fill-current")} />
                  <span>{task.isFavorite ? t("favorited") : t("favorite")}</span>
                </Button>
                {outputs[0] ? (
                  <Button type="button" variant="outline" size="sm" onClick={() => void downloadFirstOutput()}>
                    <Download className="size-3.5" />
                    <span>{t("download")}</span>
                  </Button>
                ) : null}
                <Button type="button" variant="destructive" size="sm" onClick={onDelete} aria-label={t("delete")} title={t("delete")}>
                  <Trash2 className="size-3.5" />
                </Button>
              </div>
            </aside>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function ImageStudioAssetImage({
  asset,
  labels,
  className,
  thumbnail = false,
  lazy = false,
}: {
  asset: ImageStudioAssetDTO;
  labels: ImageStudioAssetLabels;
  className?: string;
  thumbnail?: boolean;
  lazy?: boolean;
}) {
  const containerRef = React.useRef<HTMLDivElement | null>(null);
  const [visible, setVisible] = React.useState(!lazy);
  const thumbnailFileID = thumbnail ? asset.thumbnailFileID?.trim() : "";
  const thumbnailURL = useImageStudioAssetURL(visible && thumbnailFileID ? thumbnailFileID : null);
  const shouldLoadOriginal = visible && (!thumbnailFileID || thumbnailURL.status === "error");
  const originalURL = useImageStudioAssetURL(shouldLoadOriginal ? asset.fileID : null);
  const assetURL = thumbnailFileID && thumbnailURL.status !== "error" ? thumbnailURL : originalURL;

  React.useEffect(() => {
    if (!lazy || visible) return;
    const element = containerRef.current;
    if (!element || typeof IntersectionObserver === "undefined") {
      setVisible(true);
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries.some((entry) => entry.isIntersecting)) return;
        setVisible(true);
        observer.disconnect();
      },
      { rootMargin: "480px" },
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, [lazy, visible]);

  if (assetURL.status === "ready") {
    return <img src={assetURL.url} alt={asset.fileName} className={className} loading={lazy ? "lazy" : "eager"} />;
  }

  return (
    <div ref={containerRef} className={cn("flex items-center justify-center bg-muted/30 text-muted-foreground", className)}>
      {assetURL.status === "loading" ? (
        <Loader2 className="size-5 animate-spin" />
      ) : (
        <span className="px-2 text-center text-xs">{labels.failed}</span>
      )}
      <span className="sr-only">{assetURL.status === "loading" ? labels.loading : labels.failed}</span>
    </div>
  );
}

function MetaRow({ label, value }: { label: string; value: string }) {
  if (!value) return null;
  return (
    <div className="flex min-w-0 justify-between gap-3">
      <span className="shrink-0">{label}</span>
      <span className="min-w-0 truncate text-foreground">{value}</span>
    </div>
  );
}

function IconButton({
  label,
  children,
  onClick,
}: {
  label: string;
  children: React.ReactNode;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      className="flex size-7 items-center justify-center rounded-md bg-background/92 text-foreground shadow-sm transition-colors hover:bg-background"
      onClick={(event) => {
        event.stopPropagation();
        onClick();
      }}
    >
      {children}
    </button>
  );
}
