import { apiFetch } from './client.ts';
import type { TokenResponse } from './client.ts';

export interface SignUpInput {
    username: string;
    email: string;
    password: string;
}

export interface SignInInput {
    email: string;
    password: string;
}

export interface User {
    id: string;
    username: string;
    email: string;
}

export const authApi = {
    signUp(input: SignUpInput): Promise<TokenResponse> {
        return apiFetch<TokenResponse>('/signup', { method: 'POST', body: input });
    },

    signIn(input: SignInInput): Promise<TokenResponse> {
        return apiFetch<TokenResponse>('/signin', { method: 'POST', body: input });
    },

    getMe(): Promise<User> {
        return apiFetch<User>('/account');
    },

    logout(refreshToken: string): Promise<{ message: string }> {
        return apiFetch<{ message: string }>('/account/logout', {
            method: 'POST',
            body: { refresh_token: refreshToken },
        });
    },
};