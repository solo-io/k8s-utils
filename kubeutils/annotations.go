package kubeutils

// SoloClusterAnnotation was originally implemented in skv2's ezkube package
// It allows for easy mimicking of the old clustername field present in
// k8s prior to 1.24. It is stored here to make future changes easier and not
// require multi package switches as if any of the dependent packages are out of
// sync on this field it can cause hard to diagnose issues.
const SoloClusterAnnotation = "cluster.solo.io/cluster"

type annotationStore interface {
	GetAnnotations() map[string]string
}

// GetClusterName from within the annotations
func GetClusterName(as annotationStore) string {
	return as.GetAnnotations()[SoloClusterAnnotation]
}

// AddClusterName to the retrieved annotations
func AddClusterName(as annotationStore, clusterName string) map[string]string {
	anno := as.GetAnnotations()
	if anno == nil {
		anno = map[string]string{}
	}
	anno[SoloClusterAnnotation] = clusterName
	return anno
}

type settableAnnotationStore interface {
	SetAnnotations(map[string]string)
	GetAnnotations() map[string]string
}

// SetClusterName on the retrieved annotations
// Set annotations which while slow is correct.
func SetClusterName(sas settableAnnotationStore, clusterName string) {
	anno := AddClusterName(sas, clusterName)
	sas.SetAnnotations(anno)
}
