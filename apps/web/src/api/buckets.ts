import { apiFetch } from "./client.ts";

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
};
