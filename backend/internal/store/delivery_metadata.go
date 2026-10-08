package store

import (
	"context"
	"fmt"
)

func (s *Store) UpdateContact(ctx context.Context, profile, id string, c Contact) (Contact, error) {
	err := s.pool.QueryRow(ctx, `UPDATE contacts SET name=$3,role=$4,phone=$5,email=$6,note=$7 WHERE id=$1 AND profile_id=$2 RETURNING id,created_at`, id, profile, c.Name, c.Role, c.Phone, c.Email, c.Note).Scan(&c.ID, &c.CreatedAt)
	return c, domainNotFound(err)
}
func (s *Store) UpdateDocument(ctx context.Context, profile, id, name, kind string, asset *string) error {
	if asset != nil {
		var own bool
		err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE profile_id=$1 AND id=$2)`, profile, *asset).Scan(&own)
		if err != nil {
			return err
		}
		if !own {
			return ErrNotFound
		}
	}
	tag, err := s.pool.Exec(ctx, `UPDATE documents SET name=$3,kind=$4,asset_id=$5 WHERE id=$1 AND profile_id=$2`, id, profile, name, kind, asset)
	if err != nil {
		return fmt.Errorf("document metadata: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
