package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
)

const repoDir = "repos"

type PageData struct {
	Repos []string
}

func main() {
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		os.Mkdir(repoDir, 0755)
	}

	http.HandleFunc("/", listHandler)
	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/delete", deleteHandler)

	backendPath := "/usr/lib/git-core/git-http-backend"
	if _, err := os.Stat(backendPath); err == nil {
		cwd, _ := os.Getwd()
		absRepoDir := filepath.Join(cwd, repoDir)
		
		gitHandler := &cgi.Handler{
			Path: backendPath,
			Env: append(os.Environ(),
				"GIT_PROJECT_ROOT="+absRepoDir,
				"GIT_HTTP_EXPORT_ALL=true",
			),
		}
		
		http.Handle("/repos/", http.StripPrefix("/repos", gitHandler))
	} else {
		fmt.Printf("Warning: git-http-backend not found at %s. HTTP cloning will not work.\n", backendPath)
	}

	fmt.Println("Server started:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(repoDir)
	if err != nil {
		http.Error(w, "Unable to read repos directory", http.StatusInternalServerError)
		return
	}

	var repos []string
	for _, f := range files {
		if f.IsDir() && filepath.Ext(f.Name()) == ".git" {
			repos = append(repos, f.Name())
		}
	}

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Unable to parse template", http.StatusInternalServerError)
		return
	}

	data := PageData{Repos: repos}
	tmpl.Execute(w, data)
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repoName := r.FormValue("name")
	if repoName == "" {
		http.Error(w, "Repository name is required", http.StatusBadRequest)
		return
	}

	if filepath.Base(repoName) != repoName {
		http.Error(w, "Invalid repository name", http.StatusBadRequest)
		return
	}

	repoPath := filepath.Join(repoDir, repoName+".git")
	cmd := exec.Command("git", "init", "--bare", repoPath)
	if err := cmd.Run(); err != nil {
		http.Error(w, "Failed to create repository", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repoName := r.FormValue("name")
	if repoName == "" {
		http.Error(w, "Repository name is required", http.StatusBadRequest)
		return
	}

	if filepath.Base(repoName) != repoName {
		http.Error(w, "Invalid repository name", http.StatusBadRequest)
		return
	}

	repoPath := filepath.Join(repoDir, repoName)
	if err := os.RemoveAll(repoPath); err != nil {
		http.Error(w, "Failed to delete repository", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
