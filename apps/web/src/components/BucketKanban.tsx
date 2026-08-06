import { useState } from "react";
import { useNavigate } from "react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "@tanstack/react-form";
import { Plus } from "lucide-react";
import toast from "react-hot-toast";
import { useAuth } from "../auth/useAuth.ts";
import { bucketsApi } from "../api/buckets.ts";
import type { Bucket } from "../api/buckets.ts";
import { apiKeysApi } from "../api/apikeys.ts";

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
    <section className="mt-8">
      <h3 className="text-lg font-semibold text-neutral-900">Buckets</h3>
      <p className="mt-1 text-sm text-neutral-500">
        Your object-storage buckets, ready for uploads.
      </p>

      {!user || !apiKey ? (
        <div className="mt-6 rounded-xl border border-neutral-200 bg-white p-6">
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
            <p className="mt-6 text-sm text-neutral-500">Loading buckets...</p>
          )}
          {bucketsQuery.error && (
            <p className="mt-6 text-sm text-red-600">
              {bucketsQuery.error.message}
            </p>
          )}
          {!bucketsQuery.isPending && !bucketsQuery.error && (
            <div className="mt-6 overflow-x-auto">
              {creating ? (
                <div className="mx-auto w-full max-w-md">
                  <form
                    onSubmit={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      form.handleSubmit();
                    }}
                    className="rounded-xl border border-dashed border-neutral-300 bg-white p-4"
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
                            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm outline-none focus:border-blue-500"
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
                        className="flex-1 rounded-lg bg-blue-500 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        {createBucketMutation.isPending
                          ? "Creating..."
                          : "Create bucket"}
                      </button>
                      <button
                        type="button"
                        onClick={() => setCreating(false)}
                        className="rounded-lg border border-neutral-300 px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-100"
                      >
                        Cancel
                      </button>
                    </div>
                  </form>
                </div>
              ) : (
                <div className="mx-auto flex w-max gap-4 pb-2">
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
                    className="flex h-32 w-64 shrink-0 items-center justify-center rounded-xl border border-dashed border-neutral-300 bg-white text-neutral-500 transition-colors hover:border-blue-400 hover:text-blue-500"
                  >
                    <Plus className="h-6 w-6" />
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
      className="flex h-32 w-64 shrink-0 flex-col justify-between rounded-xl border border-neutral-200 bg-white p-4 text-left transition-colors hover:border-blue-400"
    >
      <span>
        <span className="block truncate font-semibold text-neutral-900">
          {bucket.name}
        </span>
        <span className="mt-1 block text-xs text-neutral-500">
          Created {new Date(bucket.created_at).toLocaleDateString()}
        </span>
      </span>
      <span
        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
          bucket.visibility === "public"
            ? "bg-green-100 text-green-700"
            : "bg-neutral-100 text-neutral-600"
        }`}
      >
        {bucket.visibility}
      </span>
    </button>
  );
}