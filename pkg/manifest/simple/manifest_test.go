// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simple_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ethersphere/bee/v2/pkg/manifest/simple"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

func TestNilPath(t *testing.T) {
	t.Parallel()

	m := simple.NewManifest()
	n, err := m.Lookup("")
	if err == nil {
		t.Fatalf("expected error, got reference %s", n.Reference())
	}
}

// struct for manifest entries for test cases
type e struct {
	path      string
	reference swarm.Address
	metadata  map[string]string
}

type testCase struct {
	name    string
	entries []e // entries to add to manifest
}

func makeTestCases(t *testing.T) []testCase {
	t.Helper()

	return []testCase{
		{
			name:    "empty-manifest",
			entries: nil,
		},
		{
			name: "one-entry",
			entries: []e{
				{
					path:      "entry-1",
					reference: swarm.RandAddress(t),
				},
			},
		},
		{
			name: "two-entries",
			entries: []e{
				{
					path:      "entry-1.txt",
					reference: swarm.RandAddress(t),
				},
				{
					path:      "entry-2.png",
					reference: swarm.RandAddress(t),
				},
			},
		},
		{
			name: "nested-entries",
			entries: []e{
				{
					path:      "text/robots.txt",
					reference: swarm.RandAddress(t),
				},
				{
					path:      "img/1.png",
					reference: swarm.RandAddress(t),
				},
				{
					path:      "img/2.jpg",
					reference: swarm.RandAddress(t),
				},
				{
					path:      "readme.md",
					reference: swarm.RandAddress(t),
				},
				{
					path: "/",
					metadata: map[string]string{
						"index-document": "readme.md",
						"error-document": "404.html",
					},
				},
			},
		},
	}
}

func TestEntries(t *testing.T) {
	t.Parallel()

	for _, tc := range makeTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := simple.NewManifest()

			checkLength(t, m, 0)

			// add entries
			for i, e := range tc.entries {
				err := m.Add(e.path, e.reference.String(), e.metadata)
				if err != nil {
					t.Fatal(err)
				}

				checkLength(t, m, i+1)
				checkEntry(t, m, e.reference.String(), e.path)
			}

			manifestLen := m.Length()

			if len(tc.entries) != manifestLen {
				t.Fatalf("expected %d entries, found %d", len(tc.entries), manifestLen)
			}

			if manifestLen == 0 {
				// special case for empty manifest
				return
			}

			// replace entry
			lastEntry := tc.entries[len(tc.entries)-1]

			newReference := swarm.RandAddress(t).String()

			err := m.Add(lastEntry.path, newReference, map[string]string{})
			if err != nil {
				t.Fatal(err)
			}

			checkLength(t, m, manifestLen) // length should not have changed
			checkEntry(t, m, newReference, lastEntry.path)

			// remove entries
			err = m.Remove("invalid/path.ext") // try removing inexistent entry
			if err != nil {
				t.Fatal(err)
			}

			checkLength(t, m, manifestLen) // length should not have changed

			for i, e := range tc.entries {
				err = m.Remove(e.path)
				if err != nil {
					t.Fatal(err)
				}

				entry, err := m.Lookup(e.path)
				if entry != nil || !errors.Is(err, simple.ErrNotFound) {
					t.Fatalf("expected path %v not to be present in the manifest, but it was found", e.path)
				}

				checkLength(t, m, manifestLen-i-1)
			}
		})
	}
}

// checkLength verifies that the given manifest length and integer match.
func checkLength(t *testing.T, m simple.Manifest, length int) {
	t.Helper()

	if m.Length() != length {
		t.Fatalf("expected length to be %d, but is %d instead", length, m.Length())
	}
}

// checkEntry verifies that an entry is equal to the one retrieved from the given manifest and path.
func checkEntry(t *testing.T, m simple.Manifest, reference, path string) {
	t.Helper()

	n, err := m.Lookup(path)
	if err != nil {
		t.Fatal(err)
	}
	if n.Reference() != reference {
		t.Fatalf("expected reference %s, got: %s", reference, n.Reference())
	}
}

// TestMarshal verifies that created manifests are successfully marshalled and unmarshalled.
// This function will add all test case entries to a manifest and marshal it.
// After, it will unmarshal the result, and verify that it is equal to the original manifest.
func TestMarshal(t *testing.T) {
	t.Parallel()

	for _, tc := range makeTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := simple.NewManifest()

			for _, e := range tc.entries {
				err := m.Add(e.path, e.reference.String(), e.metadata)
				if err != nil {
					t.Fatal(err)
				}
			}

			b, err := m.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}

			um := simple.NewManifest()
			if err := um.UnmarshalBinary(b); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(m, um) {
				t.Fatalf("marshalled and unmarshalled manifests are not equal: %v, %v", m, um)
			}
		})
	}
}

func TestHasPrefix(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		toAdd       []string
		testPrefix  []string
		shouldExist []bool
	}{
		{
			name: "simple",
			toAdd: []string{
				"index.html",
				"img/1.png",
				"img/2.png",
				"robots.txt",
			},
			testPrefix: []string{
				"img/",
				"images/",
			},
			shouldExist: []bool{
				true,
				false,
			},
		},
		{
			name: "nested-single",
			toAdd: []string{
				"some-path/file.ext",
			},
			testPrefix: []string{
				"some-path/",
				"some-path/file",
				"some-other-path/",
			},
			shouldExist: []bool{
				true,
				true,
				false,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := simple.NewManifest()

			for _, e := range tc.toAdd {
				err := m.Add(e, "", nil)
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			}

			for i := 0; i < len(tc.testPrefix); i++ {
				testPrefix := tc.testPrefix[i]
				shouldExist := tc.shouldExist[i]

				exists := m.HasPrefix(testPrefix)

				if shouldExist != exists {
					t.Errorf("expected prefix path %s to be %t, was %t", testPrefix, shouldExist, exists)
				}
			}
		})
	}
}
