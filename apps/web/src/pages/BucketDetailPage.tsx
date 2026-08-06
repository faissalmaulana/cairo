import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FolderOpen, LayoutDashboard } from "lucide-react";
import toast from "react-hot-toast";
import { useAuth } from "../auth/useAuth.ts";
import { bucketsApi } from "../api/buckets.ts";
import { apiKeysApi } from "../api/apikeys.ts";
import BucketFilesTab from "../components/sections/BucketFilesTab.tsx";
import BucketOverviewTab from "../components/sections/BucketOverviewTab.tsx";

const TABS = [
  { id: "overview", label: "Overview", icon: LayoutDashboard },
  { id: "files", label: "Files", icon: FolderOpen },
] as const;

type TabId = (typeof TABS)[number]["id"];

export default function BucketDetailPage() {
  const { bucketName } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<TabId>("overview");

  const apiKeysQuery = useQuery({
    queryKey: ["api-keys"],
    queryFn: apiKeysApi.list,
    enabled: user !== null,
  });

  const apiKey = apiKeysQuery.data?.[0]?.key ?? null;
  const ready = user !== null && apiKey !== null && bucketName !== undefined;

  const bucketQuery = useQuery({
    queryKey: ["bucket", bucketName],
    queryFn: () => bucketsApi.get(user!.id, apiKey!, bucketName!),
    enabled: ready,
  });

  const objectsQuery = useQuery({
    queryKey: ["objects", bucketName],
    queryFn: () => bucketsApi.listObjects(user!.id, apiKey!, bucketName!),
    enabled: ready,
  });

  const deleteMutation = useMutation({
    mutationFn: () => bucketsApi.remove(user!.id, apiKey!, bucketName!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      toast.success("Bucket deleted");
      navigate("/", { replace: true });
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const visibilityMutation = useMutation({
    mutationFn: (setToPublic: boolean) =>
      bucketsApi.setVisibility(user!.id, apiKey!, bucketName!, setToPublic),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bucket", bucketName] });
      toast.success("Bucket visibility updated");
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const deleteObjectMutation = useMutation({
    mutationFn: (key: string) =>
      bucketsApi.removeObject(user!.id, apiKey!, bucketName!, key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["objects", bucketName] });
      toast.success("Object deleted");
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const handleDeleteObject = (key: string) => {
    if (window.confirm(`Delete object "${key}"? This cannot be undone.`)) {
      deleteObjectMutation.mutate(key);
    }
  };

  const handleDelete = () => {
    if (window.confirm(`Delete bucket "${bucketName}"? This cannot be undone.`)) {
      deleteMutation.mutate();
    }
  };

  const bucket = bucketQuery.data;

  return (
    <main className="mx-auto max-w-full px-4 py-10">
      <Link to="/" className="text-sm text-sky-600 hover:underline">
        &larr; Back to buckets
      </Link>

      <div className="mt-4 flex gap-8">
        <aside className="w-56 shrink-0">
          <nav className="space-y-1">
            {TABS.map((tab) => (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveTab(tab.id)}
                className={`flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors ${
                  activeTab === tab.id
                    ? "bg-sky-500 text-white"
                    : "text-neutral-700 hover:bg-neutral-200"
                }`}
              >
                <tab.icon className="h-4 w-4" />
                {tab.label}
              </button>
            ))}
          </nav>
        </aside>

        <section className="min-w-0 flex-1 rounded-xl border border-neutral-200 bg-white p-6">
          {bucketQuery.isPending && (
            <p className="text-sm text-neutral-500">Loading bucket...</p>
          )}
          {bucketQuery.error && (
            <p className="text-sm text-red-600">{bucketQuery.error.message}</p>
          )}
          {bucket && activeTab === "overview" && (
            <BucketOverviewTab
              bucket={bucket}
              onDelete={handleDelete}
              isDeleting={deleteMutation.isPending}
              isToggling={visibilityMutation.isPending}
              onToggleVisibility={() =>
                visibilityMutation.mutate(bucket.visibility !== "public")
              }
            />
          )}
          {bucket && activeTab === "files" && (
            <BucketFilesTab
              accountId={user!.id}
              apiKey={apiKey!}
              objects={objectsQuery.data}
              isPending={objectsQuery.isPending}
              error={objectsQuery.error}
              onDeleteObject={handleDeleteObject}
              isDeletingObject={deleteObjectMutation.isPending}
              deletingObjectKey={deleteObjectMutation.variables ?? null}
            />
          )}
        </section>
      </div>
    </main>
  );
}