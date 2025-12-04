package v1alpha1

import (
    "k8s.io/apimachinery/pkg/runtime/schema"
    "sigs.k8s.io/controller-runtime/pkg/scheme"
)

var GroupVersion = schema.GroupVersion{Group: "clusters.honse.farm", Version: "v1alpha1"}

var SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

var AddToScheme = SchemeBuilder.AddToScheme
