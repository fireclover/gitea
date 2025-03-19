// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package services_test

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	services_model "code.gitea.io/gitea/models/services"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	_ "code.gitea.io/gitea/models"
	_ "code.gitea.io/gitea/models/actions"
	_ "code.gitea.io/gitea/models/activities"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestHasOwnerServices(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	p, err := services_model.TryInsertService(db.DefaultContext, &services_model.Service{
		OwnerID:   owner.ID,
		LowerName: "service",
	})
	assert.NotNil(t, p)
	assert.NoError(t, err)

	// A service without service versions gets automatically cleaned up and should return false
	has, err := services_model.HasOwnerServices(db.DefaultContext, owner.ID)
	assert.False(t, has)
	assert.NoError(t, err)

	pv, err := services_model.GetOrInsertVersion(db.DefaultContext, &services_model.ServiceVersion{
		ServiceID:    p.ID,
		LowerVersion: "internal",
		IsInternal:   true,
	})
	assert.NotNil(t, pv)
	assert.NoError(t, err)

	// A service with an internal service version gets automatically cleaned up and should return false
	has, err = services_model.HasOwnerServices(db.DefaultContext, owner.ID)
	assert.False(t, has)
	assert.NoError(t, err)

	pv, err = services_model.GetOrInsertVersion(db.DefaultContext, &services_model.ServiceVersion{
		ServiceID:    p.ID,
		LowerVersion: "normal",
		IsInternal:   false,
	})
	assert.NotNil(t, pv)
	assert.NoError(t, err)

	// A service with a normal service version should return true
	has, err = services_model.HasOwnerServices(db.DefaultContext, owner.ID)
	assert.True(t, has)
	assert.NoError(t, err)
}
