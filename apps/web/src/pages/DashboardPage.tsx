import { useAuth } from '../auth/useAuth.ts';

export default function DashboardPage() {
    const { user } = useAuth();

    return (
        <main className="mx-auto max-w-full px-4 py-10">
            <h2 className="text-2xl font-semibold text-neutral-900">
                Welcome, {user?.username}
            </h2>
            <p className="mt-1 text-sm text-neutral-500">{user?.email}</p>

            <div className="mt-8 rounded-xl border border-neutral-200 bg-white p-6">
                <p className="text-sm text-neutral-600">
                    Your account is ready. Buckets and objects management will live here.
                </p>
            </div>
        </main>
    );
}