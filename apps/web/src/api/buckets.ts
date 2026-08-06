import { API_BASE, ApiError, apiFetch } from "./client.ts";

export interface Bucket {
  id: string;
  name: string;
  visibility: "private" | "public";
  created_at: string;
  updated_at: string;
}

export interface ObjectMetadata {
  id: string;
  key: string;
  size: number;
  sha256sum: string;
  created_at: string;
}

export interface DownloadedObject {
  blob: Blob;
  contentType: string;
}

export const bucketsApi = {
  list(accountId: string, apiKey: string): Promise<Bucket[]> {
    return apiFetch<Bucket[]>(`/accounts/${accountId}/buckets`, {
      token: apiKey,
    });
  },

  get(accountId: string, apiKey: string, name: string): Promise<Bucket> {
    return apiFetch<Bucket>(`/accounts/${accountId}/buckets/${name}`, {
      token: apiKey,
    });
  },

  create(
    accountId: string,
    apiKey: string,
    name: string,
  ): Promise<{ message: string }> {
    return apiFetch<{ message: string }>(`/accounts/${accountId}/buckets`, {
      method: "POST",
      token: apiKey,
      body: { name },
    });
  },

  setVisibility(
    accountId: string,
    apiKey: string,
    name: string,
    setToPublic: boolean,
  ): Promise<{ message: string }> {
    return apiFetch<{ message: string }>(
      `/accounts/${accountId}/buckets/${name}/visibility`,
      {
        method: "PATCH",
        token: apiKey,
        body: { set_to_public: setToPublic },
      },
    );
  },

  remove(
    accountId: string,
    apiKey: string,
    name: string,
  ): Promise<{ message: string }> {
    return apiFetch<{ message: string }>(
      `/accounts/${accountId}/buckets/${name}`,
      { method: "DELETE", token: apiKey },
    );
  },

  listObjects(
    accountId: string,
    apiKey: string,
    bucketName: string,
  ): Promise<ObjectMetadata[]> {
    return apiFetch<ObjectMetadata[]>(
      `/accounts/${accountId}/buckets/${bucketName}/objects`,
      { token: apiKey },
    );
  },

  removeObject(
    accountId: string,
    apiKey: string,
    bucketName: string,
    key: string,
  ): Promise<{ message: string }> {
    return apiFetch<{ message: string }>(
      `/accounts/${accountId}/buckets/${bucketName}/objects/${encodeURI(key)}`,
      { method: "DELETE", token: apiKey },
    );
  },

  download(
    accountId: string,
    apiKey: string,
    bucketName: string,
    key: string,
  ): Promise<DownloadedObject> {
    const url = `${API_BASE}/accounts/${accountId}/buckets/${bucketName}/objects/${encodeURI(key)}`;
    return fetch(url, {
      headers: { Authorization: `Bearer ${apiKey}` },
    }).then(async (res) => {
      if (!res.ok) {
        let code = "SERVER_ERROR";
        let message = `Request failed with status ${res.status}`;
        try {
          const payload = (await res.json()) as {
            error?: { code?: string; message?: string };
          };
          if (payload.error?.code) {
            code = payload.error.code;
          }
          if (payload.error?.message) {
            message = payload.error.message;
          }
        } catch {
          // non-JSON error body — keep status-based message
        }
        throw new ApiError(res.status, code, message);
      }
      return {
        blob: await res.blob(),
        contentType: res.headers.get("content-type") ?? "application/octet-stream",
      };
    });
  },
};
