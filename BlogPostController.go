package blog

import (
	"bytes"
	"net/http"

	"github.com/go-catupiry/catu"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type BlogPostJSONResponse struct {
	catu.BaseListReponse
	Records *[]*BlogPostModel `json:"blog-post"`
}

type BlogPostCountJSONResponse struct {
	catu.BaseMetaResponse
}

type BlogPostFindOneJSONResponse struct {
	Record *BlogPostModel `json:"blog-post"`
}

type BlogPostBodyRequest struct {
	Record *BlogPostModel `json:"blog-post"`
}

type BlogPostTeaserTPL struct {
	Ctx    *catu.RequestContext
	Record *BlogPostModel
}

// Http blog post controller | struct with http handlers
type BlogPostController struct {
	App catu.App
}

type PostTemplateCTX struct {
	EchoContext echo.Context
	Ctx         *catu.RequestContext
	Record      *BlogPostModel
	Records     []*BlogPostModel
	Blog        *BlogModel
}

func (ctl *BlogPostController) Query(c echo.Context) error {
	var err error
	ctx := c.(*catu.RequestContext)

	var count int64
	var records []*BlogPostModel
	err = BlogPostQueryAndCountReq(&BlogPostQueryOpts{
		Records: &records,
		Count:   &count,
		Limit:   ctx.GetLimit(),
		Offset:  ctx.GetOffset(),
		C:       c,
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Debug("BlogPostFindAll Error on find contents")
	}

	ctx.Pager.Count = count

	logrus.WithFields(logrus.Fields{
		"count":             count,
		"len_records_found": len(records),
	}).Debug("BlogPostFindAll count result")

	for i := range records {
		records[i].LoadData()
	}

	resp := BlogPostJSONResponse{
		Records: &records,
	}

	resp.Meta.Count = count

	return c.JSON(200, &resp)
}

func (ctl *BlogPostController) Create(c echo.Context) error {
	logrus.Debug("BlogPostController.Create running")
	var err error
	ctx := c.(*catu.RequestContext)

	can := ctx.Can("create_blog-post")
	if !can {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}

	var body BlogPostBodyRequest

	if err := c.Bind(&body); err != nil {
		if _, ok := err.(*echo.HTTPError); ok {
			return err
		}
		return c.NoContent(http.StatusNotFound)
	}

	record := body.Record
	record.ID = 0

	if err := c.Validate(record); err != nil {
		if _, ok := err.(*echo.HTTPError); ok {
			return err
		}
		return err
	}

	logrus.WithFields(logrus.Fields{
		"body": body,
	}).Info("BlogPostController.Create params")

	err = record.Save()
	if err != nil {
		return err
	}

	err = record.LoadData()
	if err != nil {
		return err
	}

	resp := BlogPostFindOneJSONResponse{
		Record: record,
	}

	return c.JSON(http.StatusCreated, &resp)
}

func (ctl *BlogPostController) Count(c echo.Context) error {
	var err error
	ctx := c.(*catu.RequestContext)

	var count int64
	err = BlogPostCountReq(&BlogPostQueryOpts{
		Count:  &count,
		Limit:  ctx.GetLimit(),
		Offset: ctx.GetOffset(),
		C:      c,
	})

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Debug("BlogPostCount Error on find blog posts")
	}

	ctx.Pager.Count = count

	resp := BlogCountJSONResponse{}
	resp.Count = count

	return c.JSON(200, &resp)
}

func (ctl *BlogPostController) FindOne(c echo.Context) error {
	id := c.Param("id")
	ctx := c.(*catu.RequestContext)

	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Debug("BlogPostFindOne id from params")

	var record BlogPostModel
	err := BlogPostFindOne(id, &record)
	if err != nil {
		return err
	}

	if record.ID == 0 {
		logrus.WithFields(logrus.Fields{
			"id": id,
		}).Debug("BlogPostFindOne id record not found")

		return echo.NotFoundHandler(c)
	}

	can := ctx.Can("find_blog-post")
	if !can {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}

	if !record.Published {
		canAccessUnpublished := ctx.Can("access_contents_unpublished")
		if !canAccessUnpublished {
			return &catu.HTTPError{
				Code:     403,
				Message:  "Forbidden",
				Internal: errors.New("forbidden"),
			}
		}
	}

	record.LoadData()

	resp := BlogPostFindOneJSONResponse{
		Record: &record,
	}

	return c.JSON(200, &resp)
}

func (ctl *BlogPostController) Update(c echo.Context) error {
	var err error

	id := c.Param("id")

	RequestContext := c.(*catu.RequestContext)

	logrus.WithFields(logrus.Fields{
		"id":    id,
		"roles": RequestContext.GetAuthenticatedRoles(),
	}).Debug("BlogPostController.Update id from params")

	var record BlogPostModel
	err = BlogPostFindOne(id, &record)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"id":    id,
			"error": err,
		}).Debug("BlogPostController.Update error on find one")
		return errors.Wrap(err, "BlogPostController.Update error on find one")
	}

	can := RequestContext.Can("update_blog-post")
	if !can {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}

	record.LoadData()

	body := BlogPostFindOneJSONResponse{Record: &record}

	if err := c.Bind(&body); err != nil {
		logrus.WithFields(logrus.Fields{
			"id":    id,
			"error": err,
		}).Debug("BlogPostController.Update error on bind")

		if _, ok := err.(*echo.HTTPError); ok {
			return err
		}
		return c.NoContent(http.StatusNotFound)
	}

	err = record.Save()
	if err != nil {
		return err
	}
	resp := BlogPostFindOneJSONResponse{
		Record: &record,
	}

	return c.JSON(http.StatusOK, &resp)
}

