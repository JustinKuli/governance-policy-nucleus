// Copyright Contributors to the Open Cluster Management project

package v1beta1

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/util/validation"
)

var sampleNames = []string{
	"foo", "bar", "baz", "boo", "default", "kube-one", "kube-two", "kube-three",
}

// matches is only used to unit-test the behavior of `match`.
func (t *Target) matches(names []string) ([]string, error) {
	// Using a map to ensure each entry in the result is unique.
	set := make(map[string]struct{})

	for _, name := range names {
		matched, err := t.match(name)
		if err != nil {
			return nil, err
		}

		if matched {
			set[name] = struct{}{}
		}
	}

	matchingNames := make([]string, 0, len(set))
	for ns := range set {
		matchingNames = append(matchingNames, ns)
	}

	return matchingNames, nil
}

// Fuzz test to verify that excluding "*" always matches 0 names. The
// `Include` list and the input names are both fuzzed.
func FuzzMatchesExcludeAll(f *testing.F) {
	for _, tc := range []string{"", "*", "?", "foo", "quux"} {
		f.Add(tc, tc, tc)
	}

	f.Fuzz(func(t *testing.T, inc1, inc2, extraName string) {
		errs := validation.IsDNS1123Subdomain(extraName)
		if len(errs) != 0 { // K8s Names are usually required to be valid RFC 1123 DNS subdomains.
			t.Skip()
		}

		inc := []NonEmptyString{NonEmptyString(inc1), NonEmptyString(inc2)}
		sel := Target{Include: inc, Exclude: []NonEmptyString{"*"}}

		got, err := sel.matches(append(sampleNames, extraName))
		if err != nil {
			if errors.Is(err, filepath.ErrBadPattern) {
				t.Skip()
			}
			t.Errorf("Unexpected error '%v' when including '%v' and '%v', with ns '%v'",
				err, inc1, inc2, extraName)
		}
		if len(got) != 0 {
			t.Errorf("Got non-empty matches '%v', when including '%v' and '%v', with ns '%v'",
				got, inc1, inc2, extraName)
		}
	})
}

// Fuzz test to verify that including "*" will always match all names when
// excludes is empty. Additional items in the `Include` list and the input
// names are both fuzzed.
func FuzzMatchesIncludeAll(f *testing.F) {
	for _, tc := range []string{"", "*", "?", "foo", "quux"} {
		f.Add(tc, tc, tc)
	}

	f.Fuzz(func(t *testing.T, inc1, inc2, extraName string) {
		errs := validation.IsDNS1123Subdomain(extraName)
		if len(errs) != 0 { // K8s Names are usually required to be valid RFC 1123 DNS subdomains.
			t.Skip()
		}

		inc := []NonEmptyString{"*", NonEmptyString(inc1), NonEmptyString(inc2)}
		sel := Target{Include: inc, Exclude: []NonEmptyString{}}

		got, err := sel.matches(append(sampleNames, extraName))
		if err != nil {
			if errors.Is(err, filepath.ErrBadPattern) {
				t.Skip()
			}
			t.Errorf("Unexpected error '%v' when including '%v' and '%v', with ns '%v'",
				err, inc1, inc2, extraName)
		}

		desiredLen := len(sampleNames) + 1
		for _, ns := range sampleNames {
			if ns == extraName {
				desiredLen--
			}
		}

		if len(got) != desiredLen {
			t.Errorf("Incorrect matches '%v', when including '%v' and '%v', with ns '%v'",
				got, inc1, inc2, extraName)
		}
	})
}

