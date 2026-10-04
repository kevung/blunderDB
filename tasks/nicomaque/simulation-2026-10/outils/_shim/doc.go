// Fichier sans contrainte de build : le hook de pré-commit lance go vet sur tout dossier Go
// touché ; sans lui, `build constraints exclude all Go files`. Le vrai code est derrière
// `//go:build simulation` et le dossier en `_` le tient hors de `./...`.
package main
