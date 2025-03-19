// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package services

import (
	"context"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/util"

	"xorm.io/builder"
)

func init() {
	db.RegisterModel(new(ServiceFile))
}

var (
	// ErrDuplicateServiceFile indicates a duplicated service file error
	ErrDuplicateServiceFile = util.NewAlreadyExistErrorf("service file already exists")
	// ErrServiceFileNotExist indicates a service file not exist error
	ErrServiceFileNotExist = util.NewNotExistErrorf("service file does not exist")
)

// EmptyFileKey is a named constant for an empty file key
const EmptyFileKey = ""

// ServiceFile represents a service file
type ServiceFile struct {
	ID           int64              `xorm:"pk autoincr"`
	VersionID    int64              `xorm:"UNIQUE(s) INDEX NOT NULL"`
	BlobID       int64              `xorm:"INDEX NOT NULL"`
	Name         string             `xorm:"NOT NULL"`
	LowerName    string             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CompositeKey string             `xorm:"UNIQUE(s) INDEX"`
	IsLead       bool               `xorm:"NOT NULL DEFAULT false"`
	CreatedUnix  timeutil.TimeStamp `xorm:"created INDEX NOT NULL"`
}

// TryInsertFile inserts a file. If the file exists already ErrDuplicateServiceFile is returned
func TryInsertFile(ctx context.Context, pf *ServiceFile) (*ServiceFile, error) {
	e := db.GetEngine(ctx)

	existing := &ServiceFile{}

	has, err := e.Where(builder.Eq{
		"version_id":    pf.VersionID,
		"lower_name":    pf.LowerName,
		"composite_key": pf.CompositeKey,
	}).Get(existing)
	if err != nil {
		return nil, err
	}
	if has {
		return existing, ErrDuplicateServiceFile
	}
	if _, err = e.Insert(pf); err != nil {
		return nil, err
	}
	return pf, nil
}

// GetFilesByVersionID gets all files of a version
func GetFilesByVersionID(ctx context.Context, versionID int64) ([]*ServiceFile, error) {
	pfs := make([]*ServiceFile, 0, 10)
	return pfs, db.GetEngine(ctx).Where("version_id = ?", versionID).Find(&pfs)
}

// GetFileForVersionByID gets a file of a version by id
func GetFileForVersionByID(ctx context.Context, versionID, fileID int64) (*ServiceFile, error) {
	pf := &ServiceFile{
		VersionID: versionID,
	}

	has, err := db.GetEngine(ctx).ID(fileID).Get(pf)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrServiceFileNotExist
	}
	return pf, nil
}

// GetFileForVersionByName gets a file of a version by name
func GetFileForVersionByName(ctx context.Context, versionID int64, name, key string) (*ServiceFile, error) {
	if name == "" {
		return nil, ErrServiceFileNotExist
	}

	pf := &ServiceFile{}

	has, err := db.GetEngine(ctx).Where(builder.Eq{
		"version_id":    versionID,
		"lower_name":    strings.ToLower(name),
		"composite_key": key,
	}).Get(pf)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrServiceFileNotExist
	}
	return pf, nil
}

// DeleteFileByID deletes a file
func DeleteFileByID(ctx context.Context, fileID int64) error {
	_, err := db.GetEngine(ctx).ID(fileID).Delete(&ServiceFile{})
	return err
}

// ServiceFileSearchOptions are options for SearchXXX methods
type ServiceFileSearchOptions struct {
	OwnerID       int64
	ServiceType   Type
	VersionID     int64
	Query         string
	CompositeKey  string
	Properties    map[string]string
	OlderThan     time.Duration
	HashAlgorithm string
	Hash          string
	db.Paginator
}

func (opts *ServiceFileSearchOptions) toConds() builder.Cond {
	cond := builder.NewCond()

	if opts.VersionID != 0 {
		cond = cond.And(builder.Eq{"service_file.version_id": opts.VersionID})
	} else if opts.OwnerID != 0 || (opts.ServiceType != "" && opts.ServiceType != "all") {
		var versionCond builder.Cond = builder.Eq{
			"service_version.is_internal": false,
		}
		if opts.OwnerID != 0 {
			versionCond = versionCond.And(builder.Eq{"service.owner_id": opts.OwnerID})
		}
		if opts.ServiceType != "" && opts.ServiceType != "all" {
			versionCond = versionCond.And(builder.Eq{"service.type": opts.ServiceType})
		}

		in := builder.
			Select("service_version.id").
			From("service_version").
			InnerJoin("service", "service.id = service_version.service_id").
			Where(versionCond)

		cond = cond.And(builder.In("service_file.version_id", in))
	}
	if opts.CompositeKey != "" {
		cond = cond.And(builder.Eq{"service_file.composite_key": opts.CompositeKey})
	}
	if opts.Query != "" {
		cond = cond.And(builder.Like{"service_file.lower_name", strings.ToLower(opts.Query)})
	}

	if len(opts.Properties) != 0 {
		var propsCond builder.Cond = builder.Eq{
			"service_property.ref_type": PropertyTypeFile,
		}
		propsCond = propsCond.And(builder.Expr("service_property.ref_id = service_file.id"))

		propsCondBlock := builder.NewCond()
		for name, value := range opts.Properties {
			propsCondBlock = propsCondBlock.Or(builder.Eq{
				"service_property.name":  name,
				"service_property.value": value,
			})
		}
		propsCond = propsCond.And(propsCondBlock)

		cond = cond.And(builder.Eq{
			strconv.Itoa(len(opts.Properties)): builder.Select("COUNT(*)").Where(propsCond).From("service_property"),
		})
	}

	if opts.OlderThan != 0 {
		cond = cond.And(builder.Lt{"service_file.created_unix": time.Now().Add(-opts.OlderThan).Unix()})
	}

	if opts.Hash != "" {
		var field string
		switch strings.ToLower(opts.HashAlgorithm) {
		case "md5":
			field = "service_blob.hash_md5"
		case "sha1":
			field = "service_blob.hash_sha1"
		case "sha256":
			field = "service_blob.hash_sha256"
		case "sha512":
			fallthrough
		default: // default to SHA512 if not specified or unknown
			field = "service_blob.hash_sha512"
		}
		innerCond := builder.
			Expr("service_blob.id = service_file.blob_id").
			And(builder.Eq{field: opts.Hash})
		cond = cond.And(builder.Exists(builder.Select("service_blob.id").From("service_blob").Where(innerCond)))
	}

	return cond
}

// SearchFiles gets all files of services matching the search options
func SearchFiles(ctx context.Context, opts *ServiceFileSearchOptions) ([]*ServiceFile, int64, error) {
	sess := db.GetEngine(ctx).
		Where(opts.toConds())

	if opts.Paginator != nil {
		sess = db.SetSessionPagination(sess, opts)
	}

	pfs := make([]*ServiceFile, 0, 10)
	count, err := sess.FindAndCount(&pfs)
	return pfs, count, err
}

// HasFiles tests if there are files of services matching the search options
func HasFiles(ctx context.Context, opts *ServiceFileSearchOptions) (bool, error) {
	return db.Exist[ServiceFile](ctx, opts.toConds())
}

// CalculateFileSize sums up all blob sizes matching the search options.
// It does NOT respect the deduplication of blobs.
func CalculateFileSize(ctx context.Context, opts *ServiceFileSearchOptions) (int64, error) {
	return db.GetEngine(ctx).
		Table("service_file").
		Where(opts.toConds()).
		Join("INNER", "service_blob", "service_blob.id = service_file.blob_id").
		SumInt(new(ServiceBlob), "size")
}
