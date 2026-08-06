import { Outlet } from 'react-router';
import Navbar from './Navbar.tsx';

export default function ProtectedLayout() {
    return (
        <div className="min-h-screen bg-neutral-50">
            <Navbar />
            <Outlet />
        </div>
    );
}