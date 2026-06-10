package store

import (
	"context"
	"database/sql"
	"time"
)

type UserSession struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	DeviceID     string    `json:"device_id"`
	FCMToken     string    `json:"fcm_token"`
	RefreshToken *string   `json:"refresh_token,omitempty"`
	Platform     *string   `json:"platform,omitempty"`
	AppVersion   *string   `json:"app_version,omitempty"`
	UserAgent    *string   `json:"user_agent,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserSessionStorage struct {
	db DBTX
}

func NewUserSessionStorage(db DBTX) *UserSessionStorage {
	return &UserSessionStorage{db: db}
}

func scanUserSession(row *sql.Row) (*UserSession, error) {
	var s UserSession
	var refresh, platform, appVer, ua sql.NullString
	err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.DeviceID,
		&s.FCMToken,
		&refresh,
		&platform,
		&appVer,
		&ua,
		&s.LastSeenAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if refresh.Valid {
		s.RefreshToken = &refresh.String
	}
	if platform.Valid {
		s.Platform = &platform.String
	}
	if appVer.Valid {
		s.AppVersion = &appVer.String
	}
	if ua.Valid {
		s.UserAgent = &ua.String
	}
	return &s, nil
}

func (s *UserSessionStorage) Upsert(ctx context.Context, row *UserSession) error {
	q := `
		INSERT INTO user_sessions (user_id, device_id, fcm_token, refresh_token, platform, app_version, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, device_id) DO UPDATE SET
			fcm_token = EXCLUDED.fcm_token,
			refresh_token = COALESCE(EXCLUDED.refresh_token, user_sessions.refresh_token),
			platform = COALESCE(EXCLUDED.platform, user_sessions.platform),
			app_version = COALESCE(EXCLUDED.app_version, user_sessions.app_version),
			user_agent = COALESCE(EXCLUDED.user_agent, user_sessions.user_agent),
			last_seen_at = now(),
			updated_at = now()
		RETURNING id, last_seen_at, created_at, updated_at`

	var refresh, platform, appVer, ua interface{}
	if row.RefreshToken != nil {
		refresh = *row.RefreshToken
	}
	if row.Platform != nil {
		platform = *row.Platform
	}
	if row.AppVersion != nil {
		appVer = *row.AppVersion
	}
	if row.UserAgent != nil {
		ua = *row.UserAgent
	}

	return s.db.QueryRowContext(ctx, q,
		row.UserID,
		row.DeviceID,
		row.FCMToken,
		refresh,
		platform,
		appVer,
		ua,
	).Scan(&row.ID, &row.LastSeenAt, &row.CreatedAt, &row.UpdatedAt)
}

func (s *UserSessionStorage) ListByUserID(ctx context.Context, userID int64) ([]UserSession, error) {
	q := `
		SELECT id, user_id, device_id, fcm_token, refresh_token, platform, app_version, user_agent,
		       last_seen_at, created_at, updated_at
		FROM user_sessions WHERE user_id = $1 ORDER BY updated_at DESC`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UserSession
	for rows.Next() {
		var r UserSession
		var refresh, platform, appVer, ua sql.NullString
		if err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.DeviceID,
			&r.FCMToken,
			&refresh,
			&platform,
			&appVer,
			&ua,
			&r.LastSeenAt,
			&r.CreatedAt,
			&r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if refresh.Valid {
			r.RefreshToken = &refresh.String
		}
		if platform.Valid {
			r.Platform = &platform.String
		}
		if appVer.Valid {
			r.AppVersion = &appVer.String
		}
		if ua.Valid {
			r.UserAgent = &ua.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *UserSessionStorage) GetByIDForUser(ctx context.Context, id, userID int64) (*UserSession, error) {
	q := `
		SELECT id, user_id, device_id, fcm_token, refresh_token, platform, app_version, user_agent,
		       last_seen_at, created_at, updated_at
		FROM user_sessions WHERE id = $1 AND user_id = $2`
	return scanUserSession(s.db.QueryRowContext(ctx, q, id, userID))
}

func (s *UserSessionStorage) UpdateFCM(ctx context.Context, id, userID int64, fcmToken string, refreshToken *string) error {
	q := `
		UPDATE user_sessions SET
			fcm_token = $3,
			refresh_token = COALESCE($4, refresh_token),
			last_seen_at = now(),
			updated_at = now()
		WHERE id = $1 AND user_id = $2`
	res, err := s.db.ExecContext(ctx, q, id, userID, fcmToken, refreshToken)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *UserSessionStorage) Delete(ctx context.Context, id, userID int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FCMTokensByCompanyID returns distinct non-empty FCM tokens for all users in a company.
func (s *UserSessionStorage) FCMTokensByCompanyID(ctx context.Context, companyID int64) ([]string, error) {
	q := `
		SELECT DISTINCT us.fcm_token
		FROM user_sessions us
		INNER JOIN users u ON u.id = us.user_id
		WHERE u.company_id = $1 AND us.fcm_token <> ''`
	rows, err := s.db.QueryContext(ctx, q, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// FCMTokensByUserID returns distinct non-empty tokens for push (all user devices).
func (s *UserSessionStorage) FCMTokensByUserID(ctx context.Context, userID int64) ([]string, error) {
	q := `SELECT DISTINCT fcm_token FROM user_sessions WHERE user_id = $1 AND fcm_token <> ''`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (s *UserSessionStorage) DeleteByFCMToken(ctx context.Context, fcmToken string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE fcm_token = $1`, fcmToken)
	return err
}
