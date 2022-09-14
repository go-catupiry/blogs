package blog

import (
	"bytes"
	"net/http"

	"github.com/go-catupiry/catu"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type BlogJSONResponse struct {
	catu.BaseListReponse
	Records []*BlogModel `json:"blog"`
}

type BlogCountJSONResponse struct {
	catu.BaseMetaResponse
}

type BlogFindOneJSONResponse struct {
	Blog *BlogModel `json:"blog"`
}

type BlogBodyRequest struct {
	Blog *BlogModel `json:"blog"`
}

type BlogTeaserTPL struct {
	Ctx    *catu.RequestContext
	Record *BlogModel
}

type LatestNewsTPL struct {
	Ctx *catu.RequestContext
	// HasLatestNews bool
	// LatestNews    []*content.Content
}

// Http blog controller | struct with http handlers
type BlogController struct {
	App catu.App
}

func (ctl *BlogController) Query(c echo.Context) error {
	var err error
	ctx := c.(*catu.RequestContext)

	var count int64
	var records []*BlogModel
	err = BlogQueryAndCountReq(&BlogQueryOpts{
		Records: &records,
		Count:   &count,
		Limit:   ctx.GetLimit(),
		Offset:  ctx.GetOffset(),
		C:       c,
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Debug("BlogFindAll Error on find contents")
	}

	ctx.Pager.Count = count

	logrus.WithFields(logrus.Fields{
		"count":             count,
		"len_records_found": len(records),
	}).Debug("BlogFindAll count result")

	for i := range records {
		records[i].LoadData()
	}

	resp := BlogJSONResponse{
		Records: records,
	}

	resp.Meta.Count = count

	return c.JSON(200, &resp)
}

func (ctl *BlogController) Create(c echo.Context) error {
	logrus.Debug("BlogController.Create running")
	var err error
	ctx := c.(*catu.RequestContext)

	can := ctx.Can("create_blog")
	if !can {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}

	var body BlogBodyRequest

	if err := c.Bind(&body); err != nil {
		if _, ok := err.(*echo.HTTPError); ok {
			return err
		}
		return c.NoContent(http.StatusNotFound)
	}

	record := body.Blog
	record.ID = 0

	if err := c.Validate(record); err != nil {
		if _, ok := err.(*echo.HTTPError); ok {
			return err
		}
		return err
	}

	logrus.WithFields(logrus.Fields{
		"body": body,
	}).Info("BlogController.Create params")

	err = record.Save()
	if err != nil {
		return err
	}

	err = record.LoadData()
	if err != nil {
		return err
	}

	resp := BlogFindOneJSONResponse{
		Blog: record,
	}

	return c.JSON(http.StatusCreated, &resp)
}

func (ctl *BlogController) Count(c echo.Context) error {
	var err error
	ctx := c.(*catu.RequestContext)

	var count int64
	err = BlogCountReq(&BlogQueryOpts{
		Count:  &count,
		Limit:  ctx.GetLimit(),
		Offset: ctx.GetOffset(),
		C:      c,
	})

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Debug("BlogFindAll Error on find contents")
	}

	ctx.Pager.Count = count

	resp := BlogCountJSONResponse{}
	resp.Count = count

	return c.JSON(200, &resp)
}

func (ctl *BlogController) FindOne(c echo.Context) error {
	id := c.Param("id")

	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Debug("BlogFindOne id from params")

	var record BlogModel
	err := BlogFindOne(id, &record)
	if err != nil {
		return err
	}

	if record.ID == 0 {
		logrus.WithFields(logrus.Fields{
			"id": id,
		}).Debug("FindOneHandler id record not found")

		return echo.NotFoundHandler(c)
	}

	record.LoadData()

	resp := BlogFindOneJSONResponse{
		Blog: &record,
	}

	return c.JSON(200, &resp)
}

func (ctl *BlogController) Update(c echo.Context) error {
	var err error

	id := c.Param("id")

	RequestContext := c.(*catu.RequestContext)

	logrus.WithFields(logrus.Fields{
		"id":    id,
		"roles": RequestContext.GetAuthenticatedRoles(),
	}).Debug("blog.Update id from params")

	var record BlogModel
	err = BlogFindOne(id, &record)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"id":    id,
			"error": err,
		}).Debug("blog.Update error on find one")
		return errors.Wrap(err, "blog.Update error on find one")
	}

	can := RequestContext.Can("update_blog")
	if !can {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}

	record.LoadData()

	body := BlogFindOneJSONResponse{Blog: &record}

	if err := c.Bind(&body); err != nil {
		logrus.WithFields(logrus.Fields{
			"id":    id,
			"error": err,
		}).Debug("content.Update error on bind")

		if _, ok := err.(*echo.HTTPError); ok {
			return err
		}
		return c.NoContent(http.StatusNotFound)
	}

	err = record.Save()
	if err != nil {
		return err
	}
	resp := BlogFindOneJSONResponse{
		Blog: &record,
	}

	return c.JSON(http.StatusOK, &resp)
}

