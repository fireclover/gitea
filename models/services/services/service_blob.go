// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package services

import (
	"context"
	"strconv"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/perm"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/util"

	"xorm.io/builder"
)

// ErrServiceBlobNotExist indicates a service blob not exist error
var ErrServiceBlobNotExist = util.NewNotExistErrorf("service blob does not exist")

func init() {
	db.RegisterModel(new(ServiceBlob))
}

// ServiceBlob represents a service blob
type ServiceBlob struct {
	ID          int64              `xorm:"pk autoincr"`
	Size        int64              `xorm:"NOT NULL DEFAULT 0"`
	HashMD5     string             `xorm:"hash_md5 char(32) UNIQUE(md5) INDEX NOT NULL"`
	HashSHA1    string             `xorm:"hash_sha1 char(40) UNIQUE(sha1) INDEX NOT NULL"`
	HashSHA256  string             `xorm:"hash_sha256 char(64) UNIQUE(sha256) INDEX NOT NULL"`
	HashSHA512  string             `xorm:"hash_sha512 char(128) UNIQUE(sha512) INDEX NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"created INDEX NOT NULL"`
}

// GetOrInsertBlob inserts a blob. If the blob exists already the existing blob is returned
func GetOrInsertBlob(ctx context.Context, pb *ServiceBlob) (*ServiceBlob, bool, error) {
	e := db.GetEngine(ctx)

	existing := &ServiceBlob{}

	has, err := e.Where(builder.Eq{
		"size":        pb.Size,
		"hash_md5":    pb.HashMD5,
		"hash_sha1":   pb.HashSHA1,
		"hash_sha256": pb.HashSHA256,
		"hash_sha512": pb.HashSHA512,
	}).Get(existing)
	if err != nil {
		return nil, false, err
	}
	if has {
		return existing, true, nil
	}
	if _, err = e.Insert(pb); err != nil {
		return nil, false, err
	}
	return pb, false, nil
}

// GetBlobByID gets a blob by id
func GetBlobByID(ctx context.Context, blobID int64) (*ServiceBlob, error) {
	pb := &ServiceBlob{}

	has, err := db.GetEngine(ctx).ID(blobID).Get(pb)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrServiceBlobNotExist
	}
	return pb, nil
}

// ExistServiceBlobWithSHA returns if a service blob exists with the provided sha
func ExistServiceBlobWithSHA(ctx context.Context, blobSha256 string) (bool, error) {
	return db.GetEngine(ctx).Exist(&ServiceBlob{
		HashSHA256: blobSha256,
	})
}

// FindExpiredUnreferencedBlobs gets all blobs without associated files older than the specific duration
func FindExpiredUnreferencedBlobs(ctx context.Context, olderThan time.Duration) ([]*ServiceBlob, error) {
	pbs := make([]*ServiceBlob, 0, 10)
	return pbs, db.GetEngine(ctx).
		Table("service_blob").
		Join("LEFT", "service_file", "service_file.blob_id = service_blob.id").
		Where("service_file.id IS NULL AND service_blob.created_unix < ?", time.Now().Add(-olderThan).Unix()).
		Find(&pbs)
}

// DeleteBlobByID deletes a blob by id
func DeleteBlobByID(ctx context.Context, blobID int64) error {
	_, err := db.GetEngine(ctx).ID(blobID).Delete(&ServiceBlob{})
	return err
}

// GetTotalBlobSize returns the total blobs size in bytes
func GetTotalBlobSize(ctx context.Context) (int64, error) {
	return db.GetEngine(ctx).
		SumInt(&ServiceBlob{}, "size")
}

// GetTotalUnreferencedBlobSize returns the total size of all unreferenced blobs in bytes
func GetTotalUnreferencedBlobSize(ctx context.Context) (int64, error) {
	return db.GetEngine(ctx).
		Table("service_blob").
		Join("LEFT", "service_file", "service_file.blob_id = service_blob.id").
		Where("service_file.id IS NULL").
		SumInt(&ServiceBlob{}, "size")
}

// IsBlobAccessibleForUser tests if the user has access to the blob
func IsBlobAccessibleForUser(ctx context.Context, blobID int64, user *user_model.User) (bool, error) {
	if user.IsAdmin {
		return true, nil
	}

	maxTeamAuthorize := builder.
		Select("max(team.authorize)").
		From("team").
		InnerJoin("team_user", "team_user.team_id = team.id").
		Where(builder.Eq{"team_user.uid": user.ID}.And(builder.Expr("team_user.org_id = `user`.id")))

	maxTeamUnitAccessMode := builder.
		Select("max(team_unit.access_mode)").
		From("team").
		InnerJoin("team_user", "team_user.team_id = team.id").
		InnerJoin("team_unit", "team_unit.team_id = team.id").
		Where(builder.Eq{"team_user.uid": user.ID, "team_unit.type": unit.TypeServices}.And(builder.Expr("team_user.org_id = `user`.id")))

	cond := builder.Eq{"service_blob.id": blobID}.And(
		// owner = user
		builder.Eq{"`user`.id": user.ID}.
			// user can see owner
			Or(builder.Eq{"`user`.visibility": structs.VisibleTypePublic}.Or(builder.Eq{"`user`.visibility": structs.VisibleTypeLimited})).
			// owner is an organization and user has access to it
			Or(builder.Eq{"`user`.type": user_model.UserTypeOrganization}.
				And(builder.Lte{strconv.Itoa(int(perm.AccessModeRead)): maxTeamAuthorize}.Or(builder.Lte{strconv.Itoa(int(perm.AccessModeRead)): maxTeamUnitAccessMode}))),
	)

	return db.GetEngine(ctx).
		Table("service_blob").
		Join("INNER", "service_file", "service_file.blob_id = service_blob.id").
		Join("INNER", "service_version", "service_version.id = service_file.version_id").
		Join("INNER", "service", "service.id = service_version.service_id").
		Join("INNER", "user", "`user`.id = service.owner_id").
		Where(cond).
		Exist(&ServiceBlob{})
}
