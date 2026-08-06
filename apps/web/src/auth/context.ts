import { createContext } from 'react';
import type { SignInInput, SignUpInput, User } from '../api/auth.ts';

export interface AuthContextValue {
    user: User | null;
    isAuthenticated: boolean;
    isInitializing: boolean;
    isSigningIn: boolean;
    isSigningUp: boolean;
    signInError: Error | null;
    signUpError: Error | null;
    signIn: (input: SignInInput) => Promise<void>;
    signUp: (input: SignUpInput) => Promise<void>;
    signOut: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextValue | null>(null);