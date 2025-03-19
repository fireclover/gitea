// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package services

import (
	"context"
	"fmt"
	"regexp"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/util"

	"xorm.io/builder"
)

var ErrServiceCleanupRuleNotExist = util.NewNotExistErrorf("service blob does not exist")

func init() {
	db.RegisterModel(new(ServiceCleanupRule))
}

// ServiceCleanupRule represents a rule which describes when to clean up service versions
type ServiceCleanupRule struct {
	ID                   int64              `xorm:"pk autoincr"`
	Enabled              bool               `xorm:"INDEX NOT NULL DEFAULT false"`
	OwnerID              int64              `xorm:"UNIQUE(s) INDEX NOT NULL DEFAULT 0"`
	Type                 Type               `xorm:"UNIQUE(s) INDEX NOT NULL"`
	KeepCount            int                `xorm:"NOT NULL DEFAULT 0"`
	KeepPattern          string             `xorm:"NOT NULL DEFAULT ''"`
	KeepPatternMatcher   *regexp.Regexp     `xorm:"-"`
	RemoveDays           int                `xorm:"NOT NULL DEFAULT 0"`
	RemovePattern        string             `xorm:"NOT NULL DEFAULT ''"`
	RemovePatternMatcher *regexp.Regexp     `xorm:"-"`
	MatchFullName        bool               `xorm:"NOT NULL DEFAULT false"`
	CreatedUnix          timeutil.TimeStamp `xorm:"created NOT NULL DEFAULT 0"`
	UpdatedUnix          timeutil.TimeStamp `xorm:"updated NOT NULL DEFAULT 0"`
}

func (pcr *ServiceCleanupRule) CompiledPattern() error {
	if pcr.KeepPatternMatcher != nil || pcr.RemovePatternMatcher != nil {
		return nil
	}

	if pcr.KeepPattern != "" {
		var err error
		pcr.KeepPatternMatcher, err = regexp.Compile(fmt.Sprintf(`(?i)\A%s\z`, pcr.KeepPattern))
		if err != nil {
			return err
		}
	}

	if pcr.RemovePattern != "" {
		var err error
		pcr.RemovePatternMatcher, err = regexp.Compile(fmt.Sprintf(`(?i)\A%s\z`, pcr.RemovePattern))
		if err != nil {
			return err
		}
	}

	return nil
}

func InsertCleanupRule(ctx context.Context, pcr *ServiceCleanupRule) (*ServiceCleanupRule, error) {
	return pcr, db.Insert(ctx, pcr)
}

func GetCleanupRuleByID(ctx context.Context, id int64) (*ServiceCleanupRule, error) {
	pcr := &ServiceCleanupRule{}

	has, err := db.GetEngine(ctx).ID(id).Get(pcr)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrServiceCleanupRuleNotExist
	}
	return pcr, nil
}

func UpdateCleanupRule(ctx context.Context, pcr *ServiceCleanupRule) error {
	_, err := db.GetEngine(ctx).ID(pcr.ID).AllCols().Update(pcr)
	return err
}

func GetCleanupRulesByOwner(ctx context.Context, ownerID int64) ([]*ServiceCleanupRule, error) {
	pcrs := make([]*ServiceCleanupRule, 0, 10)
	return pcrs, db.GetEngine(ctx).Where("owner_id = ?", ownerID).Find(&pcrs)
}

func DeleteCleanupRuleByID(ctx context.Context, ruleID int64) error {
	_, err := db.GetEngine(ctx).ID(ruleID).Delete(&ServiceCleanupRule{})
	return err
}

func HasOwnerCleanupRuleForServiceType(ctx context.Context, ownerID int64, serviceType Type) (bool, error) {
	return db.GetEngine(ctx).
		Where("owner_id = ? AND type = ?", ownerID, serviceType).
		Exist(&ServiceCleanupRule{})
}

func IterateEnabledCleanupRules(ctx context.Context, callback func(context.Context, *ServiceCleanupRule) error) error {
	return db.Iterate(
		ctx,
		builder.Eq{"enabled": true},
		callback,
	)
}
