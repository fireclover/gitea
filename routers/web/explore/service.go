// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package explore

import (
	"net/http"

	services_model "code.gitea.io/gitea/models/services"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/templates"
	"code.gitea.io/gitea/services/context"
)

const (
	// tplExploreServices explore services page template
	tplExploreServices        templates.TplName = "explore/services"
	relevantServicesOnlyParam string            = "only_show_relevant"
)

// ServiceSearchOptions when calling search services
type ServiceSearchOptions struct {
	OwnerID          int64
	Private          bool
	Restricted       bool
	PageSize         int
	OnlyShowRelevant bool
	TplName          templates.TplName
}

// RenderServiceSearch render services search page
// This function is also used to render the Admin Servicesitory Management page.
func RenderServiceSearch(ctx *context.Context, opts *ServiceSearchOptions) {
	// Sitemap index for sitemap paths
	page := int(ctx.PathParamInt64("idx"))
	isSitemap := ctx.PathParam("idx") != ""
	if page <= 1 {
		page = ctx.FormInt("page")
	}

	if page <= 0 {
		page = 1
	}

	if isSitemap {
		opts.PageSize = setting.UI.SitemapPagingNum
	}

	var (
		services   []*services_model.Service
		count   int64
		err     error
	)

	sortOrder := ctx.FormString("sort")
	if sortOrder == "" {
		sortOrder = setting.UI.ExploreDefaultSort
	}
	sortOrder = "recentupdate"
	// if order, ok := services_model.OrderByFlatMap[sortOrder]; ok {
	// 	orderBy = order
	// } else {
	// 	sortOrder = "recentupdate"
	// 	orderBy = db.SearchOrderByRecentUpdated
	// }
	ctx.Data["SortType"] = sortOrder

	keyword := ctx.FormTrim("q")

	ctx.Data["OnlyShowRelevant"] = opts.OnlyShowRelevant

	topicOnly := ctx.FormBool("topic")
	ctx.Data["TopicOnly"] = topicOnly

	language := ctx.FormTrim("language")
	ctx.Data["Language"] = language

	archived := ctx.FormOptionalBool("archived")
	ctx.Data["IsArchived"] = archived

	fork := ctx.FormOptionalBool("fork")
	ctx.Data["IsFork"] = fork

	mirror := ctx.FormOptionalBool("mirror")
	ctx.Data["IsMirror"] = mirror

	template := ctx.FormOptionalBool("template")
	ctx.Data["IsTemplate"] = template

	private := ctx.FormOptionalBool("private")
	ctx.Data["IsPrivate"] = private

	services, err = services_model.FindUnreferencedServices(ctx)
	if err != nil {
		ctx.ServerError("SearchService", err)
		return
	}

	ctx.Data["Name"] = keyword
	ctx.Data["Total"] = count
	ctx.Data["Services"] = services
	ctx.Data["IsServiceIndexerEnabled"] = false

	pager := context.NewPagination(int(count), opts.PageSize, page, 5)
	pager.AddParamFromRequest(ctx.Req)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, opts.TplName)
}

// Services render explore services page
func Services(ctx *context.Context) {
	ctx.Data["UsersPageIsDisabled"] = setting.Service.Explore.DisableUsersPage
	ctx.Data["OrganizationsPageIsDisabled"] = setting.Service.Explore.DisableOrganizationsPage
	ctx.Data["CodePageIsDisabled"] = setting.Service.Explore.DisableCodePage
	ctx.Data["Title"] = ctx.Tr("explore")
	ctx.Data["PageIsExplore"] = true
	ctx.Data["PageIsExploreServices"] = true
	ctx.Data["IsServiceIndexerEnabled"] = false

	var ownerID int64
	if ctx.Doer != nil && !ctx.Doer.IsAdmin {
		ownerID = ctx.Doer.ID
	}

	onlyShowRelevant := false

	_ = ctx.Req.ParseForm() // parse the form first, to prepare the ctx.Req.Form field
	if len(ctx.Req.Form[relevantServicesOnlyParam]) != 0 {
		onlyShowRelevant = ctx.FormBool(relevantServicesOnlyParam)
	}

	RenderServiceSearch(ctx, &ServiceSearchOptions{
		PageSize:         setting.UI.ExplorePagingNum,
		OwnerID:          ownerID,
		Private:          ctx.Doer != nil,
		TplName:          tplExploreServices,
		OnlyShowRelevant: onlyShowRelevant,
	})
}
