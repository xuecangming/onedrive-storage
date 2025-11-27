package templates

import (
	"embed"
	"html/template"
	"io"
	"sync"
)

//go:embed *.html
var templateFS embed.FS

// Manager handles HTML template loading and rendering
type Manager struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
}

// Global template manager instance
var defaultManager *Manager
var once sync.Once

// GetManager returns the global template manager instance
func GetManager() *Manager {
	once.Do(func() {
		defaultManager = &Manager{
			templates: make(map[string]*template.Template),
		}
		defaultManager.loadTemplates()
	})
	return defaultManager
}

// loadTemplates loads all templates from embedded filesystem
func (m *Manager) loadTemplates() {
	templateNames := []string{
		"success.html",
		"setup_guide.html",
		"account_form.html",
		"account_list.html",
	}

	for _, name := range templateNames {
		tmpl, err := template.ParseFS(templateFS, name)
		if err != nil {
			panic("failed to parse template " + name + ": " + err.Error())
		}
		m.templates[name] = tmpl
	}
}

// Render executes a template by name with the given data
func (m *Manager) Render(w io.Writer, name string, data interface{}) error {
	m.mu.RLock()
	tmpl, ok := m.templates[name]
	m.mu.RUnlock()

	if !ok {
		return &TemplateNotFoundError{Name: name}
	}

	return tmpl.Execute(w, data)
}

// TemplateNotFoundError is returned when a template doesn't exist
type TemplateNotFoundError struct {
	Name string
}

func (e *TemplateNotFoundError) Error() string {
	return "template not found: " + e.Name
}

// Template data structures

// SuccessData holds data for the success page template
type SuccessData struct {
	Name  string
	Email string
}

// SetupGuideData holds data for the setup guide template
type SetupGuideData struct {
	RedirectURI string
}

// AccountListData holds data for the account list template
type AccountListData struct {
	Accounts []AccountViewData
}

// AccountViewData holds data for a single account in the list
type AccountViewData struct {
	ID          string
	Name        string
	Email       string
	SpaceInfo   string
	UsedPercent float64
	StatusClass string
	StatusText  string
}
