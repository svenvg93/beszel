//go:build testing

package hub

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSpeedtestID(t *testing.T) {
	auto := generateSpeedtestID("sys1", 0)
	assert.Equal(t, auto, generateSpeedtestID("sys1", 0), "IDs must be stable")
	assert.NotEqual(t, auto, generateSpeedtestID("sys2", 0))
	assert.NotEqual(t, auto, generateSpeedtestID("sys1", 42))
	assert.Regexp(t, "^[a-z0-9]{1,10}$", auto)
}

func TestSpeedtestServerChangeReplacesRecord(t *testing.T) {
	hub, testApp, err := createTestHub(t)
	require.NoError(t, err)
	defer cleanupTestHub(hub, testApp)
	bindSpeedtestsEvents(hub)

	user, err := createTestUser(hub)
	require.NoError(t, err)
	system, err := createTestRecord(hub, "systems", map[string]any{
		"name": "Paused", "host": "localhost", "port": "45876",
		"status": "paused", "users": []string{user.Id},
	})
	require.NoError(t, err)
	record, err := createTestRecord(hub, "speedtests", map[string]any{
		"system": system.Id, "interval": 60, "enabled": true,
	})
	require.NoError(t, err)
	assert.Equal(t, generateSpeedtestID(system.Id, 0), record.Id)

	status, body := speedtestAPIRequest(t, hub, user, http.MethodPatch, "/api/collections/speedtests/records/"+record.Id, map[string]any{"server_id": 42})
	assert.Equal(t, http.StatusOK, status, body)

	records, err := hub.FindAllRecords("speedtests")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, generateSpeedtestID(system.Id, 42), records[0].Id)
	assert.Equal(t, 42, records[0].GetInt("server_id"))
	assert.Equal(t, 60, records[0].GetInt("interval"))
	assert.True(t, records[0].GetBool("enabled"))
}

func speedtestAPIRequest(t *testing.T, hub *Hub, user *core.Record, method, url string, body any) (int, string) {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	token, err := user.NewAuthToken()
	require.NoError(t, err)
	router, err := apis.NewRouter(hub)
	require.NoError(t, err)
	handler, err := router.BuildMux()
	require.NoError(t, err)
	request := httptest.NewRequest(method, url, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code, response.Body.String()
}
