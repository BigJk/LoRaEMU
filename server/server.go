package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/BigJk/loraemu/emu"
	"image"
	"image/png"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/olahol/melody"
	echopprof "github.com/sevenNt/echo-pprof"
)

type NodeStat struct {
	Received  int `json:"received"`
	Collision int `json:"collision"`
	Sending   int `json:"sending"`
}

// Server represents the LoRa emu webserver that hosts the frontend and REST API.
type Server struct {
	sync.RWMutex

	*echo.Echo

	debug           bool
	logger          logr.Logger
	emu             *emu.Emulator
	mobility        *emu.Mobility
	websocket       *melody.Melody
	emuSessions     map[string]*melody.Session
	backgroundImage image.Image
	originX         float64
	originY         float64
	stats           map[string]NodeStat
}

// New creates a new Server instance that is bound to an emulator instance.
func New(emulator *emu.Emulator) *Server {
	return &Server{
		Echo:        echo.New(),
		logger:      logr.Discard(),
		websocket:   melody.New(),
		emu:         emulator,
		emuSessions: map[string]*melody.Session{},
		stats:       map[string]NodeStat{},
	}
}

// SetDebug sets the debug state of the server. If debug is enabled the server expects the
// vite dev server of the frontend to be running.
func (s *Server) SetDebug(state bool) {
	s.debug = state
}

// SetMobility sets a target mobility instance. This will enable the server to pause and resume
// the mobility.
func (s *Server) SetMobility(mobility *emu.Mobility) {
	s.Lock()
	defer s.Unlock()

	s.mobility = mobility
}

// SetLogger sets the logger of the server. If no logger is present no logs will be printed.
func (s *Server) SetLogger(logger logr.Logger) {
	s.Lock()
	defer s.Unlock()

	s.logger = logger
}

// SetBackgroundImage sets a background image that should be displayed in the emu.
func (s *Server) SetBackgroundImage(image image.Image) {
	s.Lock()
	defer s.Unlock()

	s.backgroundImage = image
}

func (s *Server) SetOrigin(x float64, y float64) {
	s.Lock()
	defer s.Unlock()

	s.originX = x
	s.originY = y
}

func (s *Server) ConnectedNodes() int {
	s.RLock()
	defer s.RUnlock()

	return len(s.emuSessions)
}

func (s *Server) handleDisconnect(session *melody.Session) {
	id := session.MustGet("id").(string)

	if session.MustGet("isFrontend").(bool) {
	} else {
		s.logger.Info("node disconnected", "id", id)

		s.Lock()
		delete(s.emuSessions, id)
		s.Unlock()
	}
}

func (s *Server) handleConnect(session *melody.Session) {
	id := session.MustGet("id").(string)

	if session.MustGet("isFrontend").(bool) {
		if configBytes, err := json.Marshal(map[string]interface{}{
			"event":     "Config",
			"gamma":     s.emu.GetGamma(),
			"refDist":   s.emu.GetRefDist(),
			"freq":      s.emu.GetFreq(),
			"kmRange":   s.emu.GetKMRange(),
			"startTime": s.emu.GetStartTime(),
			"origin": map[string]interface{}{
				"x": s.originX,
				"y": s.originY,
			},
			"curNodeStats": s.stats,
		}); err == nil {
			_ = session.Write(configBytes)
		}

		if nodeBytes, err := json.Marshal(map[string]interface{}{
			"event": "Nodes",
			"nodes": s.emu.Nodes(),
		}); err == nil {
			_ = session.Write(nodeBytes)
		}
	} else {
		s.Lock()

		// check if the node already is connected to, if so close the new request
		if _, ok := s.emuSessions[id]; ok {
			s.logger.Error(nil, "node denied connection", "id", id)
			_ = session.Close()
			return
		}

		s.emuSessions[session.MustGet("id").(string)] = session

		s.Unlock()

		s.logger.Info("node connected", "id", id)
	}
}

func (s *Server) handleMessageBinary(session *melody.Session, bytes []byte) {
	id := session.MustGet("id").(string)
	if err := s.emu.SendMessage(id, bytes); err != nil {
		s.logger.Error(err, "node denied connection", "id", id)
	}
}

