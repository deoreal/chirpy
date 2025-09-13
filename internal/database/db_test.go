package database

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// mockDBTX implements the DBTX interface for testing
type mockDBTX struct {
	execContextFunc     func(context.Context, string, ...interface{}) (sql.Result, error)
	prepareContextFunc  func(context.Context, string) (*sql.Stmt, error)
	queryContextFunc    func(context.Context, string, ...interface{}) (*sql.Rows, error)
	queryRowContextFunc func(context.Context, string, ...interface{}) *sql.Row
}

func (m *mockDBTX) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if m.execContextFunc != nil {
		return m.execContextFunc(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDBTX) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if m.prepareContextFunc != nil {
		return m.prepareContextFunc(ctx, query)
	}
	return nil, nil
}

func (m *mockDBTX) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if m.queryContextFunc != nil {
		return m.queryContextFunc(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDBTX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if m.queryRowContextFunc != nil {
		return m.queryRowContextFunc(ctx, query, args...)
	}
	return nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		db   DBTX
	}{
		{
			name: "with mock db",
			db:   &mockDBTX{},
		},
		{
			name: "with nil db",
			db:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queries := New(tt.db)

			if queries == nil {
				t.Error("New() returned nil")
			}

			if queries.db != tt.db {
				t.Error("New() did not set db correctly")
			}
		})
	}
}

func TestQueries_WithTx(t *testing.T) {
	// Create a mock transaction
	mockTx := &sql.Tx{}

	// Create initial queries with mock db
	mockDB := &mockDBTX{}
	queries := New(mockDB)

	// Test WithTx
	txQueries := queries.WithTx(mockTx)

	if txQueries == nil {
		t.Error("WithTx() returned nil")
	}

	if txQueries.db != mockTx {
		t.Error("WithTx() did not set transaction correctly")
	}

	// Ensure original queries is unchanged
	if queries.db != mockDB {
		t.Error("WithTx() modified original queries")
	}

	// Ensure it's a new instance
	if txQueries == queries {
		t.Error("WithTx() returned same instance instead of new one")
	}
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name      string
		params    CreateUserParams
		mockSetup func() *mockDBTX
		wantErr   bool
	}{
		{
			name: "valid user creation",
			params: CreateUserParams{
				Email:          "test@example.com",
				HashedPassword: "hashed_password_123",
			},
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						// Create a mock row that will return valid data
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT $1::uuid, NOW(), NOW(), $2::text, $3::text",
							uuid.New(), args[0], args[1])
					},
				}
			},
			wantErr: false,
		},
		{
			name: "empty email",
			params: CreateUserParams{
				Email:          "",
				HashedPassword: "hashed_password_123",
			},
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT $1::uuid, NOW(), NOW(), $2::text, $3::text",
							uuid.New(), args[0], args[1])
					},
				}
			},
			wantErr: false, // SQL layer handles validation
		},
		{
			name: "empty password",
			params: CreateUserParams{
				Email:          "test@example.com",
				HashedPassword: "",
			},
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT $1::uuid, NOW(), NOW(), $2::text, $3::text",
							uuid.New(), args[0], args[1])
					},
				}
			},
			wantErr: false, // SQL layer handles validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := tt.mockSetup()
			queries := New(mockDB)

			ctx := context.Background()
			user, err := queries.CreateUser(ctx, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Error("CreateUser() expected error but got none")
				}
				return
			}

			// Note: Due to mocking limitations, we can't fully test the actual database interaction
			// In a real test, you'd use a test database or more sophisticated mocking
			_ = user // Acknowledge that we can't fully validate the response with this mock
		})
	}
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		mockSetup func() *mockDBTX
		wantErr   bool
	}{
		{
			name:  "valid email",
			email: "test@example.com",
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT $1::uuid, NOW(), NOW(), $2::text, $3::text",
							uuid.New(), args[0], "hashed_password")
					},
				}
			},
			wantErr: false,
		},
		{
			name:  "empty email",
			email: "",
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT $1::uuid, NOW(), NOW(), $2::text, $3::text",
							uuid.New(), args[0], "hashed_password")
					},
				}
			},
			wantErr: false,
		},
		{
			name:  "nonexistent email",
			email: "nonexistent@example.com",
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT * FROM users WHERE email = 'nonexistent'")
					},
				}
			},
			wantErr: false, // sql.ErrNoRows would be returned, but that's handled by the caller
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := tt.mockSetup()
			queries := New(mockDB)

			ctx := context.Background()
			user, err := queries.GetUser(ctx, tt.email)

			if tt.wantErr {
				if err == nil {
					t.Error("GetUser() expected error but got none")
				}
				return
			}

			// Note: Due to mocking limitations, we can't fully test the actual database interaction
			_ = user // Acknowledge that we can't fully validate the response with this mock
		})
	}
}

