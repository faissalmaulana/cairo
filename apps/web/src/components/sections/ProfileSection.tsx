export default function ProfileSection({
    username,
    email,
}: {
    username: string;
    email: string;
}) {
    const initial = username.charAt(0).toUpperCase();

    return (
        <div className="space-y-8">
            <div>
                <h3 className="text-lg font-semibold text-neutral-900">Profile</h3>
                <p className="mt-1 text-sm text-neutral-500">
                    Your public profile information.
                </p>
            </div>

            <div className="flex items-center gap-4">
                <div className="flex h-16 w-16 items-center justify-center rounded-full bg-blue-500 text-xl font-semibold text-white">
                    {initial}
                </div>
                <div>
                    <p className="text-sm font-medium text-neutral-900">Avatar</p>
                    <button
                        type="button"
                        disabled
                        title="Avatar upload is read-only for now"
                        className="mt-1 cursor-not-allowed rounded-lg border border-neutral-300 px-3 py-1.5 text-sm text-neutral-400"
                    >
                        Upload avatar
                    </button>
                    <p className="mt-1 text-xs text-neutral-400">
                        Avatar upload is not available yet.
                    </p>
                </div>
            </div>

            <div className="grid max-w-md gap-4">
                <label className="block">
                    <span className="mb-1 block text-sm font-medium text-neutral-700">
                        Username
                    </span>
                    <input
                        type="text"
                        readOnly
                        value={username}
                        className="w-full cursor-not-allowed rounded-lg border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-500 outline-none"
                    />
                </label>
                <label className="block">
                    <span className="mb-1 block text-sm font-medium text-neutral-700">
                        Email
                    </span>
                    <input
                        type="email"
                        readOnly
                        value={email}
                        className="w-full cursor-not-allowed rounded-lg border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-500 outline-none"
                    />
                </label>
            </div>
        </div>
    );
}