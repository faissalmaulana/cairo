import { useState } from "react";
import { useNavigate } from "react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "@tanstack/react-form";
import { Plus, ArrowUpRight, Box } from "lucide-react";
import toast from "react-hot-toast";
import { useAuth } from "../auth/useAuth.ts";
import { bucketsApi } from "../api/buckets.ts";
import type { Bucket } from "../api/buckets.ts";
import { apiKeysApi } from "../api/apikeys.ts";
import { formatDateTime } from "../lib/format.ts";

const BUCKET_NAME_RE = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/;

function validateBucketName(name: string): string | undefined {
  if (!name) {
    return "Bucket name is required";
  }
  if (name.length < 3 || name.length > 63) {
    return "Bucket name must be 3-63 characters";
  }
  if (!BUCKET_NAME_RE.test(name)) {
    return "Lowercase letters, numbers, and hyphens only";
  }
  return undefined;
}

export default function BucketKanban() {
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [creating, setCreating] = useState(false);

  const apiKeysQuery = useQuery({
    queryKey: ["api-keys"],
    queryFn: apiKeysApi.list,
    enabled: user !== null,
  });

  const apiKey = apiKeysQuery.data?.[0]?.key ?? null;

  const bucketsQuery = useQuery({
    queryKey: ["buckets"],
    queryFn: () => bucketsApi.list(user!.id, apiKey!),
    enabled: user !== null && apiKey !== null,
  });

  const createBucketMutation = useMutation({
    mutationFn: (name: string) => bucketsApi.create(user!.id, apiKey!, name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      toast.success("Bucket created");
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const form = useForm({
    defaultValues: { name: "" },
    validators: {
      onChange: ({ value }) => validateBucketName(value.name),
      onSubmit: ({ value }) => validateBucketName(value.name),
    },
    onSubmit: async ({ value }) => {
      await createBucketMutation.mutateAsync(value.name);
      form.reset();
      setCreating(false);
    },
  });

  return (
    <section className="flex w-full flex-col items-center">
      <h3 className="text-xl font-semibold text-neutral-900">Buckets</h3>
      <p className="mt-1 text-sm text-neutral-500">
        Your object-storage buckets, ready for uploads.
      </p>

      {!user || !apiKey ? (
        <div className="mt-6 w-full max-w-md rounded-xl border border-neutral-200 bg-white p-6 text-center">
          {apiKeysQuery.isPending && (
            <p className="text-sm text-neutral-500">Loading API keys...</p>
          )}
          {apiKeysQuery.error && (
            <p className="text-sm text-red-600">{apiKeysQuery.error.message}</p>
          )}
          {!apiKeysQuery.isPending &&
            !apiKeysQuery.error &&
            (apiKeysQuery.data?.length ?? 0) === 0 && (
              <p className="text-sm text-neutral-500">
                No API key yet. Create one in Settings to load your buckets.
              </p>
            )}
        </div>
      ) : (
        <>
          {bucketsQuery.isPending && (
            <p className="mt-6 text-center text-sm text-neutral-500">
              Loading buckets...
            </p>
          )}
          {bucketsQuery.error && (
            <p className="mt-6 text-center text-sm text-red-600">
              {bucketsQuery.error.message}
            </p>
          )}
          {!bucketsQuery.isPending && !bucketsQuery.error && (
            <div className="mt-8 overflow-x-auto">
              {creating ? (
                <div className="mx-auto w-148 max-w-full">
                  <form
                    onSubmit={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      form.handleSubmit();
                    }}
                    className="rounded-xl border border-dashed border-neutral-300 bg-white p-6"
                  >
                    <h4 className="font-semibold text-neutral-900">New bucket</h4>
                    <form.Field name="name">
                      {(field) => (
                        <div className="mt-3">
                          <input
                            type="text"
                            autoComplete="off"
                            placeholder="bucket-name"
                            value={field.state.value}
                            onChange={(e) => field.handleChange(e.target.value)}
                            className="w-full rounded-lg border border-neutral-300 px-3 py-2.5 text-sm outline-none focus:border-sky-500"
                          />
                          {field.state.meta.errors.length > 0 && (
                            <p className="mt-1 text-xs text-red-600">
                              {field.state.meta.errors[0]}
                            </p>
                          )}
                        </div>
                      )}
                    </form.Field>
                    <div className="mt-4 flex gap-2">
                      <button
                        type="submit"
                        disabled={createBucketMutation.isPending}
                        className="flex-1 rounded-lg bg-sky-500 px-3 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-sky-600 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        {createBucketMutation.isPending
                          ? "Creating..."
                          : "Create bucket"}
                      </button>
                      <button
                        type="button"
                        onClick={() => setCreating(false)}
                        className="rounded-lg border border-neutral-300 px-3 py-2.5 text-sm text-neutral-700 hover:bg-neutral-100"
                      >
                        Cancel
                      </button>
                    </div>
                  </form>
                </div>
              ) : (
                <div className="mx-auto flex w-max flex-wrap items-center justify-center gap-4 pb-2">
                  {bucketsQuery.data?.length === 0 && (
                    <p className="text-sm text-neutral-500">
                      No buckets yet. Create your first one.
                    </p>
                  )}
                  {bucketsQuery.data?.map((bucket) => (
                    <BucketCard key={bucket.id} bucket={bucket} />
                  ))}
                  <button
                    type="button"
                    onClick={() => setCreating(true)}
                    title="Create bucket"
                    aria-label="Create bucket"
                    className="flex h-44 w-72 shrink-0 items-center justify-center rounded-2xl border border-dashed border-neutral-300 bg-white text-neutral-500 transition-colors hover:border-sky-400 hover:text-sky-500"
                  >
                    <Plus className="h-7 w-7" />
                  </button>
                </div>
              )}
            </div>
          )}
        </>
      )}
    </section>
  );
}

function BucketCard({ bucket }: { bucket: Bucket }) {
  const navigate = useNavigate();

  return (
    <button
      type="button"
      onClick={() => navigate(`/buckets/${bucket.name}`)}
      title={`Open ${bucket.name}`}
      className="group flex h-44 w-72 shrink-0 flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-5 text-left shadow-sm transition-all hover:border-sky-400 hover:shadow-md"
    >
      <span className="flex items-start gap-3">
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-sky-50 text-sky-500">
          <Box className="h-5 w-5" />
        </span>
        <span className="min-w-0">
          <span className="block truncate font-semibold text-neutral-900">
            {bucket.name}
          </span>
          <span className="mt-1 block text-xs text-neutral-500">
            Created {formatDateTime(bucket.created_at)}
          </span>
        </span>
      </span>
      <span className="flex items-center justify-between">
        <span
          className={`rounded-full px-2.5 py-1 text-xs font-medium ${
            bucket.visibility === "public"
              ? "bg-green-100 text-green-700"
              : "bg-neutral-100 text-neutral-600"
          }`}
        >
          {bucket.visibility}
        </span>
        <ArrowUpRight className="h-4 w-4 text-neutral-400 transition-colors group-hover:text-sky-500" />
      </span>
    </button>
  );
}
