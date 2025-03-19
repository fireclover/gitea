// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package services

import (
	"context"
	"strings"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/util"
)

// ErrServiceBlobUploadNotExist indicates a service blob upload not exist error
var ErrServiceBlobUploadNotExist = util.NewNotExistErrorf("service blob upload does not exist")

func init() {
	db.RegisterModel(new(ServiceBlobUpload))
}

// ServiceBlobUpload represents a service blob upload
type ServiceBlobUpload struct {
	ID             string             `xorm:"pk"`
	BytesReceived  int64              `xorm:"NOT NULL DEFAULT 0"`
	HashStateBytes []byte             `xorm:"BLOB"`
	CreatedUnix    timeutil.TimeStamp `xorm:"created NOT NULL"`
	UpdatedUnix    timeutil.TimeStamp `xorm:"updated INDEX NOT NULL"`
}

// CreateBlobUpload inserts a blob upload
func CreateBlobUpload(ctx context.Context) (*ServiceBlobUpload, error) {
	id, err := util.CryptoRandomString(25)
	if err != nil {
		return nil, err
	}

	pbu := &ServiceBlobUpload{
		ID: strings.ToLower(id),
	}

	_, err = db.GetEngine(ctx).Insert(pbu)
	return pbu, err
}

// GetBlobUploadByID gets a blob upload by id
func GetBlobUploadByID(ctx context.Context, id string) (*ServiceBlobUpload, error) {
	pbu := &ServiceBlobUpload{}

	has, err := db.GetEngine(ctx).ID(id).Get(pbu)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrServiceBlobUploadNotExist
	}
	return pbu, nil
}

// UpdateBlobUpload updates the blob upload
func UpdateBlobUpload(ctx context.Context, pbu *ServiceBlobUpload) error {
	_, err := db.GetEngine(ctx).ID(pbu.ID).Update(pbu)
	return err
}

// DeleteBlobUploadByID deletes the blob upload
func DeleteBlobUploadByID(ctx context.Context, id string) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(&ServiceBlobUpload{})
	return err
}

// FindExpiredBlobUploads gets all expired blob uploads
func FindExpiredBlobUploads(ctx context.Context, olderThan time.Duration) ([]*ServiceBlobUpload, error) {
	pbus := make([]*ServiceBlobUpload, 0, 10)
	return pbus, db.GetEngine(ctx).
		Where("updated_unix < ?", time.Now().Add(-olderThan).Unix()).
		Find(&pbus)
}
