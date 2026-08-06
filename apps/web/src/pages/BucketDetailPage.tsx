import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FolderOpen, LayoutDashboard, Search } from "lucide-react";
import toast from "react-hot-toast";
import { useAuth } from "../auth/useAuth.ts";
import { bucketsApi } from "../api/buckets.ts";
import type { ObjectMetadata } from "../api/buckets.ts";
import { apiKeysApi } from "../api/apikeys.ts";

const TABS = [
  { id: "overview", label: "Overview", icon: LayoutDashboard },
  { id: "files", label: "Files", icon: FolderOpen },
] as const;

type TabId = (typeof TABS)[number]["id"];

function formatBytes(size: number): string {
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function fileNameFromKey(key: string): string {
  const segments = key.split("/");
  return segments[segments.length - 1];
}

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

  const handleDelete = () => {
    if (window.confirm(`Delete bucket "${bucketName}"? This cannot be undone.`)) {
      deleteMutation.mutate();
    }
  };

  const bucket = bucketQuery.data;

  return (
    <main className="mx-auto max-w-full px-4 py-10">
      <Link to="/" className="text-sm text-blue-600 hover:underline">
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
                    ? "bg-blue-500 text-white"
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
            <OverviewTab
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
            <FilesTab
              objects={objectsQuery.data}
              isPending={objectsQuery.isPending}
              error={objectsQuery.error}
            />
          )}
        </section>
      </div>
    </main>
  );
}

function OverviewTab({
  bucket,
  onDelete,
  isDeleting,
  onToggleVisibility,
  isToggling,
}: {
  bucket: { name: string; visibility: string; created_at: string; updated_at: string };
  onDelete: () => void;
  isDeleting: boolean;
  onToggleVisibility: () => void;
  isToggling: boolean;
}) {
  const isPublic = bucket.visibility === "public";

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl font-semibold text-neutral-900">{bucket.name}</h2>
        <p className="mt-1 text-sm text-neutral-500">
          {isPublic ? "Public bucket" : "Private bucket"} · created{" "}
          {new Date(bucket.created_at).toLocaleDateString()}
        </p>
      </div>

      <div className="grid max-w-md gap-4">
        <label className="block">
          <span className="mb-1 block text-sm font-medium text-neutral-700">
            Visibility
          </span>
          <input
            type="text"
            readOnly
            value={bucket.visibility}
            className="w-full cursor-not-allowed rounded-lg border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-500 outline-none"
          />
        </label>
        <label className="block">
          <span className="mb-1 block text-sm font-medium text-neutral-700">
            Updated at
          </span>
          <input
            type="text"
            readOnly
            value={new Date(bucket.updated_at).toLocaleString()}
            className="w-full cursor-not-allowed rounded-lg border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-500 outline-none"
          />
        </label>
      </div>

      <div className="flex gap-2">
        <button
          type="button"
          onClick={onToggleVisibility}
          disabled={isToggling}
          className="rounded-lg bg-blue-500 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isToggling ? "Updating..." : isPublic ? "Make private" : "Make public"}
        </button>
        <button
          type="button"
          onClick={onDelete}
          disabled={isDeleting}
          className="rounded-lg border border-red-300 px-3 py-2 text-sm text-red-600 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isDeleting ? "Deleting..." : "Delete bucket"}
        </button>
      </div>

      <div className="rounded-lg border border-dashed border-neutral-300 p-6 text-center text-sm text-neutral-400">
        Object aggregates placeholder — usage statistics coming soon.
      </div>
    </div>
  );
}

function FilesTab({
  objects,
  isPending,
  error,
}: {
  objects: ObjectMetadata[] | undefined;
  isPending: boolean;
  error: Error | null;
}) {
  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold text-neutral-900">Files</h2>
        <p className="mt-1 text-sm text-neutral-500">
          Objects stored in this bucket.
        </p>
      </div>

      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
          <input
            type="search"
            placeholder="Search objects..."
            className="w-full rounded-lg border border-neutral-300 py-2 pl-9 pr-3 text-sm outline-none focus:border-blue-500"
          />
        </div>
        <select className="rounded-lg border border-neutral-300 px-3 py-2 text-sm text-neutral-700 outline-none focus:border-blue-500">
          <option>All sizes</option>
          <option>Large</option>
          <option>Medium</option>
          <option>Small</option>
        </select>
      </div>

      {isPending && <p className="text-sm text-neutral-500">Loading objects...</p>}
      {error && <p className="text-sm text-red-600">{error.message}</p>}

      {!isPending && !error && (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b border-neutral-200 text-left text-xs uppercase tracking-wide text-neutral-500">
                <th className="px-3 py-2 font-medium">Name</th>
                <th className="px-3 py-2 font-medium">Route</th>
                <th className="px-3 py-2 font-medium">Size</th>
                <th className="px-3 py-2 font-medium">Uploaded at</th>
              </tr>
            </thead>
            <tbody>
              {objects?.length === 0 && (
                <tr>
                  <td
                    colSpan={4}
                    className="px-3 py-6 text-center text-neutral-400"
                  >
                    No objects yet.
                  </td>
                </tr>
              )}
              {objects?.map((object) => (
                <tr
                  key={object.id}
                  className="border-b border-neutral-100 hover:bg-neutral-50"
                >
                  <td className="px-3 py-2 font-medium text-neutral-900">
                    {fileNameFromKey(object.key)}
                  </td>
                  <td className="px-3 py-2 font-mono text-xs text-neutral-600">
                    {object.key}
                  </td>
                  <td className="px-3 py-2 text-neutral-700">
                    {formatBytes(object.size)}
                  </td>
                  <td className="px-3 py-2 text-neutral-400">—</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}