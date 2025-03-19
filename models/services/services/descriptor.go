// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/services/awslambda"
	"code.gitea.io/gitea/modules/services/azurefunction"
	"code.gitea.io/gitea/modules/util"

	"github.com/hashicorp/go-version"
)

// ServicePropertyList is a list of service properties
type ServicePropertyList []*ServiceProperty

// GetByName gets the first property value with the specific name
func (l ServicePropertyList) GetByName(name string) string {
	for _, pp := range l {
		if pp.Name == name {
			return pp.Value
		}
	}
	return ""
}

// ServiceDescriptor describes a service
type ServiceDescriptor struct {
	Service           *Service
	Owner             *user_model.User
	Repository        *repo_model.Repository
	Version           *ServiceVersion
	SemVer            *version.Version
	Creator           *user_model.User
	ServiceProperties ServicePropertyList
	VersionProperties ServicePropertyList
	Metadata          any
	Files             []*ServiceFileDescriptor
}

// ServiceFileDescriptor describes a service file
type ServiceFileDescriptor struct {
	File       *ServiceFile
	Blob       *ServiceBlob
	Properties ServicePropertyList
}

// ServiceWebLink returns the relative service web link
func (pd *ServiceDescriptor) ServiceWebLink() string {
	return fmt.Sprintf("%s/-/services/%s/%s", pd.Owner.HomeLink(), string(pd.Service.Type), url.PathEscape(pd.Service.LowerName))
}

// VersionWebLink returns the relative service version web link
func (pd *ServiceDescriptor) VersionWebLink() string {
	return fmt.Sprintf("%s/%s", pd.ServiceWebLink(), url.PathEscape(pd.Version.LowerVersion))
}

// ServiceHTMLURL returns the absolute service HTML URL
func (pd *ServiceDescriptor) ServiceHTMLURL() string {
	return fmt.Sprintf("%s/-/services/%s/%s", pd.Owner.HTMLURL(), string(pd.Service.Type), url.PathEscape(pd.Service.LowerName))
}

// VersionHTMLURL returns the absolute service version HTML URL
func (pd *ServiceDescriptor) VersionHTMLURL() string {
	return fmt.Sprintf("%s/%s", pd.ServiceHTMLURL(), url.PathEscape(pd.Version.LowerVersion))
}

// CalculateBlobSize returns the total blobs size in bytes
func (pd *ServiceDescriptor) CalculateBlobSize() int64 {
	size := int64(0)
	for _, f := range pd.Files {
		size += f.Blob.Size
	}
	return size
}

// GetServiceDescriptor gets the service description for a version
func GetServiceDescriptor(ctx context.Context, pv *ServiceVersion) (*ServiceDescriptor, error) {
	p, err := GetServiceByID(ctx, pv.ServiceID)
	if err != nil {
		return nil, err
	}
	o, err := user_model.GetUserByID(ctx, p.OwnerID)
	if err != nil {
		return nil, err
	}
	repository, err := repo_model.GetRepositoryByID(ctx, p.RepoID)
	if err != nil && !repo_model.IsErrRepoNotExist(err) {
		return nil, err
	}
	creator, err := user_model.GetUserByID(ctx, pv.CreatorID)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			creator = user_model.NewGhostUser()
		} else {
			return nil, err
		}
	}
	var semVer *version.Version
	if p.SemverCompatible {
		semVer, err = version.NewVersion(pv.Version)
		if err != nil {
			return nil, err
		}
	}
	pps, err := GetProperties(ctx, PropertyTypeService, p.ID)
	if err != nil {
		return nil, err
	}
	pvps, err := GetProperties(ctx, PropertyTypeVersion, pv.ID)
	if err != nil {
		return nil, err
	}
	pfs, err := GetFilesByVersionID(ctx, pv.ID)
	if err != nil {
		return nil, err
	}

	pfds, err := GetServiceFileDescriptors(ctx, pfs)
	if err != nil {
		return nil, err
	}

	var metadata any
	switch p.Type {
	case TypeAwsLambda:
		metadata = &awslambda.VersionMetadata{}
	case TypeAzureFunction:
		metadata = &azurefunction.VersionMetadata{}
	default:
		panic(fmt.Sprintf("unknown service type: %s", string(p.Type)))
	}
	if metadata != nil {
		if err := json.Unmarshal([]byte(pv.MetadataJSON), &metadata); err != nil {
			return nil, err
		}
	}

	return &ServiceDescriptor{
		Service:           p,
		Owner:             o,
		Repository:        repository,
		Version:           pv,
		SemVer:            semVer,
		Creator:           creator,
		ServiceProperties: ServicePropertyList(pps),
		VersionProperties: ServicePropertyList(pvps),
		Metadata:          metadata,
		Files:             pfds,
	}, nil
}

// GetServiceFileDescriptor gets a service file descriptor for a service file
func GetServiceFileDescriptor(ctx context.Context, pf *ServiceFile) (*ServiceFileDescriptor, error) {
	pb, err := GetBlobByID(ctx, pf.BlobID)
	if err != nil {
		return nil, err
	}
	pfps, err := GetProperties(ctx, PropertyTypeFile, pf.ID)
	if err != nil {
		return nil, err
	}
	return &ServiceFileDescriptor{
		pf,
		pb,
		ServicePropertyList(pfps),
	}, nil
}

// GetServiceFileDescriptors gets the service file descriptors for the service files
func GetServiceFileDescriptors(ctx context.Context, pfs []*ServiceFile) ([]*ServiceFileDescriptor, error) {
	pfds := make([]*ServiceFileDescriptor, 0, len(pfs))
	for _, pf := range pfs {
		pfd, err := GetServiceFileDescriptor(ctx, pf)
		if err != nil {
			return nil, err
		}
		pfds = append(pfds, pfd)
	}
	return pfds, nil
}

// GetServiceDescriptors gets the service descriptions for the versions
func GetServiceDescriptors(ctx context.Context, pvs []*ServiceVersion) ([]*ServiceDescriptor, error) {
	pds := make([]*ServiceDescriptor, 0, len(pvs))
	for _, pv := range pvs {
		pd, err := GetServiceDescriptor(ctx, pv)
		if err != nil {
			return nil, err
		}
		pds = append(pds, pd)
	}
	return pds, nil
}
