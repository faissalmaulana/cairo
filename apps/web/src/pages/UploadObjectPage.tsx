import { useRef, useState, useSyncExternalStore } from "react";
import { Link, useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, Loader2, Upload, XCircle } from "lucide-react";
import { useAuth } from "../auth/useAuth.ts";
import { apiKeysApi } from "../api/apikeys.ts";
import { startUpload, uploadsStore } from "../api/uploads.ts";

function normalizeDirectory(value: string): string {
  return value.replace(/^\/+|\/+$/g, "");
}

export default function UploadObjectPage() {
  const { bucketName } = useParams();
  const { user } = useAuth();
  const [directory, setDirectory] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const activeUploads = useSyncExternalStore(
    uploadsStore.subscribe,
    uploadsStore.getSnapshot,
  );

  const apiKeysQuery = useQuery({
    queryKey: ["api-keys"],
    queryFn: apiKeysApi.list,
    enabled: user !== null,
  });

  const apiKey = apiKeysQuery.data?.[0]?.key ?? null;

  const normalizedDirectory = normalizeDirectory(directory);
  const key = file
    ? normalizedDirectory
      ? `${normalizedDirectory}/${file.name}`
      : file.name
    : "";

  const handleDrop = (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setDragOver(false);
    const dropped = event.dataTransfer.files[0];
    if (dropped) {
      setFile(dropped);
    }
  };

  const handleSubmit = (event: React.SubmitEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!user || !apiKey || !bucketName || !file) {
      return;
    }
    startUpload({
      accountId: user.id,
      apiKey,
      bucketName,
      key,
      file,
    });
    setDirectory("");
    setFile(null);
  };

  return (
    <main className="mx-auto flex max-w-full flex-col items-center px-4 py-10">
      <div className="w-full max-w-xl space-y-4">
        <Link
          to={`/buckets/${bucketName}`}
          className="block text-sm text-blue-600 hover:underline"
        >
          &larr; Back to {bucketName}
        </Link>

        <h2 className="text-2xl font-semibold text-neutral-900">Upload object</h2>
        <p className="text-sm text-neutral-500">
          Store a file into the "{bucketName}" bucket.
        </p>

        {!apiKey && (
          <p className="text-sm text-red-600">
            No API key available. Create one in Settings first.
          </p>
        )}

        {apiKey && (
          <form
            onSubmit={handleSubmit}
            className="space-y-6 rounded-xl border border-neutral-200 bg-white p-6"
          >
          <label className="block">
            <span className="mb-1 block text-sm font-medium text-neutral-700">
              Directory <span className="text-neutral-400">(optional)</span>
            </span>
            <input
              type="text"
              value={directory}
              onChange={(e) => setDirectory(e.target.value)}
              placeholder="e.g. images/avatars"
              className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm outline-none focus:border-blue-500"
            />
          </label>

          <div
            onClick={() => fileInputRef.current?.click()}
            onDragOver={(e) => {
              e.preventDefault();
              setDragOver(true);
            }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
            className={`flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed px-6 py-10 text-center transition-colors ${
              dragOver
                ? "border-blue-400 bg-blue-50"
                : "border-neutral-300 hover:border-blue-400"
            }`}
          >
            <Upload className="h-8 w-8 text-neutral-400" />
            <p className="mt-2 text-sm font-medium text-neutral-700">
              {file ? file.name : "Click or drop a file to upload"}
            </p>
            {file && (
              <p className="mt-1 text-xs text-neutral-500">
                {(file.size / 1024).toFixed(1)} KB
              </p>
            )}
            <input
              ref={fileInputRef}
              type="file"
              className="hidden"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
          </div>

          {key && (
            <p className="text-xs text-neutral-500">
              Will be stored at:{" "}
              <code className="font-mono text-neutral-800">{key}</code>
            </p>
          )}

          <button
            type="submit"
            disabled={!file}
            className="w-full rounded-lg bg-blue-500 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-60"
          >
            Upload
          </button>
        </form>
      )}

      {activeUploads.length > 0 && (
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-neutral-900">Uploads</h3>
          {activeUploads.map((upload) => (
            <div
              key={upload.id}
              className="rounded-xl border border-neutral-200 bg-white p-4"
            >
              <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-neutral-900">
                    {upload.fileName}
                  </p>
                  <p className="truncate font-mono text-xs text-neutral-500">
                    {upload.key}
                  </p>
                </div>
                {upload.status === "uploading" && (
                  <Loader2 className="h-5 w-5 shrink-0 animate-spin text-blue-500" />
                )}
                {upload.status === "success" && (
                  <CheckCircle2 className="h-5 w-5 shrink-0 text-green-500" />
                )}
                {upload.status === "error" && (
                  <XCircle className="h-5 w-5 shrink-0 text-red-500" />
                )}
              </div>
              <div className="mt-2 h-2 overflow-hidden rounded-full bg-neutral-100">
                <div
                  className={`h-full rounded-full transition-all ${
                    upload.status === "error" ? "bg-red-400" : "bg-blue-500"
                  }`}
                  style={{ width: `${upload.progress}%` }}
                />
              </div>
              <p className="mt-1 text-xs text-neutral-500">
                {upload.status === "error"
                  ? upload.error
                  : `${upload.progress}%`}
              </p>
            </div>
          ))}
        </div>
      )}
      </div>
    </main>
  );
}
