import { apiFetch } from './client.ts';

export interface ApiKey {
    id: string;
    key: string;
    last_used?: string;
    created_at: string;
}

export interface CreateApiKeyResponse {
    id: string;
    key: string;
    created_at: string;
}

export const apiKeysApi = {
    list(): Promise<ApiKey[]> {
        return apiFetch<ApiKey[]>('/account/apikeys');
    },

    create(): Promise<CreateApiKeyResponse> {
        return apiFetch<CreateApiKeyResponse>('/account/apikeys', { method: 'POST' });
    },

    revoke(id: string): Promise<{ message: string }> {
        return apiFetch<{ message: string }>(`/account/apikeys/${id}`, {
            method: 'DELETE',
        });
    },
};