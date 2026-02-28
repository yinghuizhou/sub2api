package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
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
		port, err := s.tunnelMgr.NextAvailablePort()
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		req.SocksPort = port
	}

	var info *TunnelInfo
	var err error

	if req.TunnelType == string(TunnelTypeWireGuard) {
		info, err = s.tunnelMgr.CreateWireGuard(req.Name, req.ConfigName, req.Region, req.SocksPort, req.ProxyID)
	} else {
		info, err = s.tunnelMgr.Create(req.Name, req.ConfigName, req.Region, req.SocksPort, req.ProxyID, req.CertID)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, info)
}

// handleRemoveTunnel deletes a tunnel by name.
func (s *Server) handleRemoveTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := validateTunnelName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.tunnelMgr.Remove(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// handleStartTunnel starts a stopped tunnel.
func (s *Server) handleStartTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := validateTunnelName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.tunnelMgr.Start(name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.tunnelMgr.Get(name))
}

// handleStopTunnel stops a running tunnel.
func (s *Server) handleStopTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := validateTunnelName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.tunnelMgr.Stop(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.tunnelMgr.Get(name))
}

// handleRestartTunnel restarts a tunnel.
func (s *Server) handleRestartTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := validateTunnelName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.tunnelMgr.Restart(name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.tunnelMgr.Get(name))
}

// handleTunnelStatus returns detailed status for a single tunnel.
func (s *Server) handleTunnelStatus(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := validateTunnelName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
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
		safeName := filepath.Base(fh.Filename)
		if err := s.store.Save(safeName, data); err != nil {
			writeError(w, http.StatusInternalServerError, "save config: "+err.Error())
			return
		}
		saved = append(saved, safeName)
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
	if err := validateConfigName(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.Delete(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// handleNextPort returns the next available SOCKS port.
func (s *Server) handleNextPort(w http.ResponseWriter, r *http.Request) {
	port, err := s.tunnelMgr.NextAvailablePort()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"port": port})
}

// handleListCerts returns all certificates.
func (s *Server) handleListCerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.certStore.List())
}

// handleImportCert imports a certificate from uploaded .ovpn data.
func (s *Server) handleImportCert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Label    string `json:"label"`
		OvpnData string `json:"ovpn_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Label == "" || req.OvpnData == "" {
		writeError(w, http.StatusBadRequest, "label and ovpn_data are required")
		return
	}
	cert, err := s.certStore.Import(req.Label, []byte(req.OvpnData))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cert)
}

// handleDeleteCert deletes a certificate by ID.
func (s *Server) handleDeleteCert(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	if err := s.certStore.Delete(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// pushConfigRequest is the JSON body for POST /api/configs/push.
type pushConfigRequest struct {
	Configs         []pushConfigItem `json:"configs"`
	ReplaceExisting bool             `json:"replace_existing"`
}

type pushConfigItem struct {
	Filename   string `json:"filename"`
	Content    string `json:"content"`     // base64 encoded
	Region     string `json:"region"`      // optional
	AutoDeploy bool   `json:"auto_deploy"`
}

type pushConfigResult struct {
	Saved    []string `json:"saved"`
	Skipped  []string `json:"skipped"`
	Deployed []string `json:"deployed"`
	Errors   []string `json:"errors"`
}

// handlePushConfigs accepts base64-encoded configs via JSON and optionally restarts tunnels.
func (s *Server) handlePushConfigs(w http.ResponseWriter, r *http.Request) {
	var req pushConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(req.Configs) == 0 {
		writeError(w, http.StatusBadRequest, "configs array is required and must not be empty")
		return
	}

	result := pushConfigResult{
		Saved:    []string{},
		Skipped:  []string{},
		Deployed: []string{},
		Errors:   []string{},
	}

	for _, item := range req.Configs {
		// Validate filename.
		if !strings.HasSuffix(item.Filename, ".ovpn") {
			result.Errors = append(result.Errors, item.Filename+": must end with .ovpn")
			continue
		}

		// Base64 decode content.
		decoded, err := base64.StdEncoding.DecodeString(item.Content)
		if err != nil {
			result.Errors = append(result.Errors, item.Filename+": base64 decode failed: "+err.Error())
			continue
		}

		// Save with metadata tracking.
		safeName := filepath.Base(item.Filename)
		_, isNew, err := s.store.SaveWithMeta(safeName, decoded)
		if err != nil {
			result.Errors = append(result.Errors, safeName+": save failed: "+err.Error())
			continue
		}

		if !isNew {
			result.Skipped = append(result.Skipped, safeName)
			continue
		}

		result.Saved = append(result.Saved, safeName)

		// Auto-deploy: restart tunnels using this config.
		if item.AutoDeploy {
			for _, t := range s.tunnelMgr.List() {
				if t.ConfigName == safeName {
					if err := s.tunnelMgr.Restart(t.Name); err != nil {
						result.Errors = append(result.Errors, t.Name+": restart failed: "+err.Error())
					} else {
						result.Deployed = append(result.Deployed, t.Name)
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, result)
}
