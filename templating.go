package contempt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/csmith/contempt/pkg/materials"
	"github.com/csmith/contempt/pkg/template"
	"github.com/csmith/contempt/pkg/template/sources"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

var engine *template.Engine

func InitTemplates(imageRegistry, alpineMirror string, includes fs.FS) {
	engine = template.NewEngine(
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
		includes,
	)

	engine.Register(sources.AlpinePackagesSource(alpineMirror))
	engine.Register(sources.ImageSource(imageRegistry))
	engine.Register(sources.GitSource())
	engine.Register(sources.HttpSource())
	engine.Register(sources.AlpineReleaseSource(alpineMirror))
	engine.Register(sources.GoReleaseSource())
	engine.Register(sources.PostgresReleaseSource())
	engine.Register(sources.UtilSource())
}

func Render(sourceLink, inBase, inRelativePath string) ([]byte, materials.BOM, error) {
	inFile := filepath.Join(inBase, inRelativePath)

	writer := &bytes.Buffer{}
	newMaterials, err := engine.Execute(writer, inFile)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to render template file %s: %v", inFile, err)
	}

	bom, _ := json.Marshal(newMaterials)
	header := fmt.Sprintf("# Generated from %s%s\n# BOM: %s\n\n", sourceLink, inRelativePath, bom)

	return append([]byte(header), writer.Bytes()...), newMaterials, nil
}

func RenderWithBOM(sourceLink, inBase, inRelativePath string, bom materials.BOM) ([]byte, error) {
	inFile := filepath.Join(inBase, inRelativePath)

	writer := &bytes.Buffer{}
	err := engine.ExecuteWithBOM(writer, inFile, bom)
	if err != nil {
		return nil, fmt.Errorf("unable to render template file %s: %v", inFile, err)
	}

	bomJson, _ := json.Marshal(bom)
	header := fmt.Sprintf("# Generated from %s%s\n# BOM: %s\n\n", sourceLink, inRelativePath, bomJson)

	return append([]byte(header), writer.Bytes()...), nil
}

func Generate(sourceLink, inBase, inRelativePath, outFile string) ([]materials.Change, error) {
	return generateInternal(sourceLink, inBase, inRelativePath, outFile, nil)
}

func GenerateWithOverrides(sourceLink, inBase, inRelativePath, outFile string, overrides materials.BOM) ([]materials.Change, error) {
	return generateInternal(sourceLink, inBase, inRelativePath, outFile, overrides)
}

func generateInternal(sourceLink, inBase, inRelativePath, outFile string, overrides materials.BOM) ([]materials.Change, error) {
	oldMaterials := materials.Read(outFile)

	// Get new versions for all materials
	newMaterials, err := CheckVersions(inBase, inRelativePath)
	if err != nil {
		return nil, fmt.Errorf("unable to check versions: %v", err)
	}

	// Build BOM: new versions for always-approved and overridden materials, old for others
	finalBom := make(materials.BOM)
	for material, oldVersion := range oldMaterials {
		if override, ok := overrides[material]; ok {
			finalBom[material] = override
		} else if materials.IsAlwaysApproved(material) {
			if newVersion, ok := newMaterials[material]; ok {
				finalBom[material] = newVersion
			} else {
				finalBom[material] = oldVersion
			}
		} else {
			finalBom[material] = oldVersion
		}
	}

	// Add any new materials that weren't in old BOM
	for material, newVersion := range newMaterials {
		if _, exists := oldMaterials[material]; !exists {
			finalBom[material] = newVersion
		}
	}

	content, err := RenderWithBOM(sourceLink, inBase, inRelativePath, finalBom)
	if err != nil {
		return nil, fmt.Errorf("unable to render template file %s: %v", outFile, err)
	}

	if err := os.WriteFile(outFile, content, os.FileMode(0600)); err != nil {
		return nil, fmt.Errorf("unable to write container file to %s: %v", outFile, err)
	}

	return materials.Diff(oldMaterials, finalBom), nil
}

func CheckVersions(inBase, inRelativePath string) (materials.BOM, error) {
	inFile := filepath.Join(inBase, inRelativePath)
	return engine.Execute(io.Discard, inFile)
}
