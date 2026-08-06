import { Link } from 'react-router';
import { useAuth } from '../auth/useAuth.ts';

export default function Avatar() {
    const { user } = useAuth();
    const initial = user?.username.charAt(0).toUpperCase() ?? '?';

    return (
        <Link
            to="/settings"
            title={`Settings for ${user?.username ?? ''}`}
            className="flex h-9 w-9 items-center justify-center rounded-full bg-blue-500 text-sm font-semibold text-white transition-colors hover:bg-blue-600"
        >
            {initial}
        </Link>
    );
}