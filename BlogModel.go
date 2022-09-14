package blog

import (
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"time"

	"github.com/go-catupiry/catu"
	"github.com/go-catupiry/catu/helpers"
	"github.com/go-catupiry/drouter"
	"github.com/go-catupiry/files"
	"github.com/go-catupiry/tags"
	"github.com/go-catupiry/user"
	"github.com/gosimple/slug"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var tagsFieldCfg tags.FieldConfigurationInterface
var logoFieldCfg files.FieldConfigurationInterface

// BlogModel Stores blog record data
type BlogModel struct {
	ID               uint64    `gorm:"primaryKey;column:id;type:int(11);not null" json:"id" filter:"param:id;type:number"`
	Title            string    `gorm:"column:title;type:varchar(255);not null" json:"title" filter:"param:title;type:string"`
	Description      string    `gorm:"column:description;type:text" json:"description" filter:"param:decription;type:string"`
	DescriptionSmall string    `gorm:"column:description_small;type:text" json:"descriptionSmall" filter:"param:descriptionSmall;type:string"`
	URLUniquePath    string    `gorm:"unique;column:urlUniquePath;type:varchar(255);not null" json:"urlUniquePath"`
	CreatedAt        time.Time `gorm:"column:createdAt;type:datetime;not null" json:"createdAt"`
	UpdatedAt        time.Time `gorm:"column:updatedAt;type:datetime;not null" json:"updatedAt"`
	CreatorID        *int64    `gorm:"index:creatorId;column:creatorId;type:int(11)" json:"creatorId,string"`

	ShowInLists bool `gorm:"column:show_in_lists;"  json:"showInLists" filter:"param:showInLists;type:bool"`

	Tags []string `gorm:"-" json:"tags"`

	HasLogo bool                `gorm:"-"`
	Logo    []*files.ImageModel `gorm:"-" json:"logo"`

	Alias         *drouter.UrlAliasModel `gorm:"-" json:"alias"`
	SetAlias      string                 `gorm:"-" json:"setAlias"`
	LinkPermanent string                 `gorm:"-" json:"linkPermanent"`

	Editors []*user.UserModel `gorm:"many2many:blogs-editors;joinForeignKey:blog_id;references:ID;joinReferences:user_id;" json:"editors"`

	Posts []*BlogPostModel `gorm:"foreignKey:BlogID;references:ID" json:"posts"`

	// `gorm:"foreignKey:UserNumber;references:MemberNumber"`
}

// TableName get sql table name
func (m *BlogModel) TableName() string {
	return "blogs"
}

func (r *BlogModel) GetIDString() string {
	return strconv.FormatUint(r.ID, 10)
}

func (r *BlogModel) LoadTeaserData() error {
	r.RefreshTerms()
	r.LoadImages()
	r.LoadPath()
	r.LoadLatestPost(1)
	return nil
}

func (r *BlogModel) LoadData() error {
	r.RefreshTerms()
	r.LoadImages()
	r.LoadPath()
	return nil
}

func (r *BlogModel) GetPath() string {
	path := ""

	if r.ID != 0 {
		path += "/blog/" + strconv.FormatUint(r.ID, 10)
	}

	return path
}

func (r *BlogModel) LoadPath() error {
	app := catu.GetApp()
	r.LinkPermanent = app.GetConfiguration().Get("APP_ORIGIN") + r.GetPath()
	return nil
}

func (r *BlogModel) GetTeaserDatesHTML(separator string) template.HTML {
	return template.HTML("")
}

// Save - Create if is new or update
func (m *BlogModel) Save() error {
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
		err = db.
			Omit("Editors").
			Save(&m).Error
		if err != nil {
			return err
		}

		db.Model(&m).Association("Editors").Replace(m.Editors)
	}

	err = files.UpdateFieldImagesByObjects(m.GetIDString(), m.Logo, logoFieldCfg)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
			"id":  m.ID,
		}).Error("BlogModel.Save error on update logo")
	}

	return nil
}

func (r *BlogModel) RefreshSlug() {
	r.URLUniquePath = helpers.TruncateString(r.URLUniquePath, 60, "")
}

func (r *BlogModel) RefreshTerms() error {
	var err error

	logrus.WithFields(logrus.Fields{
		"id": r.ID,
	}).Debug("Blog.RefreshTerms will refresh")

	// tags
	var tags []tags.TermModel
	err = tagsFieldCfg.FindManyTerm(r.GetIDString(), &tags)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logrus.WithFields(logrus.Fields{
			"id":    r.ID,
			"error": err,
		}).Error("Blog.RefreshTerms error on find tags")
		return err
	}

	for i := range tags {
		r.Tags = append(r.Tags, tags[i].Text)
	}

	logrus.WithFields(logrus.Fields{
		"id": r.ID,
	}).Debug("Blog.RefreshTerms done refresh")

	return nil
}

func (r *BlogModel) LoadImages() error {
	var err error

	logrus.WithFields(logrus.Fields{
		"id": r.ID,
	}).Debug("content.LoadImages will refresh")

	logo, err := files.GetImagesInField("blog", "logo", strconv.FormatUint(r.ID, 10), 1)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"id":         r.ID,
			"error":      err,
			"images_len": len(logo),
		}).Error("BlogModel.LoadImages error on find logo")
		return err
	}

	if len(logo) > 0 {
		logrus.WithFields(logrus.Fields{
			"id":         r.ID,
			"images_len": len(logo),
			"logo.ID":    logo[0].ID,
		}).Debug("BlogModel.LoadImages logo found")

		r.Logo = logo
		r.HasLogo = true
	} else {
		logrus.WithFields(logrus.Fields{
			"id":         r.ID,
			"images_len": len(logo),
		}).Debug("BlogModel.LoadImages logo not found")
	}

	return nil
}

