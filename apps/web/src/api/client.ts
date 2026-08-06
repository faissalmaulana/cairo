import { tokenStore } from './tokens.ts';
import type { TokenPair } from './tokens.ts';

export const API_BASE: string =
    import.meta.env.VITE_API_BASE;

export interface TokenResponse {
    access_token: string;
    refresh_token: string;
    token_type: string;
    expires_in: number;
    refresh_expires_in: number;
    api_key?: string;
}

interface Envelope<T> {
    success: boolean;
    data?: T;
    error?: { code: string; message: string };
}

export class ApiError extends Error {
    readonly status: number;
    readonly code: string;

    constructor(status: number, code: string, message: string) {
        super(message);
        this.name = 'ApiError';
        this.status = status;
        this.code = code;
    }
}

interface RequestOptions {
    method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';
    body?: unknown;
    token?: string;
}

async function parseResponse<T>(res: Response): Promise<T> {
    const payload = (await res.json()) as Envelope<T>;
    if (!res.ok || !payload.success) {
        throw new ApiError(
            res.status,
            payload.error?.code ?? 'SERVER_ERROR',
            payload.error?.message ?? `Request failed with status ${res.status}`,
        );
    }
    return payload.data as T;
}

async function perform<T>(
    path: string,
    options: RequestOptions,
    accessToken: string | null,
): Promise<T> {
    const headers: Record<string, string> = {};
    if (options.body !== undefined) {
        headers['Content-Type'] = 'application/json';
    }
    if (accessToken) {
        headers.Authorization = `Bearer ${accessToken}`;
    }

    const res = await fetch(`${API_BASE}${path}`, {
        method: options.method ?? 'GET',
        headers,
        body: options.body === undefined ? undefined : JSON.stringify(options.body),
    });

    return parseResponse<T>(res);
}

let refreshInFlight: Promise<TokenPair> | null = null;

function refreshTokens(): Promise<TokenPair> {
    if (!refreshInFlight) {
        refreshInFlight = (async (): Promise<TokenPair> => {
            const current = tokenStore.get();
            if (!current?.refreshToken) {
                throw new ApiError(401, 'INVALID_TOKEN', 'No active session');
            }

            const res = await perform<TokenResponse>(
                '/refresh',
                { method: 'POST' },
                current.refreshToken,
            );
            const pair: TokenPair = {
                accessToken: res.access_token,
                refreshToken: res.refresh_token,
            };
            tokenStore.set(pair);
            return pair;
        })().finally(() => {
            refreshInFlight = null;
        });
    }
    return refreshInFlight;
}

export async function apiFetch<T>(
    path: string,
    options: RequestOptions = {},
): Promise<T> {
    const accessToken =
        options.token !== undefined
            ? options.token
            : (tokenStore.get()?.accessToken ?? null);

    try {
        return await perform<T>(path, options, accessToken);
    } catch (err) {
        if (!(err instanceof ApiError) || err.status !== 401) {
            throw err;
        }
        try {
            const pair = await refreshTokens();
            return await perform<T>(path, options, pair.accessToken);
        } catch {
            tokenStore.set(null);
            throw err;
        }
    }
}
