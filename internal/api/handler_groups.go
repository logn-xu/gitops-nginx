package api

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleGetGroups(c *gin.Context) {
	var groups []GroupSummary
	for _, g := range s.cfg.NginxServers {
		var hosts []HostSummary
		for _, h := range g.Servers {
			hosts = append(hosts, HostSummary{
				Name:            h.Name,
				Host:            h.Host,
				ConfigDirSuffix: filepath.Base(h.NginxConfigDir),
			})
		}
		groups = append(groups, GroupSummary{
			Name:  g.Group,
			Hosts: hosts,
		})
	}
	c.JSON(http.StatusOK, GroupsResponse{Groups: groups})
}
