package hash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1alpha1 "github.com/undeadops/fooOperator/api/v1alpha1"
)

func TestHashUtils(t *testing.T) {
	templateRed := generatePodTemplate("red")
	hashRed := ComputeTemplateHash(&templateRed.Spec, nil)
	template := generatePodTemplate("red")

	t.Run("HashForSameTemplates", func(t *testing.T) {
		podHash := ComputeTemplateHash(&template.Spec, nil)
		assert.Equal(t, hashRed, podHash)
	})
	t.Run("HashForDifferentTemplates", func(t *testing.T) {
		podHash := ComputeTemplateHash(&template.Spec, pointer.Int32(1))
		assert.NotEqual(t, hashRed, podHash)
	})
}

func generatePodTemplate(image string) *v1alpha1.Foo {
	podLabels := map[string]string{"name": image}

	return &v1alpha1.Foo{
		ObjectMeta: metav1.ObjectMeta{
			Labels: podLabels,
		},
		Spec: v1alpha1.FooSpec{
			Pod: v1alpha1.FooPodSpec{
				Image: image,
			},
			Ingress: v1alpha1.FooIngressSpec{
				ServicePort: int32(8080),
			},
		},
	}
}
