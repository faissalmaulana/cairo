import BucketKanban from "../components/BucketKanban.tsx";

export default function DashboardPage() {
    return (
        <main className="mx-auto flex w-full max-w-5xl flex-col items-center px-4 py-10">
            <BucketKanban />
        </main>
    );
}