package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleBluetoothStatus(w http.ResponseWriter, r *http.Request) {
	s.btMu.RLock()
	resp := map[string]any{
		"scanning":   s.btScanning,
		"defaultMac": s.btDefaultMac,
		"connected":  nil,
	}
	s.btMu.RUnlock()
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleBluetoothScanStart(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if _, err := s.btCtl.Run(ctx, "scan", "on"); err != nil {
		writeError(w, http.StatusBadGateway, "bluetoothctl_error", err.Error())
		return
	}

	s.btMu.Lock()
	s.btScanning = true
	s.btMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleBluetoothScanStop(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if _, err := s.btCtl.Run(ctx, "scan", "off"); err != nil {
		writeError(w, http.StatusBadGateway, "bluetoothctl_error", err.Error())
		return
	}

	s.btMu.Lock()
	s.btScanning = false
	s.btMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type btDevice struct {
	MAC  string `json:"mac"`
	Name string `json:"name"`
}

func (s *Server) handleBluetoothDevices(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	out, err := s.btCtl.Run(ctx, "devices")
	if err != nil {
		writeError(w, http.StatusBadGateway, "bluetoothctl_error", err.Error())
		return
	}

	var devices []btDevice
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Device ") {
			continue
		}
		rest := strings.TrimPrefix(line, "Device ")
		parts := strings.SplitN(rest, " ", 2)
		if len(parts) != 2 {
			continue
		}
		devices = append(devices, btDevice{
			MAC:  strings.TrimSpace(parts[0]),
			Name: strings.TrimSpace(parts[1]),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"devices": devices,
	})
}

func (s *Server) handleBluetoothConnect(w http.ResponseWriter, r *http.Request) {
	mac := r.PathValue("mac")
	if strings.TrimSpace(mac) == "" {
		writeError(w, http.StatusBadRequest, "bad_mac", "Missing MAC")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if _, err := s.btCtl.Run(ctx, "connect", mac); err != nil {
		writeError(w, http.StatusBadGateway, "bluetoothctl_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleBluetoothDisconnect(w http.ResponseWriter, r *http.Request) {
	mac := r.PathValue("mac")
	if strings.TrimSpace(mac) == "" {
		writeError(w, http.StatusBadRequest, "bad_mac", "Missing MAC")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if _, err := s.btCtl.Run(ctx, "disconnect", mac); err != nil {
		writeError(w, http.StatusBadGateway, "bluetoothctl_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type btDefaultReq struct {
	MAC string `json:"mac"`
}

func (s *Server) handleBluetoothDefaultSet(w http.ResponseWriter, r *http.Request) {
	var req btDefaultReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "Invalid JSON body")
		return
	}
	if strings.TrimSpace(req.MAC) == "" {
		writeError(w, http.StatusBadRequest, "bad_mac", "Missing MAC")
		return
	}
	s.btMu.Lock()
	s.btDefaultMac = req.MAC
	s.btMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleBluetoothDefaultClear(w http.ResponseWriter, r *http.Request) {
	s.btMu.Lock()
	s.btDefaultMac = ""
	s.btMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

