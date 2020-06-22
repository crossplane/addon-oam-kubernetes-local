/*
Copyright 2019 The Crossplane Authors.

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

package core

import (
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/addon-oam-kubernetes-local/pkg/controller/core/scopes/healthscope"
	"github.com/crossplane/addon-oam-kubernetes-local/pkg/controller/core/traits/manualscalertrait"
	"github.com/crossplane/addon-oam-kubernetes-local/pkg/controller/core/workloads/containerizedworkload"
)

// Setup  controllers.
func Setup(mgr ctrl.Manager, l logging.Logger) error {
	for _, setup := range []func(ctrl.Manager, logging.Logger) error{
		containerizedworkload.Setup, manualscalertrait.Setup, healthscope.Setup,
	} {
		if err := setup(mgr, l); err != nil {
			return err
		}
	}
	return nil
}
