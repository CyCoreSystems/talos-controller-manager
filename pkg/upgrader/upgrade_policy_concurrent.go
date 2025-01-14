// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package upgrader

import (
	"log"

	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
)

type ConcurrentPolicy struct {
	Upgrader

	Concurrency int
}

type Job struct {
	node    corev1.Node
	version string
}

type Result struct {
	job Job
	err error
}

func (policy ConcurrentPolicy) Run(nodes corev1.NodeList, version string) error {
	jobs := make(chan Job, policy.Concurrency)
	results := make(chan Result, len(nodes.Items))

	for w := 0; w < policy.Concurrency; w++ {
		go policy.worker(w, jobs, results)
	}

	for _, node := range nodes.Items {
		jobs <- Job{node, version}
	}

	close(jobs)

	var result *multierror.Error

	for a := 0; a < len(nodes.Items); a++ {
		r := <-results
		if r.err != nil {
			result = multierror.Append(result, r.err)
		}
	}

	return result.ErrorOrNil()
}

func (policy ConcurrentPolicy) worker(id int, jobs <-chan Job, results chan<- Result) {
	for j := range jobs {
		log.Println("concurrent policy worker", id, "assigned to node", j.node.Name)

		if err := policy.Upgrade(j.node, j.version); err != nil {
			results <- Result{j, err}
		}

		results <- Result{j, nil}
	}
}
