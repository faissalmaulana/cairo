import { createBrowserRouter, Navigate, RouterProvider } from 'react-router';
import type { ReactNode } from 'react';
import { AuthProvider } from './auth/AuthProvider.tsx';
import { useAuth } from './auth/useAuth.ts';
import ProtectedLayout from './components/ProtectedLayout.tsx';
import BucketDetailPage from './pages/BucketDetailPage.tsx';
import DashboardPage from './pages/DashboardPage.tsx';
import UploadObjectPage from './pages/UploadObjectPage.tsx';
import SettingsPage from './pages/SettingsPage.tsx';
import SignInPage from './pages/SignInPage.tsx';
import SignUpPage from './pages/SignUpPage.tsx';
import { Toaster } from 'react-hot-toast';

function RequireAuth({ children }: { children: ReactNode }) {
    const { isAuthenticated, isInitializing } = useAuth();
    if (isInitializing) {
        return null;
    }
    if (!isAuthenticated) {
        return <Navigate to="/signin" replace />;
    }
    return <>{children}</>;
}

function GuestOnly({ children }: { children: ReactNode }) {
    const { isAuthenticated, isInitializing } = useAuth();
    if (isInitializing) {
        return null;
    }
    if (isAuthenticated) {
        return <Navigate to="/" replace />;
    }
    return <>{children}</>;
}

const router = createBrowserRouter([
    {
        path: '/',
        element: (
            <RequireAuth>
                <ProtectedLayout />
            </RequireAuth>
        ),
        children: [
            { index: true, element: <DashboardPage /> },
            { path: 'settings', element: <SettingsPage /> },
            { path: 'buckets/:bucketName', element: <BucketDetailPage /> },
            {
                path: 'buckets/:bucketName/upload',
                element: <UploadObjectPage />,
            },
        ],
    },
    {
        path: '/signin',
        element: (
            <GuestOnly>
                <SignInPage />
            </GuestOnly>
        ),
    },
    {
        path: '/signup',
        element: (
            <GuestOnly>
                <SignUpPage />
            </GuestOnly>
        ),
    },
    {
        path: '*',
        element: <Navigate to="/" replace />,
    },
]);

function App() {
    return (
        <AuthProvider>
            <RouterProvider router={router} />
            <Toaster position='bottom-right'/>
        </AuthProvider>
    );
}

export default App;