func (s *Server) onEvent(event emu.Event, node emu.Node, data any) {
	bytes, err := json.Marshal(&map[string]interface{}{
		"event": string(event),
		"node":  node,
		"data":  data,
	})

	if err != nil {
		return
	}

	s.logger.Info("event emitted for node", "event", string(event), "id", node.ID)

	_ = s.websocket.BroadcastFilter(bytes, func(session *melody.Session) bool {
		return session.MustGet("isFrontend").(bool)
	})

	switch event {
	case emu.EventCollision:
		go func() {
			s.Lock()
			defer s.Unlock()

			val, ok := s.stats[node.ID]
			if ok {
				val.Collision += 1
				s.stats[node.ID] = val
			} else {
				s.stats[node.ID] = NodeStat{
					Received:  0,
					Collision: 1,
					Sending:   0,
				}
			}
		}()
	case emu.EventReceived:
		go func() {
			s.Lock()
			defer s.Unlock()

			val, ok := s.stats[node.ID]
			if ok {
				val.Received += 1
				s.stats[node.ID] = val
			} else {
				s.stats[node.ID] = NodeStat{
					Received:  1,
					Collision: 0,
					Sending:   0,
				}
			}
		}()
	case emu.EventSending:
		go func() {
			s.Lock()
			defer s.Unlock()

			val, ok := s.stats[node.ID]
			if ok {
				val.Sending += 1
				s.stats[node.ID] = val
			} else {
				s.stats[node.ID] = NodeStat{
					Received:  0,
					Collision: 0,
					Sending:   1,
				}
			}
		}()
	}
}

func (s *Server) onReceived(node emu.Node, packet emu.RxPacket) {
	s.logger.Info("node got message", "id", node.ID, "len", len(packet.Data), "rssi", packet.RSSI)

	bytes, err := json.Marshal(packet)
	if err != nil {
		s.logger.Error(err, "error while marshaling RxPacket", "id", node.ID)
		return
	}

	_ = s.websocket.BroadcastFilter(bytes, func(session *melody.Session) bool {
		return !session.MustGet("isFrontend").(bool) && session.MustGet("id").(string) == node.ID
	})
}

func (s *Server) routeNodeWebsocketUpgrade(c echo.Context) error {
	id := c.Param("id")

	s.RLock()

	if !s.emu.HasNode(id) {
		return c.String(http.StatusBadRequest, "node doesn't exist")
	}

	if _, ok := s.emuSessions[id]; ok {
		return c.String(http.StatusBadRequest, "already connected")
	}

	s.RUnlock()

	if err := s.websocket.HandleRequestWithKeys(c.Response().Writer, c.Request(), map[string]interface{}{
		"id":         id,
		"isFrontend": false,
	}); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return nil
}

func (s *Server) routeFrontendWebsocketUpgrade(c echo.Context) error {
	if err := s.websocket.HandleRequestWithKeys(c.Response().Writer, c.Request(), map[string]interface{}{
		"id":         fmt.Sprintf("%d", rand.Int()),
		"isFrontend": true,
	}); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return nil
}

func (s *Server) routeGetNodeLatLng(c echo.Context) error {
	id := c.Param("id")

	node := s.emu.GetNode(id)
	if node.ID == "" {
		return c.JSON(http.StatusNotFound, "not found")
	}

	lat, lng := node.LatLng()

	return c.JSON(http.StatusOK, []float64{lat, lng})
}

func (s *Server) routeGetNode(c echo.Context) error {
	id := c.Param("id")

	node := s.emu.GetNode(id)
	if node.ID == "" {
		return c.JSON(http.StatusNotFound, "not found")
	}

	return c.JSON(http.StatusOK, node)
}

