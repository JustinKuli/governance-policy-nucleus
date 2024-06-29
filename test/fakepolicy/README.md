# FakePolicy

This directory contains a controller for a FakePolicy CRD in the kubebuilder style (although with
some changed details), which is meant to be used for testing features in the nucleus. Because of its
purpose, the way it uses features is highly contrived and it might not always be a good example.

## Test Suites

The "basic" suite requires very specific namespaces and configmaps in the cluster, so it was
separated from the other tests, which may add namespaces and other resources.
