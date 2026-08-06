const STORAGE_KEY = 'cairo.auth';

export interface TokenPair {
    accessToken: string;
    refreshToken: string;
}

const listeners = new Set<(pair: TokenPair | null) => void>();

function loadFromStorage(): TokenPair | null {
    try {
        const raw = localStorage.getItem(STORAGE_KEY);
        if (!raw) {
            return null;
        }
        const parsed = JSON.parse(raw) as Partial<TokenPair>;
        if (typeof parsed.accessToken !== 'string' || typeof parsed.refreshToken !== 'string') {
            return null;
        }
        return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    } catch {
        return null;
    }
}

function persist(pair: TokenPair | null): void {
    if (pair) {
        localStorage.setItem(
            STORAGE_KEY,
            JSON.stringify({ accessToken: pair.accessToken, refreshToken: pair.refreshToken }),
        );
    } else {
        localStorage.removeItem(STORAGE_KEY);
    }
}

let current: TokenPair | null = loadFromStorage();

export const tokenStore = {
    get(): TokenPair | null {
        return current;
    },

    set(pair: TokenPair | null): void {
        current = pair;
        persist(pair);
        for (const listener of listeners) {
            listener(pair);
        }
    },

    subscribe(listener: (pair: TokenPair | null) => void): () => void {
        listeners.add(listener);
        return () => {
            listeners.delete(listener);
        };
    },
};