package metadata

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/model"
	"github.com/faissalmaulana/cairo/internal/repository"
	"github.com/faissalmaulana/cairo/internal/variables"
	"github.com/google/uuid"
)

type SQLiteBucketRepository struct {
	db repository.DBTX
}

func NewSQLiteBucketRepository(db *sql.DB) *SQLiteBucketRepository {
	return &SQLiteBucketRepository{
		db: db,
	}
}

// WithTx returns a repository bound to the given transaction, so multiple
// repositories can participate in the same atomic unit of work.
func (br *SQLiteBucketRepository) WithTx(tx *sql.Tx) *SQLiteBucketRepository {
	return &SQLiteBucketRepository{
		db: tx,
	}
}

const bucketColumns = `id,name,owner_id,visibility,bucket_hash,created_at,updated_at`

// scanBucket scans a single row into a model.Bucket, converting the stored
// int timestamps and visibility back into their Go types.
func scanBucket(row *sql.Row) (model.Bucket, error) {
	var b model.Bucket
	var visibility int
	var createdAt, updatedAt int64

	err := row.Scan(
		&b.ID,
		&b.Name,
		&b.OwnerID,
		&visibility,
		&b.BucketHash,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return model.Bucket{}, err
	}

	b.Visibility = model.BucketVisibility(visibility)
	b.CreatedAt = time.Unix(createdAt, 0)
	b.UpdatedAt = time.Unix(updatedAt, 0)

	return b, nil
}

func (br *SQLiteBucketRepository) CreateBucket(ctx context.Context, newBucket model.Bucket) (string, error) {
	id := uuid.NewString()
	now := time.Now().Unix()

	query := `INSERT INTO buckets(id,name,owner_id,visibility,bucket_hash,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	_, err := br.db.ExecContext(
		queryctx,
		query,
		id,
		newBucket.Name,
		newBucket.OwnerID,
		newBucket.Visibility,
		helpers.HashName(id),
		now,
		now,
	)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (br *SQLiteBucketRepository) GetBucket(ctx context.Context, name string, ownerID string) (model.Bucket, error) {
	query := `SELECT ` + bucketColumns + ` FROM buckets WHERE name=? AND owner_id=?`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	bucket, err := scanBucket(br.db.QueryRowContext(queryctx, query, name, ownerID))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Bucket{}, ErrBucketNotFound
	}
	if err != nil {
		return model.Bucket{}, err
	}

	return bucket, nil
}

func (br *SQLiteBucketRepository) GetBucketByName(ctx context.Context, name string) (model.Bucket, error) {
	query := `SELECT ` + bucketColumns + ` FROM buckets WHERE name=?`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	bucket, err := scanBucket(br.db.QueryRowContext(queryctx, query, name))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Bucket{}, ErrBucketNotFound
	}
	if err != nil {
		return model.Bucket{}, err
	}

	return bucket, nil
}

func (br *SQLiteBucketRepository) UpdateBucket(ctx context.Context, name string, ownerID string, update model.UpdateBucketInput) error {
	query := `UPDATE buckets SET updated_at=?`
	args := []any{time.Now().Unix()}

	if update.Visibility != nil {
		query += `, visibility=?`
		args = append(args, *update.Visibility)
	}

	query += ` WHERE name=? AND owner_id=?`
	args = append(args, name, ownerID)

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	_, err := br.db.ExecContext(queryctx, query, args...)
	return err
}

func (br *SQLiteBucketRepository) ReplaceBucket(ctx context.Context, bucket model.Bucket) error {
	query := `UPDATE buckets SET name=?,owner_id=?,visibility=?,updated_at=? WHERE id=?`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	_, err := br.db.ExecContext(
		queryctx,
		query,
		bucket.Name,
		bucket.OwnerID,
		bucket.Visibility,
		time.Now().Unix(),
		bucket.ID,
	)
	return err
}

func (br *SQLiteBucketRepository) ListBuckets(ctx context.Context, ownerID string) ([]model.Bucket, error) {
	query := `SELECT ` + bucketColumns + ` FROM buckets WHERE owner_id=? ORDER BY created_at DESC`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	rows, err := br.db.QueryContext(queryctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buckets := make([]model.Bucket, 0)
	for rows.Next() {
		var b model.Bucket
		var visibility int
		var createdAt, updatedAt int64

		if err := rows.Scan(
			&b.ID,
			&b.Name,
			&b.OwnerID,
			&visibility,
			&b.BucketHash,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}

		b.Visibility = model.BucketVisibility(visibility)
		b.CreatedAt = time.Unix(createdAt, 0)
		b.UpdatedAt = time.Unix(updatedAt, 0)
		buckets = append(buckets, b)
	}

	return buckets, rows.Err()
}

func (br *SQLiteBucketRepository) DeleteBucket(ctx context.Context, name string, ownerID string) error {
	query := `DELETE FROM buckets WHERE name=? AND owner_id=?`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	result, err := br.db.ExecContext(queryctx, query, name, ownerID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrBucketNotFound
	}

	return nil
}
