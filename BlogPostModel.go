package blog

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"
	"time"

	"github.com/go-catupiry/catu"
	"github.com/go-catupiry/catu/helpers"
	"github.com/go-catupiry/files"
	"github.com/go-catupiry/tags"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type blogPostTeaserTPL struct {
	Ctx    *catu.RequestContext
	Record *BlogPostModel
}

var featuredImageFieldCfg files.FieldConfigurationInterface

type BlogPostModel struct {
	ID            uint64     `gorm:"primaryKey;column:id;type:int(11);not null" json:"id" filter:"param:id;type:number"`
	Title         string     `gorm:"column:title;type:varchar(255);not null" json:"title" filter:"param:title;type:string"`
	Teaser        string     `gorm:"column:teaser;type:text" json:"teaser" filter:"param:teaser;type:string"`
	Body          string     `gorm:"column:body;type:text" json:"body" filter:"param:body;type:string"`
	Published     bool       `gorm:"column:published;type:tinyint(1);default:0" json:"published"`
	PublishedAt   *time.Time `gorm:"column:publishedAt;type:datetime" json:"publishedAt"`
	Highlighted   uint       `gorm:"column:highlighted;type:int(11);not null;default:0" json:"highlighted" filter:"param:highlighted;type:number"`
	AllowComments bool       `gorm:"column:allowComments;type:tinyint(1);default:1" json:"allowComments"`
	URLPath       string     `gorm:"column:urlPath;type:varchar(255);not null" json:"urlPath" filter:"param:urlPath;type:string"`
	CreatedAt     time.Time  `gorm:"column:createdAt;type:datetime;not null" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"column:updatedAt;type:datetime;not null" json:"updatedAt"`
	CreatorID     *uint      `gorm:"index:creatorId;column:creatorId;type:int(11)" json:"creatorId,string"`
	// Users         Users     `gorm:"joinForeignKey:creatorId;foreignKey:id" json:"usersList"` // We.js users table
	BlogID *uint64    `gorm:"index:blogId;column:blogId;" json:"blogId" filter:"param:blogId;type:string"`
	Blog   *BlogModel `gorm:"foreignKey:BlogID;references:ID;" json:"blog"`

	InRSS bool `gorm:"column:inRSS;type:tinyint(1);default:0" json:"inRSS"`

	HasFeaturedImage bool                `gorm:"-"`
	FeaturedImage    []*files.ImageModel `gorm:"-" json:"featuredImage"`

	TagsRecords []tags.TermModel `gorm:"-" json:"-"`
	Tags        []string         `gorm:"-" json:"tags"`

	LinkPermanent string `gorm:"-" json:"linkPermanent"`

	ShowInLists bool `gorm:"column:show_in_lists;" json:"showInLists" filter:"param:showInLists;type:bool"`
}

// TableName get sql table name
func (m *BlogPostModel) TableName() string {
	return "blog_posts"
}

func (r *BlogPostModel) LoadTeaserData() error {
	r.LoadPath()
	return nil
}

func (r *BlogPostModel) LoadData() error {
	r.LoadPath()
	return nil
}

// TableName get id in string type
func (m *BlogPostModel) GetIDString() string {
	return strconv.FormatInt(int64(m.ID), 10)
}

func (r *BlogPostModel) GetPath() string {
	path := ""

	if r.ID != 0 {
		path += "/blog-post/" + strconv.FormatUint(r.ID, 10)
	}

	return path
}

func (r *BlogPostModel) LoadPath() error {
	app := catu.GetApp()
	r.LinkPermanent = app.GetConfiguration().Get("APP_ORIGIN") + r.GetPath()
	return nil
}

// Save - Create if is new or update
func (m *BlogPostModel) Save() error {
	var err error
	db := catu.GetDefaultDatabaseConnection()

	m.RefreshSlug()

	if m.ID == 0 {
		// create ....
		err = db.Create(&m).Error
		if err != nil {
			return err
		}
	} else {
		// update ...
		err = db.Save(&m).Error
		if err != nil {
			return err
		}
	}

	err = files.UpdateFieldImagesByObjects(m.GetIDString(), m.FeaturedImage, featuredImageFieldCfg)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
			"id":  m.ID,
		}).Error("BlogPostModel.Save error on update featuredImage")
	}

	return nil
}

