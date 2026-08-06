import { useNavigate, Link } from 'react-router';
import { useForm } from '@tanstack/react-form';
import { useAuth } from '../auth/useAuth.ts';

export default function SignUpPage() {
    const { signUp, isSigningUp, signUpError } = useAuth();
    const navigate = useNavigate();

    const form = useForm({
        defaultValues: { username: '', email: '', password: '' },
        onSubmit: async ({ value }) => {
            try {
                await signUp(value);
                navigate('/', { replace: true });
            } catch {
                // error is surfaced through signUpError
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
                <img
                    src="/logo/cairo-logo.png"
                    alt="cairo"
                    className="mx-auto h-10 w-auto"
                />
                <h1 className="mt-6 text-2xl font-semibold text-neutral-900">Create account</h1>
                <p className="mt-1 text-sm text-neutral-500">
                    Sign up to start storing objects.
                </p>

                <div className="mt-6 space-y-4">
                    <form.Field name="username">
                        {(field) => (
                            <label className="block">
                                <span className="mb-1 block text-sm font-medium text-neutral-700">
                                    Username
                                </span>
                                <input
                                    type="text"
                                    autoComplete="username"
                                    minLength={3}
                                    maxLength={30}
                                    required
                                    value={field.state.value}
                                    onChange={(e) => field.handleChange(e.target.value)}
                                    className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm outline-none focus:border-sky-500"
                                    placeholder="lizzy"
                                />
                            </label>
                        )}
                    </form.Field>

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
                                    className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm outline-none focus:border-sky-500"
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
                                    autoComplete="new-password"
                                    minLength={8}
                                    maxLength={72}
                                    required
                                    value={field.state.value}
                                    onChange={(e) => field.handleChange(e.target.value)}
                                    className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm outline-none focus:border-sky-500"
                                    placeholder="••••••••"
                                />
                            </label>
                        )}
                    </form.Field>
                </div>

                {signUpError && (
                    <p className="mt-4 text-sm text-red-600">{signUpError.message}</p>
                )}

                <button
                    type="submit"
                    disabled={isSigningUp}
                    className="mt-6 w-full rounded-lg bg-sky-500 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-sky-600 disabled:cursor-not-allowed disabled:opacity-60"
                >
                    {isSigningUp ? 'Creating account...' : 'Sign up'}
                </button>

                <p className="mt-4 text-center text-sm text-neutral-500">
                    Already have an account?{' '}
                    <Link className="font-medium text-sky-600 hover:underline" to="/signin">
                        Sign in
                    </Link>
                </p>
            </form>
        </div>
    );
}