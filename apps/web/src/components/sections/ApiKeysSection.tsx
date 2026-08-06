import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import toast from "react-hot-toast";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiKeysApi } from "../../api/apikeys.ts";
import type { ApiKey } from "../../api/apikeys.ts";
import { formatDateTime } from "../../lib/format.ts";
import ConfirmDialog from "../ConfirmDialog.tsx";

const QUERY_KEY = ["api-keys"] as const;

export default function ApiKeysSection() {
  const queryClient = useQueryClient();
  const [revealed, setRevealed] = useState<Set<string>>(new Set());
  const [revokingId, setRevokingId] = useState<string | null>(null);

  const keysQuery = useQuery({
    queryKey: QUERY_KEY,
    queryFn: apiKeysApi.list,
  });

  const createMutation = useMutation({
    mutationFn: apiKeysApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      toast.success("API key created");
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const revokeMutation = useMutation({
    mutationFn: apiKeysApi.revoke,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      setRevokingId(null);
      toast.success("API key revoked");
    },
    onError: (error) => {
      setRevokingId(null);
      toast.error(error.message);
    },
  });

  const toggleReveal = (id: string) => {
    setRevealed((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleRevoke = (key: ApiKey) => {
    setRevokingId(key.id);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h3 className="text-lg font-semibold text-neutral-900">API Keys</h3>
          <p className="mt-1 text-sm text-neutral-500">
            Get your api keys to use it in your application.
          </p>
        </div>
        <button
          type="button"
          onClick={() => createMutation.mutate()}
          disabled={createMutation.isPending}
          className="rounded-lg bg-sky-500 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-sky-600 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {createMutation.isPending ? "Creating..." : "Create key"}
        </button>
      </div>

      {keysQuery.isPending && (
        <p className="text-sm text-neutral-500">Loading API keys...</p>
      )}

      {keysQuery.error && (
        <p className="text-sm text-red-600">{keysQuery.error.message}</p>
      )}

      {!keysQuery.isPending &&
        !keysQuery.error &&
        keysQuery.data?.length === 0 && (
          <div className="rounded-lg border border-dashed border-neutral-300 p-6 text-center text-sm text-neutral-400">
            No API keys yet. Create one to start calling the object-storage API.
          </div>
        )}

      {!keysQuery.isPending && !keysQuery.error && keysQuery.data && (
        <ul className="space-y-2">
          {keysQuery.data.map((key) => {
            const isRevealed = revealed.has(key.id);
            return (
              <li
                key={key.id}
                className="rounded-lg border border-neutral-200 px-4 py-3"
              >
                <div className="flex items-center gap-2">
                  <input
                    type={isRevealed ? "text" : "password"}
                    readOnly
                    value={key.key}
                    className="w-full cursor-default rounded-lg border border-neutral-300 bg-neutral-50 px-3 py-2 font-mono text-xs text-neutral-800 outline-none"
                  />
                  <button
                    type="button"
                    onClick={() => toggleReveal(key.id)}
                    title={isRevealed ? "Hide key" : "Show key"}
                    aria-label={isRevealed ? "Hide key" : "Show key"}
                    className="shrink-0 rounded-lg border border-neutral-300 p-2 text-neutral-600 transition-colors hover:bg-neutral-100"
                  >
                    {isRevealed ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </button>
                  <button
                    type="button"
                    onClick={() => handleRevoke(key)}
                    disabled={revokeMutation.isPending}
                    className="shrink-0 rounded-lg border border-red-300 px-3 py-1.5 text-sm text-red-600 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    Revoke
                  </button>
                </div>
                <p className="mt-1.5 text-xs text-neutral-500">
                  Created {formatDateTime(key.created_at)}
                  {key.last_used
                    ? ` · Last used ${formatDateTime(key.last_used)}`
                    : ""}
                </p>
              </li>
            );
          })}
        </ul>
      )}

      <ConfirmDialog
        open={revokingId !== null}
        title="Revoke API key"
        message={`Revoke API key ${revokingId ?? ""}? This cannot be undone.`}
        confirmLabel="Revoke"
        tone="danger"
        isConfirming={revokeMutation.isPending}
        onConfirm={() => {
          if (revokingId !== null) {
            revokeMutation.mutate(revokingId);
          }
        }}
        onCancel={() => setRevokingId(null)}
      />
    </div>
  );
}
