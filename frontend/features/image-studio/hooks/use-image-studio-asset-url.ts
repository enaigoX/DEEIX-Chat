"use client";

import * as React from "react";

import { fetchFileContent } from "@/shared/api/file";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";

type ImageStudioAssetURLState =
  | { status: "idle"; url: "" }
  | { status: "loading"; url: "" }
  | { status: "ready"; url: string }
  | { status: "error"; url: ""; message: string };

export function useImageStudioAssetURL(fileID: string | null | undefined): ImageStudioAssetURLState {
  const objectURLRef = React.useRef<string | null>(null);
  const [state, setState] = React.useState<ImageStudioAssetURLState>({ status: "idle", url: "" });

  const revokeObjectURL = React.useCallback(() => {
    if (!objectURLRef.current) return;
    URL.revokeObjectURL(objectURLRef.current);
    objectURLRef.current = null;
  }, []);

  React.useEffect(() => {
    const normalizedFileID = fileID?.trim() ?? "";
    let cancelled = false;
    revokeObjectURL();

    if (!normalizedFileID) {
      setState({ status: "idle", url: "" });
      return undefined;
    }

    setState({ status: "loading", url: "" });

    void (async () => {
      try {
        const accessToken = await resolveAccessToken();
        if (!accessToken) {
          throw new Error("session expired");
        }
        const result = await fetchFileContent(accessToken, normalizedFileID);
        const objectURL = URL.createObjectURL(result.blob);

        if (cancelled) {
          URL.revokeObjectURL(objectURL);
          return;
        }

        objectURLRef.current = objectURL;
        setState({ status: "ready", url: objectURL });
      } catch (error) {
        if (cancelled) return;
        setState({
          status: "error",
          url: "",
          message: error instanceof Error ? error.message : "failed to load image",
        });
      }
    })();

    return () => {
      cancelled = true;
      revokeObjectURL();
    };
  }, [fileID, revokeObjectURL]);

  return state;
}

export async function downloadImageStudioAsset(fileID: string, fileName: string): Promise<void> {
  const normalizedFileID = fileID.trim();
  if (!normalizedFileID) return;

  const accessToken = await resolveAccessToken();
  if (!accessToken) {
    throw new Error("session expired");
  }

  const result = await fetchFileContent(accessToken, normalizedFileID);
  const objectURL = URL.createObjectURL(result.blob);
  try {
    const anchor = document.createElement("a");
    anchor.href = objectURL;
    anchor.download = fileName.trim() || normalizedFileID;
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
  } finally {
    URL.revokeObjectURL(objectURL);
  }
}
