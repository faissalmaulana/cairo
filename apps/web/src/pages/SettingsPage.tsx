import { useState } from 'react';
import { useAuth } from '../auth/useAuth.ts';
import ApiKeysSection from '../components/sections/ApiKeysSection.tsx';
import ProfileSection from '../components/sections/ProfileSection.tsx';
import SecuritySection from '../components/sections/SecuritySection.tsx';

const SECTIONS = [
    { id: 'profile', label: 'Profile' },
    { id: 'security', label: 'Security' },
    { id: 'api-keys', label: 'API Keys' },
] as const;

type SectionId = (typeof SECTIONS)[number]['id'];

export default function SettingsPage() {
    const { user } = useAuth();
    const [active, setActive] = useState<SectionId>('profile');

    return (
        <main className="mx-auto max-w-full px-4 py-10">
            <h2 className="text-2xl font-semibold text-neutral-900">Settings</h2>
            <p className="mt-1 text-sm text-neutral-500">
                Manage your account preferences.
            </p>

            <div className="mt-8 flex gap-8">
                <aside className="w-56 shrink-0">
                    <nav className="space-y-1">
                        {SECTIONS.map((section) => (
                            <button
                                key={section.id}
                                type="button"
                                onClick={() => setActive(section.id)}
                                className={`w-full rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors ${
                                    active === section.id
                                        ? 'bg-blue-500 text-white'
                                        : 'text-neutral-700 hover:bg-neutral-200'
                                }`}
                            >
                                {section.label}
                            </button>
                        ))}
                    </nav>
                </aside>

                <section className="flex-1 rounded-xl border border-neutral-200 bg-white p-6">
                    {active === 'profile' && (
                        <ProfileSection
                            username={user?.username ?? ''}
                            email={user?.email ?? ''}
                        />
                    )}
                    {active === 'security' && <SecuritySection />}
                    {active === 'api-keys' && <ApiKeysSection />}
                </section>
            </div>
        </main>
    );
}