func (m *BlogPostModel) Publish() error {
	if m.Published {
		// already published, skip
		return nil
	}

	db := catu.GetDefaultDatabaseConnection()

	m.Published = true
	now := time.Now()
	m.PublishedAt = &now

	err := db.Model(&m).Updates(map[string]interface{}{"published": m.Published, "publishedAt": m.PublishedAt}).Error
	if err != nil {
		return errors.Wrap(err, "error on publish blog posts")
	}

	return nil
}

func (m *BlogPostModel) UnPublish() error {
	if !m.Published {
		// already published, skip
		return nil
	}

	db := catu.GetDefaultDatabaseConnection()

	m.Published = false
	m.PublishedAt = nil

	err := db.Model(&m).Updates(map[string]interface{}{"published": m.Published, "publishedAt": m.PublishedAt}).Error
	if err != nil {
		return errors.Wrap(err, "error on unPublish blog posts")
	}

	return nil
}

func FindUnPublishedBlogPosts(records *[]*BlogPostModel, limit int) error {
	db := catu.GetDefaultDatabaseConnection()

	return db.
		Order("createdAt ASC").
		Where("published != ?", "1").
		Where("publishedAt <= DATE_ADD(now(), INTERVAL 1 MINUTE)").
		Limit(limit).
		Find(records).Error
}

func (r *BlogPostModel) RefreshSlug() {

}

func (r *BlogPostModel) Delete() error {
	db := catu.GetDefaultDatabaseConnection()
	return db.Unscoped().Delete(&r).Error
}

func PublishSchenduledBlogPosts(app catu.App) error {
	posts := []*BlogPostModel{}
	err := FindUnPublishedBlogPosts(&posts, 25)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": fmt.Sprintf("%+v\n", err),
		}).Error("PublishSchenduledBlogPosts error on publish blogs")
	}

	logrus.WithFields(logrus.Fields{
		"count": len(posts),
	}).Debug("PublishSchenduledBlogPosts count to publish")

	for _, c := range posts {
		err := c.Publish()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": fmt.Sprintf("%+v\n", err),
			}).Error("PublishSchenduledBlogPosts error on publish blog posts record")
		}

		logrus.WithFields(logrus.Fields{
			"id": c.ID,
		}).Info("PublishSchenduledBlogPosts blog posts published")
	}

	return nil
}

func (r *BlogPostModel) GetTeaserDatesHTML(separator string) template.HTML {
	if r.CreatedAt.IsZero() {
		return template.HTML("")
	}

	html := ""
	html += RenderCreatedAtHTML(r.CreatedAt, false)

	return template.HTML(html)
}

// FindOne - Find one blog post record
func BlogPostFindOne(id string, record *BlogPostModel) error {
	db := catu.GetDefaultDatabaseConnection()

	return db.
		Where("id = ? OR URLPath = ?", id, id).
		First(&record).Error
}

func (r *BlogPostModel) LoadFeaturedImage() error {
	var err error

	logrus.WithFields(logrus.Fields{
		"id": r.ID,
	}).Debug("BlogPostModel.LoadFeaturedImage will refresh")

	featuredImage, err := files.GetImagesInField("BlogPostModel", "featuredImage", strconv.FormatUint(r.ID, 10), 1)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"id":         r.ID,
			"error":      err,
			"images_len": len(featuredImage),
		}).Error("BlogPostModel.LoadFeaturedImage error on find featuredImage")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"id":         r.ID,
		"images_len": len(featuredImage),
	}).Debug("BlogPostModel.LoadFeaturedImage featuredImage found")

	if len(featuredImage) > 0 {
		r.FeaturedImage = featuredImage

	}

	return nil
}

func (r *BlogPostModel) RenderHomeBlogPostBlock(ctx *catu.RequestContext) (bytes.Buffer, error) {
	var err error
	var teaserHTML bytes.Buffer

	err = ctx.RenderTemplate(&teaserHTML, "blog/home-blogs-block", blogPostTeaserTPL{
		Ctx:    ctx,
		Record: r,
	})

	return teaserHTML, err
}