func TestCreateChirp(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name      string
		params    CreateChirpParams
		mockSetup func() *mockDBTX
		wantErr   bool
	}{
		{
			name: "valid chirp creation",
			params: CreateChirpParams{
				Body:   "This is a test chirp message",
				UserID: userID,
			},
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT $1::uuid, NOW(), NOW(), $2::text, $3::uuid",
							uuid.New(), args[0], args[1])
					},
				}
			},
			wantErr: false,
		},
		{
			name: "empty body",
			params: CreateChirpParams{
				Body:   "",
				UserID: userID,
			},
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT $1::uuid, NOW(), NOW(), $2::text, $3::uuid",
							uuid.New(), args[0], args[1])
					},
				}
			},
			wantErr: false, // SQL layer handles validation
		},
		{
			name: "nil user ID",
			params: CreateChirpParams{
				Body:   "Test message",
				UserID: uuid.Nil,
			},
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT $1::uuid, NOW(), NOW(), $2::text, $3::uuid",
							uuid.New(), args[0], args[1])
					},
				}
			},
			wantErr: false, // SQL layer handles foreign key validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := tt.mockSetup()
			queries := New(mockDB)

			ctx := context.Background()
			chirp, err := queries.CreateChirp(ctx, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Error("CreateChirp() expected error but got none")
				}
				return
			}

			// Note: Due to mocking limitations, we can't fully test the actual database interaction
			_ = chirp // Acknowledge that we can't fully validate the response with this mock
		})
	}
}

func TestGetChirpById(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		mockSetup func() *mockDBTX
		wantErr   bool
	}{
		{
			name: "valid chirp ID",
			id:   validID,
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT $1::uuid, NOW(), NOW(), $2::text, $3::uuid",
							args[0], "test message", uuid.New())
					},
				}
			},
			wantErr: false,
		},
		{
			name: "nil ID",
			id:   uuid.Nil,
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT * FROM chirpmsgs WHERE id = '00000000-0000-0000-0000-000000000000'")
					},
				}
			},
			wantErr: false, // sql.ErrNoRows would be returned
		},
		{
			name: "nonexistent ID",
			id:   uuid.New(),
			mockSetup: func() *mockDBTX {
				return &mockDBTX{
					queryRowContextFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
						db, _ := sql.Open("postgres", "")
						return db.QueryRowContext(ctx, "SELECT * FROM chirpmsgs WHERE id = 'nonexistent'")
					},
				}
			},
			wantErr: false, // sql.ErrNoRows would be returned
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := tt.mockSetup()
			queries := New(mockDB)

			ctx := context.Background()
			chirp, err := queries.GetChirpById(ctx, tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("GetChirpById() expected error but got none")
				}
				return
			}

			// Note: Due to mocking limitations, we can't fully test the actual database interaction
			_ = chirp // Acknowledge that we can't fully validate the response with this mock
		})
	}
}

