// Copyright Contributors to the Open Cluster Management project

package compliance

import (
	"context"
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	nucleusv1beta1 "open-cluster-management.io/governance-policy-nucleus/api/v1beta1"
)

// PrometheusEmitter is an emitter of compliance status via prometheus metrics.
type PrometheusEmitter struct {
	gaugeVec *prometheus.GaugeVec
}

// TODO: doc
func NewPrometheusEmitter(registry *prometheus.Registry) (PrometheusEmitter, error) {
	// TODO: maybe functional options like in https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
	e := PrometheusEmitter{
		gaugeVec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ocmio_policy_compliance",
				Help: "The compliance state of the open-cluster-management-io policy template. " +
					"-1 == NonCompliant, 0 == Unknown, 1 == NonCompliant",
			},
			[]string{
				"kind",
				"namespace",
				"name",
			},
		),
	}

	err := registry.Register(e.gaugeVec)
	if err != nil {
		alreadyRegistered := &prometheus.AlreadyRegisteredError{}

		if errors.As(err, alreadyRegistered) {
			// TODO: need to test if this is a pointer or not
			existingGauge, ok := alreadyRegistered.ExistingCollector.(prometheus.GaugeVec)
			if !ok {
				return e, fmt.Errorf("existing collector in registry was not a GaugeVev: %w", err)
			}

			e.gaugeVec = &existingGauge
		}

		return e, err
	}

	return e, nil
}

func (e PrometheusEmitter) Emit(_ context.Context, pol nucleusv1beta1.PolicyLike) error {
	gauge, err := e.gaugeVec.GetMetricWithLabelValues(
		pol.GetObjectKind().GroupVersionKind().Kind,
		pol.GetNamespace(),
		pol.GetName(),
	)

	if err != nil {
		return err
	}

	switch pol.ComplianceState() {
	case nucleusv1beta1.NonCompliant:
		gauge.Set(-1)
	case nucleusv1beta1.Compliant:
		gauge.Set(1)
	default:
		gauge.Set(0)
	}

	return nil
}
