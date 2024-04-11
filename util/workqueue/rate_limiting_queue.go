/*
Copyright 2016 The Kubernetes Authors.

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

package workqueue

import "k8s.io/utils/clock"

// RateLimitingInterface is an interface that rate limits items being added to the queue.
type RateLimitingInterface[T comparable] interface {
	DelayingInterface[T]

	// AddRateLimited adds an item to the workqueue after the rate limiter says it's ok
	AddRateLimited(item T)

	// Forget indicates that an item is finished being retried.  Doesn't matter whether it's for perm failing
	// or for success, we'll stop the rate limiter from tracking it.  This only clears the `rateLimiter`, you
	// still have to call `Done` on the queue.
	Forget(item T)

	// NumRequeues returns back how many times the item was requeued
	NumRequeues(item T) int
}

// RateLimitingQueueConfig specifies optional configurations to customize a RateLimitingInterface.

type RateLimitingQueueConfig[T comparable] struct {
	// Name for the queue. If unnamed, the metrics will not be registered.
	Name string

	// MetricsProvider optionally allows specifying a metrics provider to use for the queue
	// instead of the global provider.
	MetricsProvider MetricsProvider

	// Clock optionally allows injecting a real or fake clock for testing purposes.
	Clock clock.WithTicker

	// DelayingQueue optionally allows injecting custom delaying queue DelayingInterface instead of the default one.
	DelayingQueue DelayingInterface[T]
}

// NewRateLimitingQueue constructs a new workqueue with rateLimited queuing ability
// Remember to call Forget!  If you don't, you may end up tracking failures forever.
// NewRateLimitingQueue does not emit metrics. For use with a MetricsProvider, please use
// NewRateLimitingQueueWithConfig instead and specify a name.
func NewRateLimitingQueue[T comparable](rateLimiter RateLimiter[T]) RateLimitingInterface[T] {
	return NewRateLimitingQueueWithConfig(rateLimiter, RateLimitingQueueConfig[T]{})
}

// NewRateLimitingQueueWithConfig constructs a new workqueue with rateLimited queuing ability
// with options to customize different properties.
// Remember to call Forget!  If you don't, you may end up tracking failures forever.
func NewRateLimitingQueueWithConfig[T comparable](rateLimiter RateLimiter[T], config RateLimitingQueueConfig[T]) RateLimitingInterface[T] {
	if config.Clock == nil {
		config.Clock = clock.RealClock{}
	}

	if config.DelayingQueue == nil {
		config.DelayingQueue = NewDelayingQueueWithConfig(DelayingQueueConfig[T]{
			Name:            config.Name,
			MetricsProvider: config.MetricsProvider,
			Clock:           config.Clock,
		})
	}

	return &rateLimitingType[T]{
		DelayingInterface: config.DelayingQueue,
		rateLimiter:       rateLimiter,
	}
}

// NewNamedRateLimitingQueue constructs a new named workqueue with rateLimited queuing ability.
// Deprecated: Use NewRateLimitingQueueWithConfig instead.
func NewNamedRateLimitingQueue[T comparable](rateLimiter RateLimiter[T], name string) RateLimitingInterface[T] {
	return NewRateLimitingQueueWithConfig(rateLimiter, RateLimitingQueueConfig[T]{
		Name: name,
	})
}

// NewRateLimitingQueueWithDelayingInterface constructs a new named workqueue with rateLimited queuing ability
// with the option to inject a custom delaying queue instead of the default one.
// Deprecated: Use NewRateLimitingQueueWithConfig instead.
func NewRateLimitingQueueWithDelayingInterface[T comparable](di DelayingInterface[T], rateLimiter RateLimiter[T]) RateLimitingInterface[T] {
	return NewRateLimitingQueueWithConfig(rateLimiter, RateLimitingQueueConfig[T]{
		DelayingQueue: di,
	})
}

// rateLimitingType wraps an Interface and provides rateLimited re-enquing
type rateLimitingType[T comparable] struct {
	DelayingInterface[T]

	rateLimiter RateLimiter[T]
}

// AddRateLimited AddAfter's the item based on the time when the rate limiter says it's ok
func (q *rateLimitingType[T]) AddRateLimited(item T) {
	q.DelayingInterface.AddAfter(item, q.rateLimiter.When(item))
}

func (q *rateLimitingType[T]) NumRequeues(item T) int {
	return q.rateLimiter.NumRequeues(item)
}

func (q *rateLimitingType[T]) Forget(item T) {
	q.rateLimiter.Forget(item)
}