func TestMatches(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inc  []NonEmptyString
		exc  []NonEmptyString
		want []string
	}{
		"include trailing wildcard": {
			inc:  []NonEmptyString{"kube-*"},
			exc:  []NonEmptyString{},
			want: []string{"kube-one", "kube-two", "kube-three"},
		},
		"exclude trailing wildcard": {
			inc:  []NonEmptyString{"*"},
			exc:  []NonEmptyString{"kube-*"},
			want: []string{"foo", "bar", "baz", "boo", "default"},
		},
		"include leading wildcard": {
			inc:  []NonEmptyString{"*o"},
			exc:  []NonEmptyString{},
			want: []string{"foo", "boo", "kube-two"},
		},
		"exclude leading wildcard": {
			inc:  []NonEmptyString{"*"},
			exc:  []NonEmptyString{"*o"},
			want: []string{"bar", "baz", "default", "kube-one", "kube-three"},
		},
		"include middle wildcard": {
			inc:  []NonEmptyString{"*a*"},
			exc:  []NonEmptyString{},
			want: []string{"bar", "baz", "default"},
		},
		"exclude middle wildcard": {
			inc:  []NonEmptyString{"*"},
			exc:  []NonEmptyString{"*a*"},
			want: []string{"foo", "boo", "kube-one", "kube-two", "kube-three"},
		},
		"include one specific": {
			inc:  []NonEmptyString{"foo"},
			exc:  []NonEmptyString{},
			want: []string{"foo"},
		},
		"exclude one specific": {
			inc:  []NonEmptyString{"*"},
			exc:  []NonEmptyString{"foo"},
			want: []string{"bar", "baz", "boo", "default", "kube-one", "kube-two", "kube-three"},
		},
		"include multiple specific": {
			inc:  []NonEmptyString{"foo", "bar", "kube-three"},
			exc:  []NonEmptyString{},
			want: []string{"foo", "bar", "kube-three"},
		},
		"exclude multiple specific": {
			inc:  []NonEmptyString{"*"},
			exc:  []NonEmptyString{"foo", "bar", "kube-three"},
			want: []string{"baz", "boo", "default", "kube-one", "kube-two"},
		},
		"include single char wildcards": {
			inc:  []NonEmptyString{"?a?"},
			exc:  []NonEmptyString{},
			want: []string{"bar", "baz"},
		},
		"exclude single char wildcards": {
			inc:  []NonEmptyString{"*"},
			exc:  []NonEmptyString{"?a?"},
			want: []string{"foo", "boo", "default", "kube-one", "kube-two", "kube-three"},
		},
		"include character class": {
			inc:  []NonEmptyString{"[fb]oo"},
			exc:  []NonEmptyString{},
			want: []string{"foo", "boo"},
		},
		"exclude character class": {
			inc:  []NonEmptyString{"*"},
			exc:  []NonEmptyString{"[fb]oo"},
			want: []string{"bar", "baz", "default", "kube-one", "kube-two", "kube-three"},
		},
		// Note that if the NamespaceSelector is empty, it should return no namespaces, but that
		// is handled separately from this `matches` function
		"include and exclude are both empty": {
			inc:  []NonEmptyString{},
			exc:  []NonEmptyString{},
			want: sampleNames,
		},
	}

	for name, tcase := range tests {
		sel := Target{Include: tcase.inc, Exclude: tcase.exc}

		got, err := sel.matches(sampleNames)
		if err != nil {
			t.Errorf("Unexpected error '%v', in test '%v'", err, name)
		}

		less := func(a, b string) bool { return a < b }
		diff := cmp.Diff(tcase.want, got, cmpopts.SortSlices(less))

		if diff != "" {
			t.Errorf("Mismatch in test '%v': %v", name, diff)
		}
	}
}

func TestMatchesErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inc []NonEmptyString
		exc []NonEmptyString
		// All tests here should throw an error
	}{
		"include is malformed": {
			inc: []NonEmptyString{"kube-[system"},
			exc: []NonEmptyString{},
		},
		"exclude is malformed": {
			inc: []NonEmptyString{"*"},
			exc: []NonEmptyString{"foobar["},
		},
		"both are malformed": {
			inc: []NonEmptyString{"foo["},
			exc: []NonEmptyString{"bar["},
		},
	}

	for name, tcase := range tests {
		sel := Target{Include: tcase.inc, Exclude: tcase.exc}

		_, err := sel.matches(sampleNames)
		if err == nil {
			t.Errorf("Expected an error in test '%v', but got nil", name)
		}

		if !errors.Is(err, filepath.ErrBadPattern) {
			t.Errorf("Error mismatch in test '%v', got '%v' wanted filepath.ErrBadPattern",
				name, err)
		}
	}
}
