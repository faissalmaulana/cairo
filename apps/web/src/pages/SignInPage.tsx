import { useNavigate, Link } from 'react-router';
import { useForm } from '@tanstack/react-form';
import { useAuth } from '../auth/useAuth.ts';

export default function SignInPage() {
    const { signIn, isSigningIn, signInError } = useAuth();
    const navigate = useNavigate();

    const form = useForm({
        defaultValues: { email: '', password: '' },
        onSubmit: async ({ value }) => {
            try {
                await signIn(value);
                navigate('/', { replace: true });
            } catch {
                // error is surfaced through signInError
            }
        },
    });

    return (
        <div className="flex min-h-screen items-center justify-center bg-neutral-50 px-4 py-12">
            <form
                className="w-full max-w-sm rounded-xl border border-neutral-200 bg-white p-8 shadow-sm"
                onSubmit={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    form.handleSubmit();
                }}
            >
                <h1 className="text-2xl font-semibold text-neutral-900">Sign in</h1>
                <p className="mt-1 text-sm text-neutral-500">
                    Welcome back, access your buckets and objects.
                </p>

                <div className="mt-6 space-y-4">
                    <form.Field name="email">
                        {(field) => (
                            <label className="block">
                                <span className="mb-1 block text-sm font-medium text-neutral-700">
                                    Email
                                </span>
                                <input
                                    type="email"
                                    autoComplete="email"
                                    required
                                    value={field.state.value}
                                    onChange={(e) => field.handleChange(e.target.value)}
                                    className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm outline-none focus:border-blue-500"
                                    placeholder="you@example.com"
                                />
                            </label>
                        )}
                    </form.Field>

                    <form.Field name="password">
                        {(field) => (
                            <label className="block">
                                <span className="mb-1 block text-sm font-medium text-neutral-700">
                                    Password
                                </span>
                                <input
                                    type="password"
                                    autoComplete="current-password"
                                    required
                                    value={field.state.value}
                                    onChange={(e) => field.handleChange(e.target.value)}
                                    className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm outline-none focus:border-blue-500"
                                    placeholder="••••••••"
                                />
                            </label>
                        )}
                    </form.Field>
                </div>

                {signInError && (
                    <p className="mt-4 text-sm text-red-600">{signInError.message}</p>
                )}

                <button
                    type="submit"
                    disabled={isSigningIn}
                    className="mt-6 w-full rounded-lg bg-blue-500 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-60"
                >
                    {isSigningIn ? 'Signing in...' : 'Sign in'}
                </button>

                <p className="mt-4 text-center text-sm text-neutral-500">
                    No account yet?{' '}
                    <Link className="font-medium text-blue-600 hover:underline" to="/signup">
                        Sign up
                    </Link>
                </p>
            </form>
        </div>
    );
}