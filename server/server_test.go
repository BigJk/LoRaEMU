package server

import (
	"bytes"
	"encoding/json"
	"github.com/BigJk/loraemu/emu"
	"github.com/BigJk/loraemu/lora"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	testEmu := emu.New(800, 10, 1, 10, lora.PacketConfigDefault)
	s := New(testEmu)

	testNodeOk := emu.Node{
		ID:     "Node1",
		Online: true,
		X:      2,
		Y:      4,
		Z:      9,
		TXGain: 22,
		RXSens: -233,
	}

	testNodeInvalid := emu.Node{
		ID:     "",
		Online: true,
		X:      2,
		Y:      4,
		Z:      9,
		TXGain: 22,
		RXSens: -233,
	}

	t.Run("CreateNode", func(t *testing.T) {
		testEmu.Clear()

		nodeJson, _ := json.Marshal(testNodeOk)

		req := httptest.NewRequest(http.MethodPost, "/api/node", bytes.NewBuffer(nodeJson))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.NewContext(req, rec)

		if assert.NoError(t, s.routePostNode(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.EqualValues(t, testNodeOk, testEmu.GetNode(testNodeOk.ID))
		}
	})

	t.Run("CreateNodeInvalid", func(t *testing.T) {
		testEmu.Clear()

		nodeJson, _ := json.Marshal(testNodeInvalid)

		req := httptest.NewRequest(http.MethodPost, "/api/node", bytes.NewBuffer(nodeJson))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.NewContext(req, rec)

		if assert.NoError(t, s.routePostNode(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("PatchNode", func(t *testing.T) {
		testEmu.Clear()

		if !assert.NoError(t, testEmu.AddNode(testNodeOk)) || !assert.Len(t, testEmu.Nodes(), 1) {
			return
		}

		updatedNode := testNodeOk
		updatedNode.X = 10
		updatedNode.Y = 10

		nodeJson, _ := json.Marshal(updatedNode)

		req := httptest.NewRequest(http.MethodPut, "/api/node", bytes.NewBuffer(nodeJson))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.NewContext(req, rec)

		if assert.NoError(t, s.routePutNode(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.EqualValues(t, updatedNode, testEmu.GetNode(updatedNode.ID))
		}
	})

	t.Run("GetNode", func(t *testing.T) {
		testEmu.Clear()

		if !assert.NoError(t, testEmu.AddNode(testNodeOk)) || !assert.Len(t, testEmu.Nodes(), 1) {
			return
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := s.NewContext(req, rec)
		c.SetPath("/api/node/:id")
		c.SetParamNames("id")
		c.SetParamValues(testNodeOk.ID)

		if assert.NoError(t, s.routeGetNode(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)

			var fetched emu.Node
			if assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fetched)) {
				assert.EqualValues(t, testNodeOk, fetched)
			}
		}
	})

	t.Run("GetNodeLatLng", func(t *testing.T) {
		testEmu.Clear()

		if !assert.NoError(t, testEmu.AddNode(testNodeOk)) || !assert.Len(t, testEmu.Nodes(), 1) {
			return
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := s.NewContext(req, rec)
		c.SetPath("/api/node/:id")
		c.SetParamNames("id")
		c.SetParamValues(testNodeOk.ID)

		if assert.NoError(t, s.routeGetNodeLatLng(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)

			lat, lng := testNodeOk.LatLng()

			var fetched []float64
			if assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fetched)) {
				assert.Equal(t, []float64{lat, lng}, fetched)
			}
		}
	})

	t.Run("DeleteNode", func(t *testing.T) {
		testEmu.Clear()

		if !assert.NoError(t, testEmu.AddNode(testNodeOk)) || !assert.Len(t, testEmu.Nodes(), 1) {
			return
		}

		req := httptest.NewRequest(http.MethodDelete, "/", nil)
		rec := httptest.NewRecorder()
		c := s.NewContext(req, rec)
		c.SetPath("/api/node/:id")
		c.SetParamNames("id")
		c.SetParamValues(testNodeOk.ID)

		if assert.NoError(t, s.routeDeleteNode(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Len(t, testEmu.Nodes(), 0)
		}
	})
}
