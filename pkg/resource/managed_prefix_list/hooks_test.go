// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package managed_prefix_list

import (
	"testing"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

func entry(cidr, desc string) *svcapitypes.AddPrefixListEntry {
	e := &svcapitypes.AddPrefixListEntry{CIDR: aws.String(cidr)}
	if desc != "" {
		e.Description = aws.String(desc)
	}
	return e
}

func resourceWithEntries(entries ...*svcapitypes.AddPrefixListEntry) *resource {
	return &resource{ko: &svcapitypes.ManagedPrefixList{
		Spec: svcapitypes.ManagedPrefixListSpec{
			Name:    aws.String("pl-test"),
			Entries: entries,
		},
	}}
}

// Asserted through the real newResourceDelta so the generator.yaml wiring
// (compare.is_ignored plus the delta_post_compare hook) is covered too, not just
// customPostCompare in isolation.
func TestEntriesDelta(t *testing.T) {
	tests := []struct {
		name     string
		a        *resource
		b        *resource
		expectPL bool
	}{
		{
			// The regression: GetManagedPrefixListEntries returns entries in an
			// order AWS chooses, and an order-sensitive comparison reported a
			// difference that no update could resolve.
			name:     "same entries, different order",
			a:        resourceWithEntries(entry("10.0.1.0/24", "a"), entry("10.0.2.0/24", "b")),
			b:        resourceWithEntries(entry("10.0.2.0/24", "b"), entry("10.0.1.0/24", "a")),
			expectPL: false,
		},
		{
			name:     "identical entries",
			a:        resourceWithEntries(entry("10.0.1.0/24", "a")),
			b:        resourceWithEntries(entry("10.0.1.0/24", "a")),
			expectPL: false,
		},
		{
			name:     "added entry",
			a:        resourceWithEntries(entry("10.0.1.0/24", "a"), entry("10.0.2.0/24", "b")),
			b:        resourceWithEntries(entry("10.0.1.0/24", "a")),
			expectPL: true,
		},
		{
			name:     "removed entry",
			a:        resourceWithEntries(entry("10.0.1.0/24", "a")),
			b:        resourceWithEntries(entry("10.0.1.0/24", "a"), entry("10.0.2.0/24", "b")),
			expectPL: true,
		},
		{
			name:     "same CIDR, changed description",
			a:        resourceWithEntries(entry("10.0.1.0/24", "before")),
			b:        resourceWithEntries(entry("10.0.1.0/24", "after")),
			expectPL: true,
		},
		{
			name:     "description added",
			a:        resourceWithEntries(entry("10.0.1.0/24", "a")),
			b:        resourceWithEntries(entry("10.0.1.0/24", "")),
			expectPL: true,
		},
		{
			name:     "replaced entry",
			a:        resourceWithEntries(entry("10.0.1.0/24", "a")),
			b:        resourceWithEntries(entry("10.0.9.0/24", "a")),
			expectPL: true,
		},
		{
			name:     "both empty",
			a:        resourceWithEntries(),
			b:        resourceWithEntries(),
			expectPL: false,
		},
		{
			name:     "entries added to an empty list",
			a:        resourceWithEntries(entry("10.0.1.0/24", "a")),
			b:        resourceWithEntries(),
			expectPL: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta := newResourceDelta(tt.a, tt.b)
			assert.Equal(t, tt.expectPL, delta.DifferentAt("Spec.Entries"))
		})
	}
}

// customPostCompare runs on every delta, including before the resource exists,
// so it must tolerate nil inputs rather than panic.
func TestCustomPostCompareNilSafety(t *testing.T) {
	empty := &resource{}
	for _, tt := range []struct {
		name string
		a    *resource
		b    *resource
	}{
		{"both nil", nil, nil},
		{"a nil", nil, resourceWithEntries(entry("10.0.1.0/24", "a"))},
		{"b nil", resourceWithEntries(entry("10.0.1.0/24", "a")), nil},
		{"nil ko", empty, empty},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				customPostCompare(ackcompare.NewDelta(), tt.a, tt.b)
			})
		})
	}
}
