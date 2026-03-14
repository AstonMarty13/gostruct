![Go Version](https://img.shields.io/badge/go-1.26-blue.svg)

# 🚀 GoStruct

**GoStruct** est un outil CLI ultra-léger écrit en **Go** qui permet de générer instantanément une arborescence de projet standardisée. Fini les tâches répétitives, lancez vos nouveaux projets Go en une seule commande !

## ✨ Fonctionnalités

* 📂 **Génération automatique** de l'arborescence standard (`cmd/`, `internal/`, `pkg/`).
* ⚙️ **Initialisation du module** (`go mod init`) intégrée.
* 🛡️ **Sécurité anti-écrasement** : Vérifie si le dossier existe déjà avant d'agir.
* 📦 **Mode Full** : Option pour ajouter les dossiers `api/`, `web/` et `docs/`.

---

## 🛠️ Installation

Assurez-vous d'avoir [Go](https://go.dev/) installé sur votre système.

1.  **Clonez le projet** (ou créez le fichier `main.go`).
2.  **Installez l'outil globalement** :
    ```bash
    go install
    ```
    *L'exécutable sera disponible dans votre `$GOPATH/bin`.*

---

## 🚀 Utilisation

### Structure standard
Parfait pour les petits outils ou les bibliothèques.
```bash
gostruct mon-projet