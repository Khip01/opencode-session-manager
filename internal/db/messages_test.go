package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func setupMessagesFixture(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "opencode.db")
	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.Exec(`
		CREATE TABLE message (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			data TEXT NOT NULL
		);
		CREATE TABLE part (
			id TEXT PRIMARY KEY,
			message_id TEXT NOT NULL,
			session_id TEXT NOT NULL,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			data TEXT NOT NULL
		);
	`)
	require.NoError(t, err)
	return dbPath
}

func openForTest(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	handle, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { handle.Close() })
	return handle
}

func insertMessage(t *testing.T, dbPath, id, sessionID, data string, ts int64) {
	t.Helper()
	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.Exec(
		`INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?)`,
		id, sessionID, ts, ts, data,
	)
	require.NoError(t, err)
}

func insertPart(t *testing.T, dbPath, id, msgID, sessionID, data string) {
	t.Helper()
	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.Exec(
		`INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?, ?)`,
		id, msgID, sessionID, 1, 1, data,
	)
	require.NoError(t, err)
}

func TestJsonStringField_Basic(t *testing.T) {
	json := `{"role":"user","agent":"build","text":"hello"}`
	assert.Equal(t, "user", jsonStringField(json, "role"))
	assert.Equal(t, "build", jsonStringField(json, "agent"))
	assert.Equal(t, "hello", jsonStringField(json, "text"))
	assert.Equal(t, "", jsonStringField(json, "missing"))
}

func TestJsonStringField_WithSpaces(t *testing.T) {
	json := `  {"role":   "user"  }`
	assert.Equal(t, "user", jsonStringField(json, "role"))
}

func TestJsonStringField_EscapedChars(t *testing.T) {
	json := `{"text":"line1\nline2","role":"assistant"}`
	assert.Equal(t, `line1\nline2`, jsonStringField(json, "text"))
	assert.Equal(t, "assistant", jsonStringField(json, "role"))
}

func TestJsonStringField_NestedObject(t *testing.T) {
	json := `{"time":{"created":12345},"role":"user"}`
	assert.Equal(t, "user", jsonStringField(json, "role"))
}

func TestJsonStringField_Malformed(t *testing.T) {
	assert.Equal(t, "", jsonStringField(`{"role":`, "role"))
	assert.Equal(t, "", jsonStringField(`not json`, "role"))
	assert.Equal(t, "", jsonStringField(``, "role"))
	assert.Equal(t, "", jsonStringField(`{"role":123}`, "role"))
}

func TestJsonStringField_RealisticPayload(t *testing.T) {
	payload := `{"parentID":"msg_abc","role":"assistant","mode":"build","agent":"build","path":{"cwd":"/home/user/proj","root":"/"},"cost":0,"tokens":{"total":8912,"input":8869,"output":11,"reasoning":32,"cache":{"write":0,"read":0}},"modelID":"big-pickle","providerID":"opencode","time":{"created":1779847759575,"completed":1779847762385},"finish":"stop"}`
	assert.Equal(t, "assistant", jsonStringField(payload, "role"))
	assert.Equal(t, "build", jsonStringField(payload, "mode"))
	assert.Equal(t, "build", jsonStringField(payload, "agent"))
	assert.Equal(t, "big-pickle", jsonStringField(payload, "modelID"))
	assert.Equal(t, "opencode", jsonStringField(payload, "providerID"))
	assert.Equal(t, "stop", jsonStringField(payload, "finish"))
	assert.Equal(t, "msg_abc", jsonStringField(payload, "parentID"))
}

