package blog

import (
	"github.com/go-catupiry/catu"
	"github.com/go-catupiry/tags"
	"github.com/gookit/event"
	"github.com/sirupsen/logrus"
)

type BlogPlugin struct {
	catu.Pluginer
	Name               string
	BlogController     *BlogController
	BlogPostController *BlogPostController
}

func (r *BlogPlugin) GetName() string {
	return r.Name
}

func (r *BlogPlugin) Init(app catu.App) error {
	logrus.Debug(r.GetName() + " Init")

	r.BlogController = NewBlogController(&BlogControllerCfg{App: app})
	r.BlogPostController = NewBlogPostController(&BlogPostControllerCfg{App: app})

	app.GetEvents().On("bindRoutes", event.ListenerFunc(func(e event.Event) error {
		return r.BindRoutes(app)
	}), event.Normal)

	app.GetEvents().On("bootstrap", event.ListenerFunc(func(e event.Event) error {
		return r.Bootstrap(app)
	}), event.Normal)

	app.GetEvents().On("cron-job", event.ListenerFunc(func(e event.Event) error {
		return PublishSchenduledBlogPosts(app)
	}), event.Normal)

	return nil
}

func (r *BlogPlugin) BindRoutes(app catu.App) error {
	logrus.Debug(r.GetName() + " BindRoutes")

	blogCTL := r.BlogController
	blogPostCTL := r.BlogPostController

	router := app.SetRouterGroup("blogs", "/blogs")
	router.GET("", blogCTL.FindAllPageHandler)
	router.GET("/:blogId", blogPostCTL.FindAllPageHandler)
	router.GET("/:blogId/:blogPostId", blogPostCTL.FindOnePageHandler)

	routerApi := app.SetRouterGroup("blog-api", "/api/blog")
	app.SetResource("blog", blogCTL, routerApi)

	routerPostApi := app.SetRouterGroup("blog-post-api", "/api/blog-post")
	app.SetResource("blog-post", blogPostCTL, routerPostApi)

	return nil
}

func (r *BlogPlugin) Bootstrap(app catu.App) error {
	tagsFieldCfg = tags.NewTagFieldConfiguration("Tags", "blog", "tags")

	db := app.GetDB()

	err := db.SetupJoinTable(&BlogModel{}, "Editors", &BlogEditorsModel{})
	if err != nil {
		return err
	}

	return nil
}

type PluginCfgs struct{}

func NewPlugin(cfg *PluginCfgs) *BlogPlugin {
	p := BlogPlugin{Name: "blog"}
	return &p
}
