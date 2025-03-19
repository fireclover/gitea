// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package services

import (
	"context"

	"code.gitea.io/gitea/models/db"

	"xorm.io/builder"
)

func init() {
	db.RegisterModel(new(ServiceProperty))
}

type PropertyType int64

const (
	// PropertyTypeVersion means the reference is a service version
	PropertyTypeVersion PropertyType = iota // 0
	// PropertyTypeFile means the reference is a service file
	PropertyTypeFile // 1
	// PropertyTypeService means the reference is a service
	PropertyTypeService // 2
)

// ServiceProperty represents a property of a service, version or file
type ServiceProperty struct {
	ID      int64        `xorm:"pk autoincr"`
	RefType PropertyType `xorm:"INDEX NOT NULL"`
	RefID   int64        `xorm:"INDEX NOT NULL"`
	Name    string       `xorm:"INDEX NOT NULL"`
	Value   string       `xorm:"TEXT NOT NULL"`
}

// InsertProperty creates a property
func InsertProperty(ctx context.Context, refType PropertyType, refID int64, name, value string) (*ServiceProperty, error) {
	pp := &ServiceProperty{
		RefType: refType,
		RefID:   refID,
		Name:    name,
		Value:   value,
	}

	_, err := db.GetEngine(ctx).Insert(pp)
	return pp, err
}

// GetProperties gets all properties
func GetProperties(ctx context.Context, refType PropertyType, refID int64) ([]*ServiceProperty, error) {
	pps := make([]*ServiceProperty, 0, 10)
	return pps, db.GetEngine(ctx).Where("ref_type = ? AND ref_id = ?", refType, refID).Find(&pps)
}

// GetPropertiesByName gets all properties with a specific name
func GetPropertiesByName(ctx context.Context, refType PropertyType, refID int64, name string) ([]*ServiceProperty, error) {
	pps := make([]*ServiceProperty, 0, 10)
	return pps, db.GetEngine(ctx).Where("ref_type = ? AND ref_id = ? AND name = ?", refType, refID, name).Find(&pps)
}

// UpdateProperty updates a property
func UpdateProperty(ctx context.Context, pp *ServiceProperty) error {
	_, err := db.GetEngine(ctx).ID(pp.ID).Update(pp)
	return err
}

// DeleteAllProperties deletes all properties of a ref
func DeleteAllProperties(ctx context.Context, refType PropertyType, refID int64) error {
	_, err := db.GetEngine(ctx).Where("ref_type = ? AND ref_id = ?", refType, refID).Delete(&ServiceProperty{})
	return err
}

// DeletePropertyByID deletes a property
func DeletePropertyByID(ctx context.Context, propertyID int64) error {
	_, err := db.GetEngine(ctx).ID(propertyID).Delete(&ServiceProperty{})
	return err
}

// DeletePropertyByName deletes properties by name
func DeletePropertyByName(ctx context.Context, refType PropertyType, refID int64, name string) error {
	_, err := db.GetEngine(ctx).Where("ref_type = ? AND ref_id = ? AND name = ?", refType, refID, name).Delete(&ServiceProperty{})
	return err
}

type DistinctPropertyDependency struct {
	Name  string
	Value string
}

// GetDistinctPropertyValues returns all distinct property values for a given type.
// Optional: Search only in dependence of another property.
func GetDistinctPropertyValues(ctx context.Context, serviceType Type, ownerID int64, refType PropertyType, propertyName string, dep *DistinctPropertyDependency) ([]string, error) {
	var cond builder.Cond = builder.Eq{
		"service_property.ref_type": refType,
		"service_property.name":     propertyName,
		"service.type":              serviceType,
		"service.owner_id":          ownerID,
	}
	if dep != nil {
		innerCond := builder.
			Expr("pp.ref_id = service_property.ref_id").
			And(builder.Eq{
				"pp.ref_type": refType,
				"pp.name":     dep.Name,
				"pp.value":    dep.Value,
			})
		cond = cond.And(builder.Exists(builder.Select("pp.ref_id").From("service_property pp").Where(innerCond)))
	}

	values := make([]string, 0, 5)
	return values, db.GetEngine(ctx).
		Table("service_property").
		Distinct("service_property.value").
		Join("INNER", "service_file", "service_file.id = service_property.ref_id").
		Join("INNER", "service_version", "service_version.id = service_file.version_id").
		Join("INNER", "service", "service.id = service_version.service_id").
		Where(cond).
		Find(&values)
}
