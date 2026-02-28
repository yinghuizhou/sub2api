package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// handleHealth returns agent health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"uptime":  time.Now().Format(time.RFC3339),
		"tunnels": len(s.tunnelMgr.List()),
	})
}

// handleListTunnels returns all tunnels.
func (s *Server) handleListTunnels(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.tunnelMgr.List())
}

// handleCreateTunnel creates a new tunnel from JSON body.
func (s *Server) handleCreateTunnel(w http.ResponseWriter, r *http.Request) {
	var req createTunnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" || req.ConfigName == "" {
		writeError(w, http.StatusBadRequest, "name and config_name are required")
		return
	}
	if req.SocksPort == 0 {
		req.SocksPort = s.tunnelMgr.NextAvailablePort()
	}

	info, err := s.tunnelMgr.Create(req.Name, req.ConfigName, req.Region, req.SocksPort, req.ProxyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, info)
}

// handleRemoveTunnel deletes a tunnel by name.
func (s *Server) handleRemoveTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.tunnelMgr.Remove(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// handleStartTunnel starts a stopped tunnel.
func (s *Server) handleStartTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.tunnelMgr.Start(name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.tunnelMgr.Get(name))
}

// handleStopTunnel stops a running tunnel.
func (s *Server) handleStopTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.tunnelMgr.Stop(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.tunnelMgr.Get(name))
}

// handleRestartTunnel restarts a tunnel.
func (s *Server) handleRestartTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.tunnelMgr.Restart(name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.tunnelMgr.Get(name))
}

// handleTunnelStatus returns detailed status for a single tunnel.
func (s *Server) handleTunnelStatus(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	info := s.tunnelMgr.Get(name)
	if info == nil {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

// handleUploadConfigs handles multipart upload of .ovpn files.
func (s *Server) handleUploadConfigs(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "parse multipart: "+err.Error())
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "no files provided (field name: files)")
		return
	}

	var saved []string
	for _, fh := range files {
		if !strings.HasSuffix(fh.Filename, ".ovpn") {
			continue
		}
		f, err := fh.Open()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "open uploaded file: "+err.Error())
			return
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "read uploaded file: "+err.Error())
			return
		}
		if err := s.store.Save(fh.Filename, data); err != nil {
			writeError(w, http.StatusInternalServerError, "save config: "+err.Error())
			return
		}
		saved = append(saved, fh.Filename)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"uploaded": saved,
		"count":    len(saved),
	})
}

// handleListConfigs returns all .ovpn configs.
func (s *Server) handleListConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := s.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

// handleDeleteConfig deletes an .ovpn config file.
func (s *Server) handleDeleteConfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.store.Delete(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// handleNextPort returns the next available SOCKS port.
func (s *Server) handleNextPort(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]int{
		"port": s.tunnelMgr.NextAvailablePort(),
	})
}
