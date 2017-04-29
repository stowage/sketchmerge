package sketchmerge

import (
	"encoding/json"
	"log"
	"github.com/twinj/uuid"
	"strings"
)
type Resolution struct {
	LocalBranch MainDiff `json:"local_branch"`
	RemoteBranch MainDiff `json:"remote_branch"`
}

type Collisions struct {
	Resolutions map[string]Resolution `json:"resolution"`
}


func PrepareCollisions(idDiffMapDocDst, idDiffMapDocSrc map[string]interface{}) Collisions {

	info1, _ := json.MarshalIndent(idDiffMapDocDst, "", "  ")
	info2, _ := json.MarshalIndent(idDiffMapDocSrc, "", "  ")
	log.Printf("collisions: %v \n %v \n", string(info1), string(info2))

	collisions := Collisions{make(map[string]Resolution)}
	collisionCodes := make(map[string]string)

	for key, itemDst := range idDiffMapDocDst {
		if itemSrc, ok := idDiffMapDocSrc[key]; ok {
			code, ok := collisionCodes[key]
			if !ok {
				code = strings.ToUpper(uuid.NewV4().String())
				collisionCodes[key] = code
			}

			localBrunch := MainDiff{Diff: itemDst.(Difference).GetDiff()["local"].(map[string]interface{})}
			itemDst.(Difference).SetCollision(code)
			remoteBrunch := MainDiff{Diff: itemSrc.(Difference).GetDiff()["local"].(map[string]interface{})}
			itemSrc.(Difference).SetCollision(code)
			collisions.Resolutions[code] = Resolution{localBrunch, remoteBrunch}
		}
	}
	return collisions

}

func ProcessCollisionsData(collisionsDst, collisionsSrc Collisions) ([]byte, error) {
	collisions := map[string]Collisions { "collision_dst" : collisionsDst, "collision_src" : collisionsSrc}
	return json.MarshalIndent(collisions, "", "  ")

}
