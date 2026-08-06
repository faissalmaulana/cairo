import { useAuth } from '../auth/useAuth.ts';

export default function SettingsPage() {
    const { user } = useAuth();

    return (
        <main className="mx-auto max-w-full px-4 py-10">
            <h2 className="text-2xl font-semibold text-neutral-900">Settings</h2>
            <p className="mt-1 text-sm text-neutral-500">
                Account settings for {user?.username}.
            </p>
            <div className="mt-8 rounded-xl border border-neutral-200 bg-white p-6">
                <p className="text-sm text-neutral-600">
                    Settings placeholder — coming soon.
                </p>
            </div>
        </main>
    );
}