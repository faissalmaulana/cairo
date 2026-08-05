package metadata

import (
	"context"
	"database/sql"
	"errors"

	"github.com/faissalmaulana/cairo/internal/model"
	"github.com/faissalmaulana/cairo/internal/repository"
	"github.com/faissalmaulana/cairo/internal/variables"
	"github.com/google/uuid"
)

type SQLiteObjectRepository struct {
	db repository.DBTX
}

func NewSQLiteObjectRepository(db *sql.DB) *SQLiteObjectRepository {
	return &SQLiteObjectRepository{
		db: db,
	}
}

// WithTx returns a repository bound to the given transaction, so multiple
// repositories can participate in the same atomic unit of work.
func (or *SQLiteObjectRepository) WithTx(tx *sql.Tx) *SQLiteObjectRepository {
	return &SQLiteObjectRepository{
		db: tx,
	}
}

func scanObject(row *sql.Row) (model.Object, error) {
	var obj model.Object

	err := row.Scan(
		&obj.ID,
		&obj.BucketID,
		&obj.Key,
		&obj.Path,
		&obj.Size,
		&obj.Sha256sum,
		&obj.ContentType,
	)
	if err != nil {
		return model.Object{}, err
	}

	return obj, nil
}

func (or *SQLiteObjectRepository) CreateObject(ctx context.Context, object model.Object) (string, error) {
	id := uuid.NewString()

	query := `INSERT INTO objects(id,bucket_id,key,path,size,sha256sum,content_type) VALUES(?,?,?,?,?,?,?)`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	_, err := or.db.ExecContext(
		queryctx,
		query,
		id,
		object.BucketID,
		object.Key,
		object.Path,
		object.Size,
		object.Sha256sum,
		object.ContentType,
	)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (or *SQLiteObjectRepository) GetObject(ctx context.Context, bucketID, name string) (model.Object, error) {
	query := `SELECT id,bucket_id,key,path,size,sha256sum,content_type FROM objects WHERE bucket_id=? AND key=?`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	object, err := scanObject(or.db.QueryRowContext(queryctx, query, bucketID, name))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Object{}, ErrObjectNotFound
	}
	if err != nil {
		return model.Object{}, err
	}

	return object, nil
}

func (or *SQLiteObjectRepository) ListObjects(ctx context.Context, bucketID string) ([]model.Object, error) {
	query := `SELECT id,bucket_id,key,path,size,sha256sum,content_type FROM objects WHERE bucket_id=? ORDER BY key`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	rows, err := or.db.QueryContext(queryctx, query, bucketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	objects := make([]model.Object, 0)
	for rows.Next() {
		var obj model.Object

		if err := rows.Scan(
			&obj.ID,
			&obj.BucketID,
			&obj.Key,
			&obj.Path,
			&obj.Size,
			&obj.Sha256sum,
			&obj.ContentType,
		); err != nil {
			return nil, err
		}

		objects = append(objects, obj)
	}

	return objects, rows.Err()
}

func (or *SQLiteObjectRepository) DeleteObject(ctx context.Context, bucketID, name string) error {
	query := `DELETE FROM objects WHERE bucket_id=? AND key=?`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	result, err := or.db.ExecContext(queryctx, query, bucketID, name)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrObjectNotFound
	}

	return nil
}

func (or *SQLiteObjectRepository) DeleteObjectsByBucket(ctx context.Context, bucketID string) error {
	query := `DELETE FROM objects WHERE bucket_id=?`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	_, err := or.db.ExecContext(queryctx, query, bucketID)
	return err
}
