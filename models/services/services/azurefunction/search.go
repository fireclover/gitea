// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package azurefunction

import (
	"context"

	services_model "code.gitea.io/gitea/models/services"
	azurefunction_module "code.gitea.io/gitea/modules/services/azurefunction"
)

// GetBranches gets all available branches
func GetBranches(ctx context.Context, ownerID int64) ([]string, error) {
	return services_model.GetDistinctPropertyValues(
		ctx,
		services_model.TypeAzureFunction,
		ownerID,
		services_model.PropertyTypeFile,
		azurefunction_module.PropertyBranch,
		nil,
	)
}

// GetRepositories gets all available repositories for the given branch
func GetRepositories(ctx context.Context, ownerID int64, branch string) ([]string, error) {
	return services_model.GetDistinctPropertyValues(
		ctx,
		services_model.TypeAzureFunction,
		ownerID,
		services_model.PropertyTypeFile,
		azurefunction_module.PropertyRepository,
		&services_model.DistinctPropertyDependency{
			Name:  azurefunction_module.PropertyBranch,
			Value: branch,
		},
	)
}

// GetArchitectures gets all available architectures for the given repository
func GetArchitectures(ctx context.Context, ownerID int64, repository string) ([]string, error) {
	return services_model.GetDistinctPropertyValues(
		ctx,
		services_model.TypeAzureFunction,
		ownerID,
		services_model.PropertyTypeFile,
		azurefunction_module.PropertyArchitecture,
		&services_model.DistinctPropertyDependency{
			Name:  azurefunction_module.PropertyRepository,
			Value: repository,
		},
	)
}
