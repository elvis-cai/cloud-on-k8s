// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"testing"

	"github.com/elastic/cloud-on-k8s/operators/pkg/apis/kibana/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPseudoNamespacedResourceName(t *testing.T) {
	type args struct {
		kibana v1alpha1.Kibana
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{kibana: v1alpha1.Kibana{ObjectMeta: metav1.ObjectMeta{Name: "a-name"}}},
			want: "a-name-kibana",
		},
		{
			args: args{kibana: v1alpha1.Kibana{ObjectMeta: metav1.ObjectMeta{Name: "another-name"}}},
			want: "another-name-kibana",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PseudoNamespacedResourceName(tt.args.kibana); got != tt.want {
				t.Errorf("PseudoNamespacedResourceName() = %v, want %v", got, tt.want)
			}
		})
	}
}
