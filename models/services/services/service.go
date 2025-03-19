// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package services

import (
	"context"
	"fmt"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/util"

	"xorm.io/builder"
)

func init() {
	db.RegisterModel(new(Service))
}

var (
	// ErrDuplicateService indicates a duplicated service error
	ErrDuplicateService = util.NewAlreadyExistErrorf("service already exists")
	// ErrServiceNotExist indicates a service not exist error
	ErrServiceNotExist = util.NewNotExistErrorf("service does not exist")
)

// Type of a service
type Type string

// List of supported services
const (
	TypeAwsLambda           Type = "awslambda"
	TypeAzureFunction       Type = "azurefunction"
)

var TypeList = []Type{
	TypeAwsLambda,
	TypeAzureFunction,
}

// Name gets the name of the service type
func (pt Type) Name() string {
	switch pt {
	case TypeAwsLambda:
		return "AwsLambda"
	case TypeAzureFunction:
		return "AzureFunction"
	}
	panic(fmt.Sprintf("unknown service type: %s", string(pt)))
}

// SVGName gets the name of the service type svg image
func (pt Type) SVGName() string {
	switch pt {
	case TypeAwsLambda:
		return "gitea-alpine"
	case TypeAzureFunction:
		return "gitea-cargo"
	}
	panic(fmt.Sprintf("unknown service type: %s", string(pt)))
}

// Service represents a service
type Service struct {
	ID               int64  `xorm:"pk autoincr"`
	OwnerID          int64  `xorm:"UNIQUE(s) INDEX NOT NULL"`
	RepoID           int64  `xorm:"INDEX"`
	Type             Type   `xorm:"UNIQUE(s) INDEX NOT NULL"`
	Name             string `xorm:"NOT NULL"`
	LowerName        string `xorm:"UNIQUE(s) INDEX NOT NULL"`
	SemverCompatible bool   `xorm:"NOT NULL DEFAULT false"`
	IsInternal       bool   `xorm:"NOT NULL DEFAULT false"`
}

// TryInsertService inserts a service. If a service exists already, ErrDuplicateService is returned
func TryInsertService(ctx context.Context, p *Service) (*Service, error) {
	e := db.GetEngine(ctx)

	existing := &Service{}

	has, err := e.Where(builder.Eq{
		"owner_id":   p.OwnerID,
		"type":       p.Type,
		"lower_name": p.LowerName,
	}).Get(existing)
	if err != nil {
		return nil, err
	}
	if has {
		return existing, ErrDuplicateService
	}
	if _, err = e.Insert(p); err != nil {
		return nil, err
	}
	return p, nil
}

// DeleteServiceByID deletes a service by id
func DeleteServiceByID(ctx context.Context, serviceID int64) error {
	_, err := db.GetEngine(ctx).ID(serviceID).Delete(&Service{})
	return err
}

// SetRepositoryLink sets the linked repository
func SetRepositoryLink(ctx context.Context, serviceID, repoID int64) error {
	_, err := db.GetEngine(ctx).ID(serviceID).Cols("repo_id").Update(&Service{RepoID: repoID})
	return err
}

// UnlinkRepositoryFromAllServices unlinks every service from the repository
func UnlinkRepositoryFromAllServices(ctx context.Context, repoID int64) error {
	_, err := db.GetEngine(ctx).Where("repo_id = ?", repoID).Cols("repo_id").Update(&Service{})
	return err
}

// GetServiceByID gets a service by id
func GetServiceByID(ctx context.Context, serviceID int64) (*Service, error) {
	p := &Service{}

	has, err := db.GetEngine(ctx).ID(serviceID).Get(p)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrServiceNotExist
	}
	return p, nil
}

// UpdateServiceNameByID updates the service's name, it is only for internal usage, for example: rename some legacy services
func UpdateServiceNameByID(ctx context.Context, ownerID int64, serviceType Type, serviceID int64, name string) error {
	var cond builder.Cond = builder.Eq{
		"service.id":          serviceID,
		"service.owner_id":    ownerID,
		"service.type":        serviceType,
		"service.is_internal": false,
	}
	_, err := db.GetEngine(ctx).Where(cond).Update(&Service{Name: name, LowerName: strings.ToLower(name)})
	return err
}

// GetServiceByName gets a service by name
func GetServiceByName(ctx context.Context, ownerID int64, serviceType Type, name string) (*Service, error) {
	var cond builder.Cond = builder.Eq{
		"service.owner_id":    ownerID,
		"service.type":        serviceType,
		"service.lower_name":  strings.ToLower(name),
		"service.is_internal": false,
	}

	p := &Service{}

	has, err := db.GetEngine(ctx).
		Where(cond).
		Get(p)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrServiceNotExist
	}
	return p, nil
}

// GetServicesByType gets all services of a specific type
func GetServicesByType(ctx context.Context, ownerID int64, serviceType Type) ([]*Service, error) {
	var cond builder.Cond = builder.Eq{
		"service.owner_id":    ownerID,
		"service.type":        serviceType,
		"service.is_internal": false,
	}

	ps := make([]*Service, 0, 10)
	return ps, db.GetEngine(ctx).
		Where(cond).
		Find(&ps)
}

// FindUnreferencedServices gets all services without associated versions
func FindUnreferencedServices(ctx context.Context) ([]*Service, error) {
	in := builder.
		Select("service.id").
		From("service").
		LeftJoin("service_version", "service_version.service_id = service.id").
		Where(builder.Expr("service_version.id IS NULL"))

	ps := make([]*Service, 0, 10)
	return ps, db.GetEngine(ctx).
		// double select workaround for MySQL
		// https://stackoverflow.com/questions/4471277/mysql-delete-from-with-subquery-as-condition
		Where(builder.In("service.id", builder.Select("id").From(in, "temp"))).
		Find(&ps)
}

// ErrUserOwnServices notifies that the user (still) owns the services.
type ErrUserOwnServices struct {
	UID int64
}

// IsErrUserOwnServices checks if an error is an ErrUserOwnServices.
func IsErrUserOwnServices(err error) bool {
	_, ok := err.(ErrUserOwnServices)
	return ok
}

func (err ErrUserOwnServices) Error() string {
	return fmt.Sprintf("user still has ownership of services [uid: %d]", err.UID)
}

// HasOwnerServices tests if a user/org has accessible services
func HasOwnerServices(ctx context.Context, ownerID int64) (bool, error) {
	return db.GetEngine(ctx).
		Table("service_version").
		Join("INNER", "service", "service.id = service_version.service_id").
		Where(builder.Eq{
			"service_version.is_internal": false,
			"service.owner_id":            ownerID,
		}).
		Exist(&ServiceVersion{})
}

// HasRepositoryServices tests if a repository has services
func HasRepositoryServices(ctx context.Context, repositoryID int64) (bool, error) {
	return db.GetEngine(ctx).Where("repo_id = ?", repositoryID).Exist(&Service{})
}
