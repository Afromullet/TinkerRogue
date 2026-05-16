// Package templates loads, validates, and stores all static JSON game data.
//
// All loaders run once at boot via ReadGameData(); see registry.go for the
// load-order contract. Registries are package-level globals consumed read-only
// after boot.
//
// For the full gamedata reference and the procedure for adding a new gamedata
// JSON file, see resources/docs/project_documentation/Architecture/GAMEDATA_OVERVIEW.md.
package templates