type BlogPostQueryOpts struct {
	BlogID  int64
	Records *[]*BlogPostModel
	Count   *int64
	Limit   int
	Offset  int
	C       echo.Context
	IsHTML  bool
}

func BlogPostQueryAndCountReq(opts *BlogPostQueryOpts) error {
	db := catu.GetDefaultDatabaseConnection()

	c := opts.C
	ctx := c.(*catu.RequestContext)

	q := c.QueryParam("q")
	showInLists := c.QueryParam("showInLists")
	blogId := c.QueryParam("blogId")

	query := db

	canAccessUnpublished := ctx.Can("access_blogs_unpublished")
	if !canAccessUnpublished {
		p := c.QueryParam("published")
		if p != "" {
			logrus.WithFields(logrus.Fields{
				"queryParams.published": p,
			}).Warn("BlogQueryAndCountReq forbidden published query param")
			return nil
		}
	}

	queryI, err := ctx.Query.SetDatabaseQueryForModel(query, &BlogPostModel{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": fmt.Sprintf("%+v\n", err),
		}).Error("BlogQueryAndCountReq error")
	}
	query = queryI.(*gorm.DB)

	if q != "" {
		query = query.Where(
			db.Where("title LIKE ?", "%"+q+"%").Or(db.Where("body LIKE ?", "%"+q+"%")),
		)
	}

	if showInLists == "" {
		// default:
		if opts.IsHTML {
			query = query.Where("show_in_lists = ?", "1")
		}
	} else if showInLists == "false" || showInLists == "0" {
		query = query.Where("show_in_lists = ?", "0")
	} else if showInLists == "all" {
		// skip ... ignore that filter
	} else {
		query = query.Where("show_in_lists = ?", "1")
	}

	if !canAccessUnpublished {
		query = query.Where("published = ?", "1")
	}

	if blogId != "" {
		query = query.Where("blogId = ?", blogId)
	}

	orderColumn, orderIsDesc, orderValid := helpers.ParseUrlQueryOrder(c.QueryParam("order"))

	if orderValid {
		query = query.Order(clause.OrderByColumn{
			Column: clause.Column{Table: clause.CurrentTable, Name: orderColumn},
			Desc:   orderIsDesc,
		})
	} else {
		query = query.Order("highlighted DESC").
			Order("publishedAt DESC").
			Order("id DESC")
	}

	query = query.Limit(opts.Limit).
		Offset(opts.Offset)

	err = query.Find(opts.Records).Error
	if err != nil {
		return err
	}

	return BlogPostCountReq(opts)
}

func BlogPostCountReq(opts *BlogPostQueryOpts) error {
	db := catu.GetDefaultDatabaseConnection()

	c := opts.C

	q := c.QueryParam("q")
	// showInLists := c.QueryParam("showInLists")

	ctx := c.(*catu.RequestContext)

	// Count ...
	queryCount := db

	if q != "" {
		queryCount = queryCount.Or(
			db.Where("title LIKE ?", "%"+q+"%"),
			db.Where("body LIKE ?", "%"+q+"%"),
		)
	}

	queryICount, err := ctx.Query.SetDatabaseQueryForModel(queryCount, &BlogModel{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": fmt.Sprintf("%+v\n", err),
		}).Error("BlogQueryAndCountReq count error")
	}
	queryCount = queryICount.(*gorm.DB)

	return queryCount.
		Table("blog_posts").
		Count(opts.Count).Error
}

func BlogPostCountQuery(count *int64) error {
	db := catu.GetDefaultDatabaseConnection()

	var records BlogModel
	return db.
		// Select([]string{"ult", "date"}).
		Order("createdAt desc").
		Find(&records).
		Count(count).Error
}

func NewBlogPostModel() *BlogPostModel {
	return &BlogPostModel{

		ShowInLists: true,
	}
}

// Find many blog post records
func BlogPostFindLatest(records *[]*BlogPostModel, limit int) error {
	db := catu.GetDefaultDatabaseConnection()

	return db.
		// Select([]string{"ult", "date"}).
		Omit("Editors").
		Order("createdAt desc").
		Limit(limit).
		Find(records).Error
}
