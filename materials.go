package contempt

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

var materials map[string]string

func readBillOfMaterials(target string) map[string]string {
	res := make(map[string]string)
	bs, err := os.ReadFile(target)
	if err != nil {
		log.Printf("Unable to read existing file (%s) for BOM: %v", target, err)
		return res
	}

	bomLine := strings.SplitN(string(bs), "\n", 3)[1]
	if !strings.HasPrefix(bomLine, "# BOM: ") {
		log.Printf("Existing file (%s) does not appear to have a BOM", target)
		return res
	}

	if err := json.Unmarshal([]byte(strings.TrimPrefix(bomLine, "# BOM: ")), &res); err != nil {
		log.Printf("Existing file (%s) has invalid BOM: %v", target, err)
		return res
	}

	return res
}

type Change struct {
	Material string
	Old      string
	New      string
}

func diffMaterials(oldBom, newBom map[string]string) []Change {
	var res []Change
	for i := range newBom {
		if oldBom[i] != newBom[i] {
			res = append(res, Change{
				Material: i,
				Old:      oldBom[i],
				New:      newBom[i],
			})
		}
	}
	return res
}
