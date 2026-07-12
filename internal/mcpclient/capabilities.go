// MCP server capability introspection for the admin UI.
package mcpclient

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolCapability describes one MCP tool bridged into the registry.
type ToolCapability struct {
	Name        string `json:"name"`         // registry name (mcp_<server>_<tool>)
	MCPName     string `json:"mcp_name"`     // original MCP tool name
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// ResourceInfo is a readable MCP resource.
type ResourceInfo struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mime_type,omitempty"`
}

// ResourceTemplateInfo is an MCP resource template.
type ResourceTemplateInfo struct {
	URITemplate string `json:"uri_template"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mime_type,omitempty"`
}

// ServerCapabilities lists tools, resources, and templates from a live session.
type ServerCapabilities struct {
	Connected bool                   `json:"connected"`
	Tools     []ToolCapability       `json:"tools"`
	Resources []ResourceInfo         `json:"resources"`
	Templates []ResourceTemplateInfo `json:"templates"`
}

// ServerCapabilities returns live MCP capabilities for a connected server.
func (m *Manager) ServerCapabilities(ctx context.Context, serverID string) (*ServerCapabilities, error) {
	m.mu.Lock()
	c, ok := m.sessions[serverID]
	m.mu.Unlock()
	if !ok || c == nil {
		return &ServerCapabilities{Connected: false}, nil
	}

	out := &ServerCapabilities{Connected: true}

	tools, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}
	for _, t := range tools.Tools {
		fullName := ToolName(c.serverName, t.Name)
		out.Tools = append(out.Tools, ToolCapability{
			Name:        fullName,
			MCPName:     t.Name,
			Description: toolDescription(c.serverName, t),
			Enabled:     m.reg.IsEnabled(fullName),
		})
	}

	resources, err := c.session.ListResources(ctx, nil)
	if err == nil {
		for _, r := range resources.Resources {
			out.Resources = append(out.Resources, mapResource(r))
		}
	}

	templates, err := c.session.ListResourceTemplates(ctx, nil)
	if err == nil {
		for _, rt := range templates.ResourceTemplates {
			out.Templates = append(out.Templates, mapResourceTemplate(rt))
		}
	}

	return out, nil
}

func mapResource(r *sdkmcp.Resource) ResourceInfo {
	if r == nil {
		return ResourceInfo{}
	}
	return ResourceInfo{
		URI:         r.URI,
		Name:        r.Name,
		Title:       r.Title,
		Description: r.Description,
		MIMEType:    r.MIMEType,
	}
}

func mapResourceTemplate(rt *sdkmcp.ResourceTemplate) ResourceTemplateInfo {
	if rt == nil {
		return ResourceTemplateInfo{}
	}
	return ResourceTemplateInfo{
		URITemplate: rt.URITemplate,
		Name:        rt.Name,
		Title:       rt.Title,
		Description: rt.Description,
		MIMEType:    rt.MIMEType,
	}
}
