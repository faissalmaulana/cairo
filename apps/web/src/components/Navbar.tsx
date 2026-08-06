import { Link, useNavigate } from 'react-router';
import { useAuth } from '../auth/useAuth.ts';
import Avatar from './Avatar.tsx';

export default function Navbar() {
    const { signOut } = useAuth();
    const navigate = useNavigate();

    const handleSignOut = async () => {
        try {
            await signOut();
        } finally {
            navigate('/signin', { replace: true });
        }
    };

    return (
        <header className="border-b border-neutral-200 bg-white">
            <div className="mx-auto flex max-w-full items-center justify-between px-4 py-4">
                <Link to="/" className="text-lg font-semibold text-neutral-900">
                    cairo
                </Link>
                <div className="flex items-center gap-3">
                    <Avatar />
                    <button
                        type="button"
                        onClick={handleSignOut}
                        className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm text-neutral-700 hover:bg-neutral-100"
                    >
                        Sign out
                    </button>
                </div>
            </div>
        </header>
    );
}