func (s *Server) routeDeleteNode(c echo.Context) error {
	id := c.Param("id")

	err := s.emu.RemoveNode(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

func (s *Server) routePutNode(c echo.Context) error {
	var updated emu.Node

	if err := c.Bind(&updated); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := s.emu.UpdateNode(updated.ID, func(node *emu.Node) error {
		*node = updated
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

func (s *Server) routePutNodeMeta(c echo.Context) error {
	updated := map[string]interface{}{}

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	if err := json.Unmarshal(body, &updated); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err = s.emu.UpdateNode(c.Param("id"), func(node *emu.Node) error {
		node.Meta = updated
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

func (s *Server) routePostNode(c echo.Context) error {
	var newNode emu.Node

	if err := c.Bind(&newNode); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	err := s.emu.AddNode(newNode)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

func (s *Server) routeGetNodes(c echo.Context) error {
	return c.JSON(http.StatusOK, s.emu.Nodes())
}

func (s *Server) routeGetNodeIDs(c echo.Context) error {
	return c.JSON(http.StatusOK, s.emu.NodeIDs())
}

func (s *Server) routeGetEmuPause(c echo.Context) error {
	if s.mobility == nil {
		return c.JSON(http.StatusNotFound, "no mobility file active")
	}

	return c.JSON(http.StatusOK, s.mobility.GetPause())
}

func (s *Server) routePostEmuPause(c echo.Context) error {
	val := struct {
		State bool `json:"state"`
	}{}

	if err := c.Bind(&val); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	if s.mobility == nil {
		return c.JSON(http.StatusNotFound, "no mobility file active")
	}

	s.mobility.SetPause(val.State)

	return c.NoContent(http.StatusOK)
}

func (s *Server) routeGetBackgroundImage(c echo.Context) error {
	s.RLock()
	defer s.RUnlock()

	if s.backgroundImage == nil {
		return c.NoContent(http.StatusNotFound)
	}

	buf := &bytes.Buffer{}
	if err := png.Encode(buf, s.backgroundImage); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	return c.Stream(http.StatusOK, "image/png", buf)
}

// Start the LoRa emu server that hosts the frontend of the emulator.
func (s *Server) Start(bind string) error {
	s.HideBanner = true

	// sets event for the LoRa emu
	s.emu.SetOnEvent(s.onEvent)
	s.emu.SetOnReceived(s.onReceived)

	// set handlers for the websocket
	s.websocket.Upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	s.websocket.HandleDisconnect(s.handleDisconnect)
	s.websocket.HandleConnect(s.handleConnect)
	s.websocket.HandleMessageBinary(s.handleMessageBinary)

	// set websocket upgrader
	s.GET("/api/emu/:id", s.routeNodeWebsocketUpgrade).Name = "Node Websocket"
	s.GET("/api/ws", s.routeFrontendWebsocketUpgrade).Name = "Frontend Websocket"

	// other api routes
	s.GET("/api/nodes", s.routeGetNodes).Name = "Get Nodes"
	s.GET("/api/node_ids", s.routeGetNodeIDs).Name = "Get Node IDs"
	s.GET("/api/node/:id/latlng", s.routeGetNodeLatLng).Name = "Get Node LatLng"
	s.GET("/api/node/:id", s.routeGetNode).Name = "Get Node"
	s.PUT("/api/node/update", s.routePutNode).Name = "Update Node"
	s.PUT("/api/node/:id/meta", s.routePutNodeMeta).Name = "Update Node Meta Info"
	s.POST("/api/node/create", s.routePostNode).Name = "Create Node"
	s.DELETE("/api/node/:id", s.routeDeleteNode).Name = "Delete Node"
	s.GET("/api/emu/pause", s.routeGetEmuPause).Name = "Get Pause Emu"
	s.POST("/api/emu/pause", s.routePostEmuPause).Name = "Pause Emu"
	s.GET("/api/background", s.routeGetBackgroundImage).Name = "Get Background Image"

	// api route that shows all available routes
	s.GET("/api/routes", func(c echo.Context) error {
		return c.JSONPretty(http.StatusOK, s.Routes(), "\t")
	}).Name = "Available Routes"

	// if debug is active proxy non /api or /debug routes to vite and register pprof
	if s.debug {
		echopprof.Wrap(s.Echo)

		viteUrl, err := url.Parse("http://127.0.0.1:9272")
		if err != nil {
			return err
		}

		s.Use(middleware.ProxyWithConfig(middleware.ProxyConfig{Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Request().URL.Path, "/api") || strings.HasPrefix(c.Request().URL.Path, "/debug")
		}, Balancer: middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{{URL: viteUrl}})}))
	} else {
		s.Static("/", "./frontend/dist")

		// If a frontend folder is also in the same directory as the executable regardless of the
		// working directory we expose that too to reduce problems that could arise from running
		// the emu server from a different location.
		if exec, err := os.Executable(); err == nil {
			if _, err := os.Stat(filepath.Join(filepath.Dir(exec), "/frontend/dist")); err == nil {
				s.Static("/", filepath.Join(filepath.Dir(exec), "/frontend/dist"))
			}
		}
	}

	return s.Echo.Start(bind)
}

// Stop the server.
func (s *Server) Stop() error {
	return s.Echo.Close()
}