func (ctl *BlogPostController) Delete(c echo.Context) error {
	var err error

	id := c.Param("id")

	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Debug("BlogPostController.Delete id from params")

	RequestContext := c.(*catu.RequestContext)

	var record BlogPostModel
	err = BlogPostFindOne(id, &record)
	if err != nil {
		return err
	}

	if record.ID == 0 {
		return c.JSON(http.StatusNotFound, make(map[string]string))
	}

	can := RequestContext.Can("delete_blog-post")
	if !can {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}

	err = record.Delete()
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (ctl *BlogPostController) FindAllPageHandler(c echo.Context) error {
	var err error
	ctx := c.(*catu.RequestContext)

	switch ctx.GetResponseContentType() {
	case "application/json":
		return ctl.FindOne(c)
	}

	err = loadCtxBlog(ctx)
	if err != nil {
		return err
	}

	ctx.Title = "Blogs"
	ctx.MetaTags.Title = "Blogs do Monitor do Mercado"

	var count int64
	var records []*BlogPostModel
	err = BlogPostQueryAndCountReq(&BlogPostQueryOpts{
		Records: &records,
		Count:   &count,
		Limit:   ctx.GetLimit(),
		Offset:  ctx.GetOffset(),
		C:       c,
		IsHTML:  true,
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Debug("BlogPostController.FindAllPageHandler Error on find contents")
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
	ctx.Set("RequestPath", ctx.Request().URL.String())

	// err = content.LoadMostReadBlockData(ctx)
	// if err != nil {
	// 	logrus.WithFields(logrus.Fields{
	// 		"err": err.Error(),
	// 	}).Error("BlogPostController.FindAllPageHandler error on render sidebar block")
	// }

	return c.Render(http.StatusOK, "blog-post/findAll", &catu.TemplateCTX{
		Ctx: ctx,
	})
}

func (ctl *BlogPostController) FindOnePageHandler(c echo.Context) error {
	var err error
	ctx := c.(*catu.RequestContext)

	switch ctx.GetResponseContentType() {
	case "application/json":
		return ctl.FindOne(c)
	}
	// id or urlUniquePath
	id := c.Param("blogPostId")

	err = loadCtxBlog(ctx)
	if err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Debug("BlogPostController.FindOnePageHandler id from params")

	var record BlogPostModel
	err = BlogPostFindOne(id, &record)
	if err != nil {
		return err
	}

	if record.ID == 0 {
		logrus.WithFields(logrus.Fields{
			"id": id,
		}).Debug("BlogPostController.FindOnePageHandler id record not found")
		return echo.NotFoundHandler(c)
	}

	if !record.Published {
		canAccessUnpublished := ctx.Can("access_contents_unpublished")
		if !canAccessUnpublished {
			return &catu.HTTPError{
				Code:     403,
				Message:  "Forbidden",
				Internal: errors.New("forbidden"),
			}
		}
	}

	record.LoadData()

	ctx.Title = record.Title
	ctx.BodyClass = append(ctx.BodyClass, "body-blog-post-findOne")

	ctx.MetaTags.Title = record.Title
	ctx.MetaTags.Description = record.Teaser
	if record.HasFeaturedImage {
		ctx.MetaTags.ImageURL = record.FeaturedImage[0].URLs["medium"]
	}

	// err = content.LoadMostReadBlockData(ctx)
	// if err != nil {
	// 	logrus.WithFields(logrus.Fields{
	// 		"err": err.Error(),
	// 	}).Error("BlogPostController.FindOnePageHandler error on render sidebar block")
	// }

	return ctx.Render(http.StatusOK, "blog-post/findOne", &catu.TemplateCTX{
		Ctx:    ctx,
		Record: &record,
	})
}

func loadCtxBlog(ctx *catu.RequestContext) error {
	var err error
	blogId := ctx.Param("blogId")

	var blog BlogModel

	if blogId != "" {
		err = BlogFindOne(blogId, &blog)
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return errors.Wrap(err, "error on find blog") // unknow error
			}
			return &catu.HTTPError{
				Code:     404,
				Message:  "blog not found",
				Internal: err,
			}
		}

		err := blog.LoadTeaserData()
		if err != nil {
			return errors.Wrap(err, "error on load blog teaser")
		}

		ctx.Set("blog", blog)
	}

	return nil
}

type BlogPostControllerCfg struct {
	App catu.App
}

func NewBlogPostController(cfg *BlogPostControllerCfg) *BlogPostController {
	ctx := BlogPostController{App: cfg.App}

	return &ctx
}
