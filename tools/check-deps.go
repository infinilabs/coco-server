// Package main provides a tool to validate module dependency hierarchy
// and detect potential circular import risks.
//
// Usage: go run tools/check-deps.go
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Layer definitions following ARCHITECTURE.md
var layers = map[string]int{
	// Layer 0: Core Foundation
	"common": 0,
	"core":   0,

	// Layer 1: Basic Services
	"document":   1,
	"attachment": 1,
	"system":     1,

	// Layer 2: Domain Services
	"connector": 2,
	"llm":       2,
	"search":    2,

	// Layer 3: Business Logic
	"datasource":  3,
	"integration": 3,

	// Layer 4: Application Services
	"assistant": 4,
}

type Dependency struct {
	From   string
	To     string
	File   string
	Line   int
	Valid  bool
	Reason string
}

func main() {
	fmt.Println("🔍 Checking module dependency hierarchy...")

	// Parse all Go files in modules directory
	deps, err := analyzeDependencies("modules")
	if err != nil {
		fmt.Printf("❌ Error analyzing dependencies: %v\n", err)
		os.Exit(1)
	}

	// Validate dependencies against layer rules
	violations := validateDependencies(deps)

	// Report results
	if len(violations) == 0 {
		fmt.Println("✅ All dependencies comply with the architecture!")
		fmt.Printf("📊 Total dependencies analyzed: %d\n", len(deps))
		printDependencyMatrix(deps)
	} else {
		fmt.Printf("❌ Found %d architecture violations:\n\n", len(violations))
		for _, violation := range violations {
			fmt.Printf("🚫 %s → %s\n", violation.From, violation.To)
			fmt.Printf("   File: %s:%d\n", violation.File, violation.Line)
			fmt.Printf("   Reason: %s\n\n", violation.Reason)
		}
		os.Exit(1)
	}
}

func analyzeDependencies(moduleDir string) ([]Dependency, error) {
	var dependencies []Dependency

	err := filepath.Walk(moduleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Determine which module this file belongs to
		relPath, _ := filepath.Rel(moduleDir, path)
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) == 0 {
			return nil
		}
		currentModule := parts[0]

		// Skip if not a known module
		if _, exists := layers[currentModule]; !exists {
			return nil
		}

		// Parse the Go file
		fileDeps, err := parseFileImports(path, currentModule)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		dependencies = append(dependencies, fileDeps...)
		return nil
	})

	return dependencies, err
}

func parseFileImports(filename string, currentModule string) ([]Dependency, error) {
	var dependencies []Dependency

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, imp := range node.Imports {
		if imp.Path == nil {
			continue
		}

		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check if this is a module import
		if strings.HasPrefix(importPath, "infini.sh/coco/modules/") {
			targetPath := strings.TrimPrefix(importPath, "infini.sh/coco/modules/")
			parts := strings.Split(targetPath, "/")
			if len(parts) > 0 {
				targetModule := parts[0]

				// Skip self-imports and submodule imports
				if targetModule == currentModule {
					continue
				}

				// Check if target is a known module
				if _, exists := layers[targetModule]; exists {
					pos := fset.Position(imp.Pos())
					dependencies = append(dependencies, Dependency{
						From: currentModule,
						To:   targetModule,
						File: filename,
						Line: pos.Line,
					})
				}
			}
		}
	}

	return dependencies, nil
}

func validateDependencies(deps []Dependency) []Dependency {
	var violations []Dependency

	for _, dep := range deps {
		fromLayer, fromExists := layers[dep.From]
		toLayer, toExists := layers[dep.To]

		if !fromExists || !toExists {
			continue
		}

		// Check layer rules: can only import from same or lower layers
		if toLayer > fromLayer {
			dep.Valid = false
			dep.Reason = fmt.Sprintf("Layer %d module cannot import from Layer %d module (upward dependency)", fromLayer, toLayer)
			violations = append(violations, dep)
		} else if toLayer == fromLayer && dep.From != dep.To {
			// Peer imports within same layer are discouraged but not forbidden
			// Only flag as violation for certain combinations
			if shouldFlagPeerImport(dep.From, dep.To) {
				dep.Valid = false
				dep.Reason = fmt.Sprintf("Peer import within Layer %d (consider refactoring)", fromLayer)
				violations = append(violations, dep)
			}
		} else {
			dep.Valid = true
		}
	}

	return violations
}

func shouldFlagPeerImport(from, to string) bool {
	// Currently allow peer imports within same layer
	// Could be made more strict in the future
	return false
}

func printDependencyMatrix(deps []Dependency) {
	fmt.Println("\n📋 Dependency Summary by Layer:")

	// Count dependencies by layer
	layerCounts := make(map[int]map[int]int)
	for layer := 0; layer <= 4; layer++ {
		layerCounts[layer] = make(map[int]int)
	}

	for _, dep := range deps {
		fromLayer := layers[dep.From]
		toLayer := layers[dep.To]
		layerCounts[fromLayer][toLayer]++
	}

	// Print matrix
	fmt.Printf("\n%-12s", "From\\To")
	for layer := 0; layer <= 4; layer++ {
		fmt.Printf("%-8s", fmt.Sprintf("L%d", layer))
	}
	fmt.Println()

	for fromLayer := 0; fromLayer <= 4; fromLayer++ {
		fmt.Printf("Layer %-6d", fromLayer)
		for toLayer := 0; toLayer <= 4; toLayer++ {
			count := layerCounts[fromLayer][toLayer]
			if count > 0 {
				fmt.Printf("%-8d", count)
			} else {
				fmt.Printf("%-8s", "-")
			}
		}
		fmt.Println()
	}

	// Print module details
	fmt.Println("\n📦 Module Layer Assignment:")

	// Group modules by layer
	modulesByLayer := make(map[int][]string)
	for module, layer := range layers {
		modulesByLayer[layer] = append(modulesByLayer[layer], module)
	}

	for layer := 0; layer <= 4; layer++ {
		if modules, exists := modulesByLayer[layer]; exists {
			sort.Strings(modules)
			fmt.Printf("Layer %d: %s\n", layer, strings.Join(modules, ", "))
		}
	}
}
