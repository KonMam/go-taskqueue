package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go-taskqueue/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMain(m *testing.M) {
	// Setup and teardown will be handled for each test function
	// to ensure a clean database for each run.
	os.Exit(m.Run())
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "taskqueue",
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "taskqueue",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(2 * time.Minute),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start postgres container")

	mappedPort, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get mapped port")

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "Failed to get container host")

	connStr := fmt.Sprintf("postgres://taskqueue:password@%s:%s/taskqueue", host, mappedPort.Port())

	dbPool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err, "Failed to create db pool")

	err = dbPool.Ping(ctx)
	require.NoError(t, err, "Failed to ping database")

	schema, err := os.ReadFile("../db/schema.sql")
	require.NoError(t, err, "Failed to read schema.sql")
	_, err = dbPool.Exec(ctx, string(schema))
	require.NoError(t, err, "Failed to apply schema")

	// Teardown function to be called at the end of the test
	t.Cleanup(func() {
		dbPool.Close()
		err := postgresContainer.Terminate(ctx)
		if err != nil {
			t.Fatalf("Failed to terminate postgres container: %s", err)
		}
	})

	return dbPool
}

func TestGetTasks(t *testing.T) {
	dbPool := setupTestDB(t)
	appServer := NewServer(":8080", dbPool)

	server := httptest.NewServer(appServer.Handler)
	defer server.Close()

	ctx := context.Background()

	tasksToInsert := []model.Task{
		{Type: "image.resize", Payload: json.RawMessage(`{"source": "/path/to/img.jpg"}`), Status: "queued"},
		{Type: "email.send", Payload: json.RawMessage(`{"to": "user@example.com"}`), Status: "processing"},
		{Type: "report.generate", Payload: json.RawMessage(`{"type": "monthly"}`), Status: "completed"},
		{Type: "report.generate", Payload: json.RawMessage(`{"type": "weekly"}`), Status: "queued"},
		{Type: "email.send", Payload: json.RawMessage(`{"to": "user@example.com"}`), Status: "failed"},
	}

	for _, task := range tasksToInsert {
		_, err := dbPool.Exec(ctx,
			"INSERT INTO tasks (type, payload, status) VALUES ($1, $2, $3)",
			task.Type, task.Payload, task.Status,
		)
		require.NoError(t, err, "Failed to insert test data")
	}

	testCases := []struct {
		name           string
		url            string
		expectedStatus int
		expectedCount  int
		expectedTypes  []string
	}{
		{
			name:           "Get all tasks",
			url:            fmt.Sprintf("%s/tasks", server.URL),
			expectedStatus: http.StatusOK,
			expectedCount:  5,
			expectedTypes:  []string{"image.resize", "email.send", "report.generate", "report.generate", "email.send"},
		},
		{
			name:           "Get tasks with status 'queued'",
			url:            fmt.Sprintf("%s/tasks?status=queued", server.URL),
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectedTypes:  []string{"image.resize", "report.generate"},
		},
		{
			name:           "Get tasks with status 'processing'",
			url:            fmt.Sprintf("%s/tasks?status=processing", server.URL),
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedTypes:  []string{"email.send"},
		},
		{
			name:           "Get tasks with status 'completed'",
			url:            fmt.Sprintf("%s/tasks?status=completed", server.URL),
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedTypes:  []string{"report.generate"},
		},
		{
			name:           "Get tasks with status 'failed'",
			url:            fmt.Sprintf("%s/tasks?status=failed", server.URL),
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedTypes:  []string{"email.send"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.url)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
				defer resp.Body.Close()
			}()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tc.expectedCount > 0 {
				var tasks []model.Task
				err = json.Unmarshal(body, &tasks)
				require.NoError(t, err, "Failed to decode response body")

				assert.Len(t, tasks, tc.expectedCount)

				var receivedTypes []string
				for _, task := range tasks {
					receivedTypes = append(receivedTypes, task.Type)
				}
				assert.ElementsMatch(t, tc.expectedTypes, receivedTypes)
			} else {
				bodyStr := string(body)
				assert.True(t, bodyStr == "[]\n" || bodyStr == "null\n", "Expected empty array or null, got %s", bodyStr)
			}
		})
	}
}

func TestGetTask(t *testing.T) {
	dbPool := setupTestDB(t)
	appServer := NewServer(":8080", dbPool)
	server := httptest.NewServer(appServer.Handler)
	defer server.Close()

	ctx := context.Background()

	var insertedID int
	taskToInsert := model.Task{Type: "image.resize", Payload: json.RawMessage(`{"source": "/path/to/img.jpg"}`), Status: "pending"}
	err := dbPool.QueryRow(ctx,
		"INSERT INTO tasks (type, payload, status) VALUES ($1, $2, $3) RETURNING id",
		taskToInsert.Type, taskToInsert.Payload, taskToInsert.Status,
	).Scan(&insertedID)
	require.NoError(t, err, "Failed to insert test data and get ID")

	testCases := []struct {
		name           string
		url            string
		expectedStatus int
		expectedID     int
		expectError    bool
	}{
		{
			name:           "Get existing task by ID",
			url:            fmt.Sprintf("%s/tasks/%d", server.URL, insertedID),
			expectedStatus: http.StatusOK,
			expectedID:     insertedID,
			expectError:    false,
		},
		{
			name:           "Get non-existent task by ID",
			url:            fmt.Sprintf("%s/tasks/99999", server.URL), // An ID that doesn't exist
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "Get task with invalid ID",
			url:            fmt.Sprintf("%s/tasks/abc", server.URL), // Invalid ID format
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.url)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			if !tc.expectError {
				var task model.Task
				err = json.NewDecoder(resp.Body).Decode(&task)
				require.NoError(t, err, "Failed to decode response")
				assert.Equal(t, tc.expectedID, task.ID)
				assert.Equal(t, "image.resize", task.Type)
			}
		})
	}
}