func TestListMessages_Empty(t *testing.T) {
	dbPath := setupMessagesFixture(t)
	handle := openForTest(t, dbPath)

	msgs, err := ListMessages(context.Background(), handle, "ses_x", 10)
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestListMessages_ReturnsRoleAgentModel(t *testing.T) {
	dbPath := setupMessagesFixture(t)
	insertMessage(t, dbPath, "msg_1", "ses_a", `{"role":"user","agent":"build","modelID":"big-pickle","providerID":"opencode"}`, 1000)
	insertMessage(t, dbPath, "msg_2", "ses_a", `{"role":"assistant","agent":"build","modelID":"big-pickle"}`, 2000)
	insertMessage(t, dbPath, "msg_3", "ses_b", `{"role":"user"}`, 3000)

	handle := openForTest(t, dbPath)

	msgs, err := ListMessages(context.Background(), handle, "ses_a", 10)
	require.NoError(t, err)
	require.Len(t, msgs, 2)

	assert.Equal(t, "msg_1", msgs[0].ID)
	assert.Equal(t, "user", msgs[0].Role)
	assert.Equal(t, "build", msgs[0].Agent)
	assert.Equal(t, "big-pickle", msgs[0].ModelID)
	assert.Equal(t, "opencode", msgs[0].ProviderID)

	assert.Equal(t, "msg_2", msgs[1].ID)
	assert.Equal(t, "assistant", msgs[1].Role)
}

func TestListMessages_AttachesParts(t *testing.T) {
	dbPath := setupMessagesFixture(t)
	insertMessage(t, dbPath, "msg_1", "ses_a", `{"role":"user"}`, 1000)
	insertPart(t, dbPath, "p_1", "msg_1", "ses_a", `{"type":"text","text":"hello world"}`)
	insertPart(t, dbPath, "p_2", "msg_1", "ses_a", `{"type":"step-start"}`)

	handle := openForTest(t, dbPath)

	msgs, err := ListMessages(context.Background(), handle, "ses_a", 10)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Len(t, msgs[0].Parts, 2)

	assert.Equal(t, "text", msgs[0].Parts[0].Type)
	assert.Equal(t, "hello world", msgs[0].Parts[0].Text)
	assert.Equal(t, "step-start", msgs[0].Parts[1].Type)
	assert.Equal(t, "", msgs[0].Parts[1].Text)
}

func TestListMessages_RespectsLimit(t *testing.T) {
	dbPath := setupMessagesFixture(t)
	for i := 0; i < 10; i++ {
		ts := int64((i + 1) * 1000)
		insertMessage(t, dbPath,
			fmt.Sprintf("msg_%d", i), "ses_a",
			fmt.Sprintf(`{"role":"user","time":{"created":%d}}`, ts),
			ts)
	}

	handle := openForTest(t, dbPath)

	msgs, err := ListMessages(context.Background(), handle, "ses_a", 3)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)

	msgsAll, err := ListMessages(context.Background(), handle, "ses_a", 0)
	require.NoError(t, err)
	assert.Len(t, msgsAll, 10)
}

func TestListMessages_FiltersBySession(t *testing.T) {
	dbPath := setupMessagesFixture(t)
	insertMessage(t, dbPath, "msg_1", "ses_a", `{"role":"user"}`, 1000)
	insertMessage(t, dbPath, "msg_2", "ses_b", `{"role":"user"}`, 2000)
	insertMessage(t, dbPath, "msg_3", "ses_a", `{"role":"assistant"}`, 3000)

	handle := openForTest(t, dbPath)

	msgsA, err := ListMessages(context.Background(), handle, "ses_a", 0)
	require.NoError(t, err)
	assert.Len(t, msgsA, 2)

	msgsB, err := ListMessages(context.Background(), handle, "ses_b", 0)
	require.NoError(t, err)
	assert.Len(t, msgsB, 1)
	assert.Equal(t, "user", msgsB[0].Role)
}

func TestListMessages_OrderedChronologically(t *testing.T) {
	dbPath := setupMessagesFixture(t)
	insertMessage(t, dbPath, "msg_late", "ses_a", `{"role":"user"}`, 5000)
	insertMessage(t, dbPath, "msg_early", "ses_a", `{"role":"user"}`, 1000)
	insertMessage(t, dbPath, "msg_mid", "ses_a", `{"role":"user"}`, 3000)

	handle := openForTest(t, dbPath)

	msgs, err := ListMessages(context.Background(), handle, "ses_a", 0)
	require.NoError(t, err)
	require.Len(t, msgs, 3)
	assert.Equal(t, "msg_early", msgs[0].ID)
	assert.Equal(t, "msg_mid", msgs[1].ID)
	assert.Equal(t, "msg_late", msgs[2].ID)
}
