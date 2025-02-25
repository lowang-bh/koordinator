/*
Copyright 2022 The Koordinator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sloconfig

import (
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"

	"github.com/koordinator-sh/koordinator/apis/configuration"
	slov1alpha1 "github.com/koordinator-sh/koordinator/apis/slo/v1alpha1"
	"github.com/koordinator-sh/koordinator/pkg/util"
)

func NewDefaultColocationCfg() *configuration.ColocationCfg {
	defaultCfg := DefaultColocationCfg()
	return &defaultCfg
}

func DefaultColocationCfg() configuration.ColocationCfg {
	return configuration.ColocationCfg{
		ColocationStrategy: DefaultColocationStrategy(),
	}
}

func DefaultColocationStrategy() configuration.ColocationStrategy {
	calculatePolicy := configuration.CalculateByPodUsage
	var defaultMemoryCollectPolicy slov1alpha1.NodeMemoryCollectPolicy = slov1alpha1.UsageWithoutPageCache
	cfg := configuration.ColocationStrategy{
		Enable:                         pointer.Bool(false),
		MetricAggregateDurationSeconds: pointer.Int64(300),
		MetricReportIntervalSeconds:    pointer.Int64(60),
		MetricAggregatePolicy: &slov1alpha1.AggregatePolicy{
			Durations: []metav1.Duration{
				{Duration: 5 * time.Minute},
				{Duration: 10 * time.Minute},
				{Duration: 30 * time.Minute},
			},
		},
		MetricMemoryCollectPolicy:     &defaultMemoryCollectPolicy,
		CPUReclaimThresholdPercent:    pointer.Int64(60),
		MemoryReclaimThresholdPercent: pointer.Int64(65),
		MemoryCalculatePolicy:         &calculatePolicy,
		DegradeTimeMinutes:            pointer.Int64(15),
		UpdateTimeThresholdSeconds:    pointer.Int64(300),
		ResourceDiffThreshold:         pointer.Float64(0.1),
	}
	cfg.ColocationStrategyExtender = defaultColocationStrategyExtender
	return cfg
}

func IsColocationStrategyValid(strategy *configuration.ColocationStrategy) bool {
	return strategy != nil &&
		(strategy.MetricAggregateDurationSeconds == nil || *strategy.MetricAggregateDurationSeconds > 0) &&
		(strategy.MetricReportIntervalSeconds == nil || *strategy.MetricReportIntervalSeconds > 0) &&
		(strategy.CPUReclaimThresholdPercent == nil || *strategy.CPUReclaimThresholdPercent > 0) &&
		(strategy.MemoryReclaimThresholdPercent == nil || *strategy.MemoryReclaimThresholdPercent > 0) &&
		(strategy.DegradeTimeMinutes == nil || *strategy.DegradeTimeMinutes > 0) &&
		(strategy.UpdateTimeThresholdSeconds == nil || *strategy.UpdateTimeThresholdSeconds > 0) &&
		(strategy.ResourceDiffThreshold == nil || *strategy.ResourceDiffThreshold > 0) &&
		(strategy.MetricMemoryCollectPolicy == nil || len(*strategy.MetricMemoryCollectPolicy) > 0)
}

func IsNodeColocationCfgValid(nodeCfg *configuration.NodeColocationCfg) bool {
	if nodeCfg == nil {
		return false
	}
	if nodeCfg.NodeSelector.MatchLabels == nil {
		return false
	}
	if _, err := metav1.LabelSelectorAsSelector(nodeCfg.NodeSelector); err != nil {
		return false
	}
	// node colocation should not be empty
	return !reflect.DeepEqual(&nodeCfg.ColocationStrategy, &configuration.ColocationStrategy{})
}

func GetNodeColocationStrategy(cfg *configuration.ColocationCfg, node *corev1.Node) *configuration.ColocationStrategy {
	if cfg == nil || node == nil {
		return nil
	}

	strategy := cfg.ColocationStrategy.DeepCopy()

	nodeLabels := labels.Set(node.Labels)
	for _, nodeCfg := range cfg.NodeConfigs {
		selector, err := metav1.LabelSelectorAsSelector(nodeCfg.NodeSelector)
		if err != nil || !selector.Matches(nodeLabels) {
			continue
		}

		merged, err := util.MergeCfg(strategy, &nodeCfg.ColocationStrategy)
		if err != nil {
			continue
		}

		strategy, _ = merged.(*configuration.ColocationStrategy)
		break
	}

	return strategy
}
