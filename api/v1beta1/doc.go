// Copyright Contributors to the Open Cluster Management project

// Package v1beta1 contains types which are useful or necessary for implementing a policy "template"
// which will interact with the ocm-io policy framework addon. The types here are meant to be
// embedded in a CRD, allowing utilities (defined elsewhere) to work with all templates more
// generally.
//
// Code here can be changed over time, but the intent is to do so in a backwards-compatible way, and
// not remove things without a deprecation notice. Code here is meant to be usable by templates,
// even before it graduates to `v1`; based on historical trends, many types may begin here, but
// should try to advance to something more stable once they've proven their worth.
package v1beta1