func (ctl *BlogController) Delete(c echo.Context) error {
	var err error

	id := c.Param("id")

	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Debug("blog.DeleteOneHandler id from params")

	RequestContext := c.(*catu.RequestContext)

	var record BlogModel
	err = BlogFindOne(id, &record)
	if err != nil {
		return err
	}

	if record.ID == 0 {
		return c.JSON(http.StatusNotFound, make(map[string]string))
	}

	can := RequestContext.Can("update_blog")
	if !can {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}

	err = record.Delete()
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (ctl *BlogController) FindAllPageHandler(c echo.Context) error {
	var err error
	RequestContext := c.(*catu.RequestContext)

	switch RequestContext.GetResponseContentType() {
	case "application/json":
		return ctl.FindOne(c)
	}

	RequestContext.Title = "Blogs"
	RequestContext.MetaTags.Title = "Blogs do Monitor do Mercado"

	var count int64
	var records []*BlogModel
	err = BlogQueryAndCountReq(&BlogQueryOpts{
		Records: &records,
		Count:   &count,
		Limit:   RequestContext.GetLimit(),
		Offset:  RequestContext.GetOffset(),
		C:       c,
		IsHTML:  true,
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Debug("BlogFindAll Error on find contents")
	}

	RequestContext.Pager.Count = count

	var teaserList []string

	logrus.WithFields(logrus.Fields{
		"count":             count,
		"len_records_found": len(records),
	}).Debug("BlogFindAll count result")

	for i := range records {
		records[i].LoadTeaserData()

		var teaserHTML bytes.Buffer

		err = RequestContext.RenderTemplate(&teaserHTML, "blog/teaser", BlogTeaserTPL{
			Ctx:    RequestContext,
			Record: records[i],
		})
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("blog.BlogFindAll error on render teaser")
		} else {
			teaserList = append(teaserList, teaserHTML.String())
		}
	}

	RequestContext.Set("hasRecords", true)
	RequestContext.Set("records", teaserList)

	RequestContext.Set("RequestPath", RequestContext.Request().URL.String())

	// err = content.LoadMostReadBlockData(RequestContext)
	// if err != nil {
	// 	logrus.WithFields(logrus.Fields{
	// 		"err": err.Error(),
	// 	}).Error("FindAllPageHandler error on render sidebar block")
	// }

	return c.Render(http.StatusOK, "blog/findAll", &catu.TemplateCTX{
		Ctx: RequestContext,
	})
}

func (ctl *BlogController) FindOnePageHandler(c echo.Context) error {
	var err error
	ctx := c.(*catu.RequestContext)

	switch ctx.GetResponseContentType() {
	case "application/json":
		return ctl.FindOne(c)
	}
	// id or urlUniquePath
	id := c.Param("id")

	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Debug("FindOnePageHandler id from params")

	var record BlogModel
	err = BlogFindOne(id, &record)
	if err != nil {
		return err
	}

	if record.ID == 0 {
		logrus.WithFields(logrus.Fields{
			"id": id,
		}).Debug("FindOnePageHandler id record not found")
		return echo.NotFoundHandler(c)
	}

	record.LoadData()

	ctx.Title = record.Title
	ctx.BodyClass = append(ctx.BodyClass, "body-content-findOne")

	ctx.MetaTags.Title = record.Title
	ctx.MetaTags.Description = record.Description
	if record.HasLogo {
		ctx.MetaTags.ImageURL = record.Logo[0].URLs["medium"]
	}

	err = LoadPageBlogPosts(ctx)
	if err != nil {
		return err
	}

	// err = content.LoadMostReadBlockData(ctx)
	// if err != nil {
	// 	logrus.WithFields(logrus.Fields{
	// 		"err": err.Error(),
	// 	}).Error("FindOnePageHandler error on render sidebar block")
	// }

	return c.Render(http.StatusOK, "blog/findOne", &catu.TemplateCTX{
		Ctx:    ctx,
		Record: &record,
	})
}

type BlogControllerCfg struct {
	App catu.App
}

func NewBlogController(cfg *BlogControllerCfg) *BlogController {
	ctx := BlogController{App: cfg.App}

	return &ctx
}

func LoadPageBlogPosts(ctx *catu.RequestContext) error {
	var err error
	var count int64
	var records []*BlogPostModel
	err = BlogPostQueryAndCountReq(&BlogPostQueryOpts{
		Records: &records,
		Count:   &count,
		Limit:   ctx.GetLimit(),
		Offset:  ctx.GetOffset(),
		C:       ctx.EchoContext,
		IsHTML:  true,
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Debug("BlogController.FindOnePageHandler Error on find records")
	}

	ctx.Pager.Count = count

	var teaserList []string

	logrus.WithFields(logrus.Fields{
		"count":             count,
		"len_records_found": len(records),
	}).Debug("BlogPostController.FindAllPageHandler count result")

	for i := range records {
		records[i].LoadTeaserData()

		var teaserHTML bytes.Buffer

		err = ctx.RenderTemplate(&teaserHTML, "blog-post/teaser", BlogPostTeaserTPL{
			Ctx:    ctx,
			Record: records[i],
		})
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("BlogPostController.FindAllPageHandler error on render teaser")
		} else {
			teaserList = append(teaserList, teaserHTML.String())
		}
	}

	if len(records) == 0 {
		ctx.Set("hasRecords", false)
	} else {
		ctx.Set("hasRecords", true)
	}

	ctx.Set("records", teaserList)

	return nil
}
