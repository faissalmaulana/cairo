import { Fragment, useState } from "react";
import { Link, useParams } from "react-router";
import { File, Folder, FolderOpen, Search, Trash2, Upload } from "lucide-react";
import type { ObjectMetadata } from "../../api/buckets.ts";

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
  objects,
  isPending,
  error,
  onDeleteObject,
  isDeletingObject,
  deletingObjectKey,
}: {
  objects: ObjectMetadata[] | undefined;
  isPending: boolean;
  error: Error | null;
  onDeleteObject: (key: string) => void;
  isDeletingObject: boolean;
  deletingObjectKey: string | null;
}) {
  const { bucketName } = useParams();
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

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
            className="flex shrink-0 items-center gap-2 rounded-lg bg-blue-500 px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-blue-600"
          >
            <Upload className="h-4 w-4" />
            Upload
          </Link>
        </div>
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
                <th className="px-3 py-2 font-medium">Size</th>
                <th className="px-3 py-2 font-medium">Uploaded at</th>
                <th className="px-3 py-2 font-medium">Delete</th>
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
              {objects && (
                <TreeRows
                  nodes={buildTree(objects)}
                  depth={0}
                  expanded={expanded}
                  onToggle={toggleExpanded}
                  onDeleteObject={onDeleteObject}
                  isDeletingObject={isDeletingObject}
                  deletingObjectKey={deletingObjectKey}
                />
              )}
            </tbody>
          </table>
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
  onDeleteObject,
  isDeletingObject,
  deletingObjectKey,
}: {
  nodes: TreeNode[];
  depth: number;
  expanded: Set<string>;
  onToggle: (path: string) => void;
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
                    <FolderOpen className="h-4 w-4 text-blue-500" />
                  ) : (
                    <Folder className="h-4 w-4 text-blue-500" />
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
                onDeleteObject={onDeleteObject}
                isDeletingObject={isDeletingObject}
                deletingObjectKey={deletingObjectKey}
              />
            )}
          </Fragment>
        ) : (
          <tr
            key={node.key}
            className="border-b border-neutral-100 hover:bg-neutral-50"
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
            <td className="px-3 py-2 text-neutral-400">—</td>
            <td className="px-3 py-2">
              <button
                type="button"
                onClick={() => onDeleteObject(node.key)}
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