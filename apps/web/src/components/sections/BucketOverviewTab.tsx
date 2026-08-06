import type { Bucket } from "../../api/buckets.ts";
import { formatDateTime } from "../../lib/format.ts";

export default function BucketOverviewTab({
  bucket,
  onDelete,
  isDeleting,
  onToggleVisibility,
  isToggling,
}: {
  bucket: Bucket;
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
          {formatDateTime(bucket.created_at)}
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
            value={formatDateTime(bucket.updated_at)}
            className="w-full cursor-not-allowed rounded-lg border border-neutral-300 bg-neutral-50 px-3 py-2 text-sm text-neutral-500 outline-none"
          />
        </label>
      </div>

      <div className="flex gap-2">
        <button
          type="button"
          onClick={onToggleVisibility}
          disabled={isToggling}
          className="rounded-lg bg-sky-500 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-sky-600 disabled:cursor-not-allowed disabled:opacity-60"
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

    </div>
  );
}
