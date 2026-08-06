import { Fragment, useState } from "react";
import { Link, useParams } from "react-router";
import {
  Download,
  File,
  Folder,
  FolderOpen,
  Search,
  Trash2,
  Upload,
  X,
} from "lucide-react";
import { bucketsApi } from "../../api/buckets.ts";
import type { ObjectMetadata } from "../../api/buckets.ts";
import { formatDateTime } from "../../lib/format.ts";

interface FileNode {
  type: "file";
  key: string;
  name: string;
  meta: ObjectMetadata;
}

interface DirNode {
  type: "dir";
  name: string;
  path: string;
  children: TreeNode[];
}

type TreeNode = FileNode | DirNode;

function buildTree(objects: ObjectMetadata[]): TreeNode[] {
  const root: DirNode = { type: "dir", name: "", path: "", children: [] };

  for (const object of objects) {
    const segments = object.key.split("/").filter(Boolean);
    let dir = root;
    let currentPath = "";
    for (let i = 0; i < segments.length - 1; i++) {
      currentPath = currentPath
        ? `${currentPath}/${segments[i]}`
        : segments[i];
      let child = dir.children.find(
        (node): node is DirNode =>
          node.type === "dir" && node.name === segments[i],
      );
      if (!child) {
        child = { type: "dir", name: segments[i], path: currentPath, children: [] };
        dir.children.push(child);
      }
      dir = child;
    }
    dir.children.push({
      type: "file",
      key: object.key,
      name: segments[segments.length - 1],
      meta: object,
    });
  }

  const sortNodes = (nodes: TreeNode[]): TreeNode[] =>
    nodes.sort((a, b) => {
      if (a.type !== b.type) {
        return a.type === "dir" ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    });

  for (const node of root.children) {
    if (node.type === "dir") {
      sortNodes(node.children);
    }
  }
  return sortNodes(root.children);
}

function formatBytes(size: number): string {
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

export default function BucketFilesTab({
  accountId,
  apiKey,
  objects,
  isPending,
  error,
  onDeleteObject,
  isDeletingObject,
  deletingObjectKey,
}: {
  accountId: string;
  apiKey: string;
  objects: ObjectMetadata[] | undefined;
  isPending: boolean;
  error: Error | null;
  onDeleteObject: (key: string) => void;
  isDeletingObject: boolean;
  deletingObjectKey: string | null;
}) {
  const { bucketName } = useParams();
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [query, setQuery] = useState("");
  const [preview, setPreview] = useState<FileNode | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewContentType, setPreviewContentType] = useState<string | null>(
    null,
  );
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [isLoadingPreview, setIsLoadingPreview] = useState(false);

  const toggleExpanded = (path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  };

  const searchQuery = query.trim().toLowerCase();
  const filteredObjects =
    searchQuery === ""
      ? objects
      : objects?.filter((object) =>
          object.key.toLowerCase().includes(searchQuery),
        );
  const tree =
    filteredObjects === undefined ? undefined : buildTree(filteredObjects);

  const openPreview = async (node: FileNode) => {
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
    }
    setPreview(node);
    setPreviewUrl(null);
    setPreviewContentType(null);
    setPreviewError(null);
    setIsLoadingPreview(true);
    try {
      const { blob, contentType } = await bucketsApi.download(
        accountId,
        apiKey,
        bucketName!,
        node.key,
      );
      setPreviewContentType(contentType);
      setPreviewUrl(URL.createObjectURL(blob));
    } catch (err) {
      setPreviewError(
        err instanceof Error ? err.message : "Failed to load object",
      );
    } finally {
      setIsLoadingPreview(false);
    }
  };

  const closePreview = () => {
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
    }
    setPreview(null);
    setPreviewUrl(null);
    setPreviewContentType(null);
    setPreviewError(null);
  };

  return (
    <div className="space-y-4">
      <div>
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-2xl font-semibold text-neutral-900">Files</h2>
            <p className="mt-1 text-sm text-neutral-500">
              Objects stored in this bucket.
            </p>
          </div>
          <Link
            to={`/buckets/${bucketName}/upload`}
            className="flex shrink-0 items-center gap-2 rounded-lg bg-sky-500 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-sky-600"
          >
            <Upload className="h-4 w-4" />
            Upload
          </Link>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <div className="relative w-full">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
          <input
            type="search"
            placeholder="Search objects..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full rounded-lg border border-neutral-300 py-2 pl-9 pr-3 text-sm outline-none focus:border-sky-500"
          />
        </div>
      </div>

      {isPending && <p className="text-sm text-neutral-500">Loading objects...</p>}
      {error && <p className="text-sm text-red-600">{error.message}</p>}

      {!isPending && !error && (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b border-neutral-200 text-left text-xs uppercase tracking-wide text-neutral-500">
                <th className="px-3 py-2 font-medium">Name</th>
                <th className="px-3 py-2 font-medium">Size</th>
                <th className="px-3 py-2 font-medium">Uploaded at</th>
                <th className="px-3 py-2 font-medium">Delete</th>
              </tr>
            </thead>
            <tbody>
              {tree === undefined ? null : tree.length === 0 ? (
                <tr>
                  <td
                    colSpan={4}
                    className="px-3 py-6 text-center text-neutral-400"
                  >
                    {searchQuery === ""
                      ? "No objects yet."
                      : "No objects match your search."}
                  </td>
                </tr>
              ) : null}
              {tree && (
                <TreeRows
                  nodes={tree}
                  depth={0}
                  expanded={expanded}
                  onToggle={toggleExpanded}
                  onOpenFile={openPreview}
                  onDeleteObject={onDeleteObject}
                  isDeletingObject={isDeletingObject}
                  deletingObjectKey={deletingObjectKey}
                />
              )}
            </tbody>
          </table>
        </div>
      )}

      {preview && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={closePreview}
        >
          <div
            className="flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden rounded-xl bg-white shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-start justify-between gap-4 border-b border-neutral-200 px-4 py-3">
              <div className="min-w-0">
                <h3 className="truncate font-semibold text-neutral-900">
                  {preview.name}
                </h3>
                <p className="truncate font-mono text-xs text-neutral-500">
                  {preview.key} · {formatBytes(preview.meta.size)}
                </p>
              </div>
              <button
                type="button"
                onClick={closePreview}
                title="Close"
                aria-label="Close"
                className="rounded p-1 text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="min-h-0 flex-1 overflow-auto bg-neutral-100">
              {isLoadingPreview && (
                <p className="p-4 text-sm text-neutral-500">
                  Loading preview...
                </p>
              )}
              {previewError && (
                <p className="p-4 text-sm text-red-600">{previewError}</p>
              )}
              {previewUrl && previewContentType?.startsWith("image/") && (
                <img
                  src={previewUrl}
                  alt={preview.name}
                  className="mx-auto max-h-full max-w-full"
                />
              )}
              {previewUrl && !previewContentType?.startsWith("image/") && (
                <iframe
                  src={previewUrl}
                  title={preview.name}
                  className="h-full w-full"
                />
              )}
            </div>
            <div className="border-t border-neutral-200 px-4 py-3">
              {previewUrl && (
                <a
                  href={previewUrl}
                  download={preview.name}
                  className="inline-flex items-center gap-2 rounded-lg bg-sky-500 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-sky-600"
                >
                  <Download className="h-4 w-4" />
                  Download
                </a>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function TreeRows({
  nodes,
  depth,
  expanded,
  onToggle,
  onOpenFile,
  onDeleteObject,
  isDeletingObject,
  deletingObjectKey,
}: {
  nodes: TreeNode[];
  depth: number;
  expanded: Set<string>;
  onToggle: (path: string) => void;
  onOpenFile: (node: FileNode) => void;
  onDeleteObject: (key: string) => void;
  isDeletingObject: boolean;
  deletingObjectKey: string | null;
}) {
  return (
    <>
      {nodes.map((node) =>
        node.type === "dir" ? (
          <Fragment key={node.path}>
            <tr
              className="cursor-pointer border-b border-neutral-100 hover:bg-neutral-50"
              onClick={() => onToggle(node.path)}
            >
              <td
                className="py-2 font-medium text-neutral-900"
                style={{ paddingLeft: `${8 + depth * 20}px` }}
              >
                <span className="flex items-center gap-1">
                  {expanded.has(node.path) ? (
                    <FolderOpen className="h-4 w-4 text-sky-500" />
                  ) : (
                    <Folder className="h-4 w-4 text-sky-500" />
                  )}
                  {node.name}
                </span>
              </td>
              <td className="px-3 py-2 text-neutral-400">—</td>
              <td className="px-3 py-2 text-neutral-400">—</td>
              <td className="px-3 py-2" />
            </tr>
            {expanded.has(node.path) && (
              <TreeRows
                nodes={node.children}
                depth={depth + 1}
                expanded={expanded}
                onToggle={onToggle}
                onOpenFile={onOpenFile}
                onDeleteObject={onDeleteObject}
                isDeletingObject={isDeletingObject}
                deletingObjectKey={deletingObjectKey}
              />
            )}
          </Fragment>
        ) : (
          <tr
            key={node.key}
            className="cursor-pointer border-b border-neutral-100 hover:bg-neutral-50"
            onClick={() => onOpenFile(node)}
          >
            <td
              className="py-2 font-medium text-neutral-900"
              style={{ paddingLeft: `${8 + depth * 20}px` }}
            >
              <span className="flex items-center gap-1">
                <File className="h-4 w-4 text-neutral-400" />
                {node.name}
              </span>
            </td>
            <td className="px-3 py-2 text-neutral-700">
              {formatBytes(node.meta.size)}
            </td>
            <td className="px-3 py-2 text-neutral-400">
              {formatDateTime(node.meta.created_at)}
            </td>
            <td className="px-3 py-2">
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  onDeleteObject(node.key);
                }}
                disabled={isDeletingObject && deletingObjectKey === node.key}
                title={`Delete ${node.name}`}
                aria-label={`Delete ${node.name}`}
                className="rounded p-1 text-red-600 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </td>
          </tr>
        ),
      )}
    </>
  );
}
