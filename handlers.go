package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

var templates *template.Template

func initTemplates() error {
	funcMap := template.FuncMap{
		"urlquery": url.QueryEscape,
		"base":     filepath.Base,
	}

	var err error
	templates, err = template.New("").Funcs(funcMap).ParseGlob("frontend/templates/*.html")
	return err
}

// HTML Route Handlers

func handleHome(w http.ResponseWriter, r *http.Request) {
	projects, err := getAllProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Projects": projects,
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleProjectFiles(w http.ResponseWriter, r *http.Request) {
	// Get everything after /projects/
	fullPath := strings.TrimPrefix(r.URL.Path, "/projects/")

	// Ensure leading slash for absolute paths
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}

	// URL decode the full path (may already be decoded by Go's http server)
	decodedPath, err := url.PathUnescape(fullPath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Find the first registered project that matches the beginning of the path
	projects, err := getAllProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var project string
	var childPath string

	for _, p := range projects {
		if decodedPath == p.Directory || strings.HasPrefix(decodedPath, p.Directory+"/") {
			project = p.Directory
			childPath = strings.TrimPrefix(decodedPath, p.Directory)
			childPath = strings.TrimPrefix(childPath, "/")
			break
		}
	}

	if project == "" {
		http.NotFound(w, r)
		return
	}

	// Build absolute path
	absPath := filepath.Join(project, childPath)

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// If directory, show markdown file listing
	if info.IsDir() {
		// Redirect to add trailing slash if needed (for proper relative URLs)
		if !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
			return
		}
		renderDirectoryListing(w, r, project, childPath)
		return
	}

	// If markdown file, render viewer
	if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
		renderViewer(w, r, project, childPath)
		return
	}

	// Otherwise serve raw file
	prefix := "/projects/" + url.PathEscape(project)
	fs := http.FileServer(http.Dir(project))
	http.StripPrefix(prefix, fs).ServeHTTP(w, r)
}

func renderViewer(w http.ResponseWriter, r *http.Request, projectDir, filePath string) {
	absPath := filepath.Join(projectDir, filePath)

	// Read markdown file
	content, err := os.ReadFile(absPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Render markdown to HTML
	html, err := RenderMarkdownWithLineNumbers(content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get comments for this file
	comments, err := getComments(projectDir, filePath, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"ProjectDir":  projectDir,
		"FilePath":    filePath,
		"HTMLContent": template.HTML(html),
		"Comments":    comments,
	}

	if err := templates.ExecuteTemplate(w, "viewer.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var skipDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"vendor":        true,
	".next":         true,
	".nuxt":         true,
	"dist":          true,
	"build":         true,
	"target":        true,
	".venv":         true,
	"venv":          true,
	"__pycache__":   true,
	".pytest_cache": true,
	".idea":         true,
	".vscode":       true,
	".DS_Store":     true,
	".direnv":       true,
	".mypy_cache":   true,
	".vim":          true,
	".ruff_cache":   true,
}

func shouldSkipDir(name string) bool {
	return skipDirs[name]
}

func hasMarkdownFiles(dirPath string) bool {
	// Use filepath.WalkDir for efficient traversal
	found := false
	_ = filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		// Skip common directories
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			found = true
			return filepath.SkipAll // Stop walking once we find one
		}
		return nil
	})
	return found
}

func renderDirectoryListing(w http.ResponseWriter, r *http.Request, projectDir, childPath string) {
	absPath := filepath.Join(projectDir, childPath)

	// Read directory contents
	entries, err := os.ReadDir(absPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter for directories and markdown files
	type Entry struct {
		Name  string
		IsDir bool
		Path  string
	}

	var filteredEntries []Entry
	for _, entry := range entries {
		if entry.IsDir() {
			// Skip common directories
			if shouldSkipDir(entry.Name()) {
				continue
			}
			// Only include directories that contain markdown files
			dirFullPath := filepath.Join(absPath, entry.Name())
			if hasMarkdownFiles(dirFullPath) {
				filteredEntries = append(filteredEntries, Entry{
					Name:  entry.Name(),
					IsDir: true,
					Path:  filepath.Join(childPath, entry.Name()),
				})
			}
		} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			// Include only markdown files
			filteredEntries = append(filteredEntries, Entry{
				Name:  entry.Name(),
				IsDir: false,
				Path:  filepath.Join(childPath, entry.Name()),
			})
		}
	}

	data := map[string]interface{}{
		"ProjectDir": projectDir,
		"ChildPath":  childPath,
		"Entries":    filteredEntries,
	}

	if err := templates.ExecuteTemplate(w, "directory.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// API Handlers

func handleCreateComment(w http.ResponseWriter, r *http.Request) {
	var comment Comment

	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := createComment(&comment); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(comment); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleUpdateComment(w http.ResponseWriter, r *http.Request) {
	// Extract comment ID from URL path
	commentID := chi.URLParam(r, "id")

	var req struct {
		CommentText string `json:"comment_text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := updateComment(commentID, req.CommentText); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "updated"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	// Extract comment ID from URL path
	commentID := chi.URLParam(r, "id")

	if err := deleteComment(commentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "deleted"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
