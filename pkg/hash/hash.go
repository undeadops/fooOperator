package hash

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"

	"k8s.io/apimachinery/pkg/util/rand"

	v1alpha1 "github.com/undeadops/fooOperator/api/v1alpha1"
)

// ComputePodTemplateHash returns a hash value calculated from pod template.
// The hash will be safe encoded to avoid bad words.
func ComputeTemplateHash(template *v1alpha1.FooSpec, collisionCount *int32) string {
	podTemplateSpecHasher := fnv.New32a()
	stepsBytes, err := json.Marshal(template)
	if err != nil {
		panic(err)
	}
	_, err = podTemplateSpecHasher.Write(stepsBytes)
	if err != nil {
		panic(err)
	}
	if collisionCount != nil {
		collisionCountBytes := make([]byte, 8)
		binary.LittleEndian.PutUint32(collisionCountBytes, uint32(*collisionCount))
		_, err = podTemplateSpecHasher.Write(collisionCountBytes)
		if err != nil {
			panic(err)
		}
	}
	return rand.SafeEncodeString(fmt.Sprint(podTemplateSpecHasher.Sum32()))
}
