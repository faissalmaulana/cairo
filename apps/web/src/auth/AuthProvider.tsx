import { useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ApiError } from '../api/client.ts';
import { authApi } from '../api/auth.ts';
import { AuthContext } from './context.ts';
import type { AuthContextValue } from './context.ts';
import { tokenStore } from '../api/tokens.ts';
import type { TokenPair } from '../api/tokens.ts';

export function AuthProvider({ children }: { children: ReactNode }) {
    const queryClient = useQueryClient();
    const [tokens, setTokens] = useState<TokenPair | null>(() => tokenStore.get());
    const [initialized, setInitialized] = useState(false);

    useEffect(() => {
        const unsubscribe = tokenStore.subscribe((pair) => setTokens(pair));
        setInitialized(true);
        return unsubscribe;
    }, []);

    const meQuery = useQuery({
        queryKey: ['me'],
        queryFn: authApi.getMe,
        enabled: initialized && tokens !== null,
        retry: false,
    });

    useEffect(() => {
        if (!tokens) {
            return;
        }
        if (meQuery.error instanceof ApiError && meQuery.error.status === 401) {
            tokenStore.set(null);
            queryClient.removeQueries({ queryKey: ['me'] });
        }
    }, [tokens, meQuery.error, queryClient]);

    const signInMutation = useMutation({
        mutationFn: authApi.signIn,
        onSuccess: (response) => {
            tokenStore.set({
                accessToken: response.access_token,
                refreshToken: response.refresh_token,
            });
        },
    });

    const signUpMutation = useMutation({
        mutationFn: authApi.signUp,
        onSuccess: (response) => {
            tokenStore.set({
                accessToken: response.access_token,
                refreshToken: response.refresh_token,
            });
        },
    });

    const signOutMutation = useMutation({
        mutationFn: async () => {
            const current = tokenStore.get();
            if (!current) {
                return;
            }
            try {
                await authApi.logout(current.refreshToken);
            } catch {
                // best effort — always clear local session
            }
        },
        onSettled: () => {
            tokenStore.set(null);
            queryClient.clear();
        },
    });

    const value = useMemo<AuthContextValue>(
        () => ({
            user: meQuery.data ?? null,
            isAuthenticated: tokens !== null,
            isInitializing: !initialized,
            isSigningIn: signInMutation.isPending,
            isSigningUp: signUpMutation.isPending,
            signInError: signInMutation.error ?? null,
            signUpError: signUpMutation.error ?? null,
            signIn: (input) => signInMutation.mutateAsync(input).then(() => undefined),
            signUp: (input) => signUpMutation.mutateAsync(input).then(() => undefined),
            signOut: () => signOutMutation.mutateAsync(),
        }),
        [
            meQuery.data,
            tokens,
            initialized,
            signInMutation,
            signUpMutation,
            signOutMutation,
        ],
    );

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
