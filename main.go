package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	isFull := flag.Bool("full", false, "Créer une structure complète")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: gostruct [-full] <nom-du-projet>")
		return
	}

	projectName := args[0]

	// 1. Vérification d'existence
	if _, err := os.Stat(projectName); !os.IsNotExist(err) {
		fmt.Printf("❌ Erreur : Le dossier '%s' existe déjà.\n", projectName)
		return
	}

	// 2. Dossiers à créer
	dirs := []string{
		filepath.Join(projectName, "cmd"),
		filepath.Join(projectName, "internal"),
		filepath.Join(projectName, "pkg"),
		filepath.Join(projectName, "docs", "skills"), // Ajout du dossier skills
	}

	if *isFull {
		dirs = append(dirs, filepath.Join(projectName, "api"), filepath.Join(projectName, "web"))
	}

	for _, dir := range dirs {
		os.MkdirAll(dir, 0755)
	}

	// 3. Création du .gitignore
	gitignoreContent := "bin/\n*.exe\n*.out\nvendor/\n"
	os.WriteFile(filepath.Join(projectName, ".gitignore"), []byte(gitignoreContent), 0644)

	// 4. Création du CLAUDE.md (Guide pour l'IA)
	claudeContent := fmt.Sprintf("# Guide Claude pour %s\n\n## Commandes de Build\n- Build: `go build -o bin/main ./cmd/main.go`\n- Run: `go run ./cmd/main.go`\n", projectName)
	os.WriteFile(filepath.Join(projectName, "CLAUDE.md"), []byte(claudeContent), 0644)

	// 5. Création d'une fiche de skill Go par défaut
	skillContent := "# Skills Go appris\n\n- Structure de dossiers standard\n- Utilisation de `os/exec` pour les commandes CLI\n- Gestion des drapeaux avec `flag`"
	os.WriteFile(filepath.Join(projectName, "docs", "skills", "go_basics.md"), []byte(skillContent), 0644)

	// 6. go mod init
	cmd := exec.Command("go", "mod", "init", projectName)
	cmd.Dir = projectName
	cmd.Run()

	fmt.Printf("🚀 Projet '%s' créé avec .gitignore, CLAUDE.md et dossier skills !\n", projectName)
}