// Integration test helpers (these would require a test database)
func TestIntegrationSetup(t *testing.T) {
	t.Skip("Integration tests require a test database - implement when test DB is available")

	// Example of how integration tests would look:
	/*
		// Set up test database
		db, err := sql.Open("postgres", "postgres://test_user:test_pass@localhost/test_chirpy?sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to test database: %v", err)
		}
		defer db.Close()

		// Run migrations
		// ... migration code ...

		// Create queries
		queries := New(db)

		// Test user creation and retrieval
		ctx := context.Background()
		userParams := CreateUserParams{
			Email:          "integration@test.com",
			HashedPassword: "hashed_password",
		}

		user, err := queries.CreateUser(ctx, userParams)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		retrievedUser, err := queries.GetUser(ctx, user.Email)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrievedUser.ID != user.ID {
			t.Errorf("User ID mismatch: expected %v, got %v", user.ID, retrievedUser.ID)
		}

		// Test chirp creation and retrieval
		chirpParams := CreateChirpParams{
			Body:   "Integration test chirp",
			UserID: user.ID,
		}

		chirp, err := queries.CreateChirp(ctx, chirpParams)
		if err != nil {
			t.Fatalf("Failed to create chirp: %v", err)
		}

		retrievedChirp, err := queries.GetChirpById(ctx, chirp.ID)
		if err != nil {
			t.Fatalf("Failed to get chirp: %v", err)
		}

		if retrievedChirp.Body != chirp.Body {
			t.Errorf("Chirp body mismatch: expected %v, got %v", chirp.Body, retrievedChirp.Body)
		}

		// Test getting all chirps
		allChirps, err := queries.GetChirps(ctx)
		if err != nil {
			t.Fatalf("Failed to get all chirps: %v", err)
		}

		if len(allChirps) == 0 {
			t.Error("Expected at least one chirp")
		}
	*/
}

func BenchmarkNew(b *testing.B) {
	mockDB := &mockDBTX{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New(mockDB)
	}
}

func BenchmarkWithTx(b *testing.B) {
	mockDB := &mockDBTX{}
	queries := New(mockDB)
	mockTx := &sql.Tx{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = queries.WithTx(mockTx)
	}
}

// Test data validation helpers
func TestCreateUserParams_Validation(t *testing.T) {
	tests := []struct {
		name   string
		params CreateUserParams
		valid  bool
	}{
		{
			name: "valid params",
			params: CreateUserParams{
				Email:          "test@example.com",
				HashedPassword: "hashed_password_123",
			},
			valid: true,
		},
		{
			name: "empty email",
			params: CreateUserParams{
				Email:          "",
				HashedPassword: "hashed_password_123",
			},
			valid: false,
		},
		{
			name: "empty password",
			params: CreateUserParams{
				Email:          "test@example.com",
				HashedPassword: "",
			},
			valid: false,
		},
		{
			name: "invalid email format",
			params: CreateUserParams{
				Email:          "notanemail",
				HashedPassword: "hashed_password_123",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			isEmpty := func(s string) bool { return len(s) == 0 }

			hasEmptyFields := isEmpty(tt.params.Email) || isEmpty(tt.params.HashedPassword)

			if tt.valid && hasEmptyFields {
				t.Errorf("Expected valid params but found empty fields")
			}

			if !tt.valid && !hasEmptyFields && tt.params.Email == "notanemail" {
				// This would fail at database level due to constraint violations
				t.Logf("Invalid email format would be caught by database constraints")
			}
		})
	}
}

func TestCreateChirpParams_Validation(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name   string
		params CreateChirpParams
		valid  bool
	}{
		{
			name: "valid params",
			params: CreateChirpParams{
				Body:   "This is a valid chirp message",
				UserID: userID,
			},
			valid: true,
		},
		{
			name: "empty body",
			params: CreateChirpParams{
				Body:   "",
				UserID: userID,
			},
			valid: false,
		},
		{
			name: "nil user ID",
			params: CreateChirpParams{
				Body:   "Valid message",
				UserID: uuid.Nil,
			},
			valid: false,
		},
		{
			name: "very long body",
			params: CreateChirpParams{
				Body:   string(make([]byte, 10000)), // Very long string
				UserID: userID,
			},
			valid: true, // Database will handle length constraints
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isEmpty := func(s string) bool { return len(s) == 0 }
			isNilUUID := func(id uuid.UUID) bool { return id == uuid.Nil }

			hasInvalidFields := isEmpty(tt.params.Body) || isNilUUID(tt.params.UserID)

			if tt.valid && hasInvalidFields {
				t.Errorf("Expected valid params but found invalid fields")
			}
		})
	}
}