func (r *BlogModel) Delete() error {
	db := catu.GetDefaultDatabaseConnection()
	return db.Unscoped().Delete(&r).Error
}

func (r *BlogModel) LoadLatestPost(limit int) error {
	db := catu.GetDefaultDatabaseConnection()

	return db.Model(&BlogPostModel{}).
		Where("published = ?", true).
		Where("blogId = ?", r.GetIDString()).
		Limit(limit).
		Order("published = 1").
		Find(&r.Posts).Error
}

// FindOne - Find one blog record
func BlogFindOne(idOrSlug string, record *BlogModel) error {
	db := catu.GetDefaultDatabaseConnection()
	return db.
		Preload("Editors").
		Where("id = ? OR urlUniquePath = ?", idOrSlug, idOrSlug).
		First(&record).Error
}

// FindLatest - Find many blog records
func BlogFindLatest(records *[]BlogModel, limit int) error {
	db := catu.GetDefaultDatabaseConnection()

	return db.
		// Select([]string{"ult", "date"}).
		Omit("Editors").
		Order("createdAt desc").
		Limit(limit).
		Find(records).Error
}

func BlogQuery(records *[]BlogModel, limit int) error {
	db := catu.GetDefaultDatabaseConnection()

	return db.
		// Select([]string{"ult", "date"}).
		// Order("publishedAt desc").
		Limit(limit).
		Find(records).Error
}

type BlogQueryOpts struct {
	Records *[]*BlogModel
	Count   *int64
	Limit   int
	Offset  int
	C       echo.Context
	IsHTML  bool
}

func BlogQueryAndCountReq(opts *BlogQueryOpts) error {
	db := catu.GetDefaultDatabaseConnection()
	c := opts.C
	ctx := c.(*catu.RequestContext)

	q := c.QueryParam("q")

	query := db

	queryI, err := ctx.Query.SetDatabaseQueryForModel(query, &BlogModel{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": fmt.Sprintf("%+v\n", err),
		}).Error("ContentQueryAndCountReq error")
	}
	query = queryI.(*gorm.DB)

	if q != "" {
		query = query.Where(
			db.Where("title LIKE ?", "%"+q+"%").Or(db.Where("description LIKE ?", "%"+q+"%")),
		)
	}

	// query = query.Where("published = ?", "1")

	orderColumn, orderIsDesc, orderValid := helpers.ParseUrlQueryOrder(c.QueryParam("order"))

	if orderValid {
		query = query.Order(clause.OrderByColumn{
			Column: clause.Column{Table: clause.CurrentTable, Name: orderColumn},
			Desc:   orderIsDesc,
		})
	} else {
		query = query.Order("createdAt DESC").
			Order("id DESC")
	}

	query = query.Limit(opts.Limit).
		Offset(opts.Offset)

	query = query.Preload("Editors")

	err = query.Find(opts.Records).Error
	if err != nil {
		return err
	}

	// Count ...
	queryCount := db
	if q != "" {
		queryCount = queryCount.Or(
			db.Where("title LIKE ?", "%"+q+"%"),
			db.Where("description LIKE ?", "%"+q+"%"),
		)
	}

	return queryCount.
		Table("blogs").
		Count(opts.Count).Error
}

func BlogCountReq(opts *BlogQueryOpts) error {
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
			db.Where("description LIKE ?", "%"+q+"%"),
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
		Table("blogs").
		Count(opts.Count).Error
}

func CountQuery(count *int64) error {
	db := catu.GetDefaultDatabaseConnection()

	var records BlogModel
	return db.
		// Select([]string{"ult", "date"}).
		Order("createdAt desc").
		Find(&records).
		Count(count).Error
}

func RenderCreatedAtHTML(date time.Time, isHighlighted bool) string {
	hiClass := getHiClass(isHighlighted)

	return `<span ` + hiClass + ` data-toggle="tooltip" title="Data de criação, conteúdo despublicado">
      <i class="fa fa-square-o" aria-hidden="true"></i> ` + helpers.FormatDate(&date, "02/01/2006 15:04") + `
    </span>`
}

func getHiClass(isHighlighted bool) string {
	if isHighlighted {
		return `class="text-dark"`
	}

	return ""
}

func (r *BlogModel) UrlAliasUpsert() error {
	alias := ""
	idString := strconv.FormatUint(r.ID, 10)
	slug.MaxLength = 45

	if r.SetAlias != "" {
		alias = r.SetAlias

	} else {

		alias = "/blogs/" + r.URLUniquePath
	}

	target := "/blogs/" + idString

	var aliasRecord drouter.UrlAliasModel
	err := drouter.URLAliasUpsert(alias, target, "", &aliasRecord)
	if err != nil {
		return err
	}
	r.Alias = &aliasRecord

	return nil
}

func NewBlogModel() *BlogModel {
	return &BlogModel{
		// Published:   false,
		ShowInLists: true,
	}
}
