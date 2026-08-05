-- +goose Up
-- Objects no longer reference buckets via a foreign key: bucket_id is a plain
-- string. Ownership is validated by the application layer (the bucket is looked
-- up by owner first), and deleting a bucket must delete its objects explicitly.
CREATE TABLE objects_new(
    id           TEXT PRIMARY KEY,
    bucket_id    TEXT NOT NULL,
    key          TEXT NOT NULL,
    path         TEXT NOT NULL,
    size         INTEGER NOT NULL,
    sha256sum    TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT ''
);

INSERT INTO objects_new(id,bucket_id,key,path,size,sha256sum,content_type)
    SELECT id,bucket_id,key,path,size,sha256sum,content_type FROM objects;

DROP TABLE objects;
ALTER TABLE objects_new RENAME TO objects;
CREATE INDEX idx_objects_bucket_id ON objects(bucket_id);

-- +goose Down
CREATE TABLE objects_old(
    id           TEXT PRIMARY KEY,
    bucket_id    TEXT NOT NULL REFERENCES buckets(id) ON DELETE CASCADE,
    key          TEXT NOT NULL,
    path         TEXT NOT NULL,
    size         INTEGER NOT NULL,
    sha256sum    TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT ''
);

INSERT INTO objects_old(id,bucket_id,key,path,size,sha256sum,content_type)
    SELECT id,bucket_id,key,path,size,sha256sum,content_type FROM objects;

DROP TABLE objects;
ALTER TABLE objects_old RENAME TO objects;
CREATE INDEX idx_objects_bucket_id ON objects(bucket_id);
