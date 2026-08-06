import toast from "react-hot-toast";
import { API_BASE } from "./client.ts";
import { queryClient } from "./queryClient.ts";

export interface ActiveUpload {
  id: string;
  bucketName: string;
  key: string;
  fileName: string;
  progress: number;
  status: "uploading" | "success" | "error";
  error?: string;
}

let uploads: Record<string, ActiveUpload> = {};
let snapshot: ActiveUpload[] = [];
const listeners = new Set<() => void>();

function rebuildSnapshot() {
  snapshot = Object.values(uploads);
}

function emit() {
  rebuildSnapshot();
  for (const listener of listeners) {
    listener();
  }
}

export const uploadsStore = {
  subscribe(listener: () => void): () => void {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  },
  getSnapshot(): ActiveUpload[] {
    return snapshot;
  },
};

let idCounter = 0;

function removeUpload(id: string) {
  if (!uploads[id]) {
    return;
  }
  delete uploads[id];
  emit();
}

function clearSoon(id: string) {
  setTimeout(() => removeUpload(id), 4000);
}

export interface StartUploadInput {
  accountId: string;
  apiKey: string;
  bucketName: string;
  key: string;
  file: File;
}

export function startUpload(input: StartUploadInput): void {
  const id = `upload-${++idCounter}`;
  uploads[id] = {
    id,
    bucketName: input.bucketName,
    key: input.key,
    fileName: input.file.name,
    progress: 0,
    status: "uploading",
  };
  emit();

  const url = `${API_BASE}/accounts/${input.accountId}/buckets/${input.bucketName}/objects/${encodeURI(input.key)}`;

  const xhr = new XMLHttpRequest();
  xhr.open("PUT", url);
  xhr.setRequestHeader("Authorization", `Bearer ${input.apiKey}`);

  xhr.upload.onprogress = (event) => {
    if (!event.lengthComputable) {
      return;
    }
    uploads[id] = {
      ...uploads[id],
      progress: Math.round((event.loaded / event.total) * 100),
    };
    emit();
  };

  xhr.onload = () => {
    if (xhr.status >= 200 && xhr.status < 300) {
      uploads[id] = { ...uploads[id], progress: 100, status: "success" };
      emit();
      queryClient.invalidateQueries({ queryKey: ["objects", input.bucketName] });
      toast.success("Object uploaded");
      clearSoon(id);
      return;
    }

    const message = errorMessageFromResponse(xhr.responseText, xhr.status);
    uploads[id] = { ...uploads[id], status: "error", error: message };
    emit();
    toast.error(message);
    clearSoon(id);
  };

  xhr.onerror = () => {
    const message = "Upload failed — network error";
    uploads[id] = { ...uploads[id], status: "error", error: message };
    emit();
    toast.error(message);
    clearSoon(id);
  };

  const form = new FormData();
  form.append("file", input.file);
  xhr.send(form);
}

function errorMessageFromResponse(responseText: string, status: number): string {
  try {
    const parsed = JSON.parse(responseText) as {
      error?: { code?: string; message?: string };
    };
    if (parsed.error?.message) {
      return parsed.error.message;
    }
  } catch {
    // fall through to the status-based message
  }
  return `Upload failed with status ${status}`;
}